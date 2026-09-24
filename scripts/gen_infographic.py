#!/usr/bin/env python3
"""
gen_infographic.py - Generate hand-drawn infographics via OpenAI-compatible image API.

Usage:
    python scripts/gen_infographic.py "<prompt>" "<output_png_path>"

Required environment variables:
    IMAGE_API_KEY       - API key for the image service
    IMAGE_BASE_URL      - Base URL of the image API gateway
    IMAGE_MODEL         - Model name (default: gpt-image-2)

The script:
1. Accepts the final image prompt as argv[1] (already prefixed by caller)
2. Accepts absolute output path as argv[2]
3. Reads API credentials from environment only (never argv)
4. Validates the response is a real image (rejects gateway error pages / HTML)
5. Saves atomically (writes .tmp then os.replace)
6. Prints progress as single-line JSON to stdout

Exit codes:
    0 - Success
    1 - Runtime error (API failure, non-image response, download error, etc.)
    2 - Missing required arguments or environment
"""

import base64
import os
import sys
import json
import urllib.request
from openai import OpenAI


# Byte signatures for common image formats. Used to reject gateway responses
# that are actually HTML/JSON error pages mis-saved as .png.
IMAGE_SIGNATURES = (b"\x89PNG", b"\xff\xd8\xff", b"GIF8", b"RIFF")


def print_stage(stage: str, **kwargs):
    """Print a single-line JSON progress update to stdout."""
    print(json.dumps({"stage": stage, **kwargs}, ensure_ascii=False))


def fail(error: str, code: int = 1):
    """Report an error on stdout (JSON) and stderr (plain), then exit.

    Writing to stderr lets the Go pipeline capture a useful message
    (it only reads stderr, not stdout).
    """
    print_stage("error", error=error)
    sys.stderr.write(error + "\n")
    sys.exit(code)


def fetch_image_bytes(client, model, prompt, api_key):
    """Call the image API and return raw image bytes.

    Accepts either b64_json (inline) or url (downloaded). When downloading,
    sends the API key as a Bearer header — some gateways require auth for the
    image fetch as well as the generation call — and verifies the response
    content-type is an image.
    """
    response = client.images.generate(
        model=model,
        prompt=prompt,
        quality="high",
        size="1792x1024",
        output_format="png",
    )
    item = response.data[0]

    b64 = getattr(item, "b64_json", None)
    if b64:
        return base64.b64decode(b64)

    url = getattr(item, "url", None)
    if not url:
        raise RuntimeError("image API returned neither b64_json nor url")

    req = urllib.request.Request(url, headers={"Authorization": f"Bearer {api_key}"})
    with urllib.request.urlopen(req, timeout=120) as resp:
        ctype = resp.headers.get_content_type()
        data = resp.read()
    if not ctype.startswith("image/"):
        snippet = data[:80].decode("utf-8", "replace")
        raise RuntimeError(
            f"image endpoint returned non-image content-type {ctype!r} "
            f"({len(data)} bytes); first bytes: {snippet!r}"
        )
    return data


def assert_is_image(data: bytes):
    """Raise if the bytes don't start with a known image signature."""
    if not data[:12].startswith(IMAGE_SIGNATURES):
        snippet = data[:80].decode("utf-8", "replace")
        raise RuntimeError(f"downloaded content is not an image; first bytes: {snippet!r}")


def main():
    if len(sys.argv) < 3:
        fail("Usage: gen_infographic.py <prompt> <output_path>", code=2)
        return  # unreachable; fail() exits

    final_prompt = sys.argv[1]
    output_path = sys.argv[2]

    api_key = os.environ.get("IMAGE_API_KEY")
    base_url = os.environ.get("IMAGE_BASE_URL")
    model = os.environ.get("IMAGE_MODEL", "gpt-image-2")

    if not api_key:
        fail("Missing IMAGE_API_KEY environment variable", code=2)
    if not base_url:
        fail("Missing IMAGE_BASE_URL environment variable", code=2)
    if not final_prompt:
        fail("Prompt cannot be empty", code=2)

    try:
        parent_dir = os.path.dirname(output_path)
        if parent_dir:
            os.makedirs(parent_dir, exist_ok=True)

        print_stage("requesting")
        client = OpenAI(api_key=api_key, base_url=base_url)
        data = fetch_image_bytes(client, model, final_prompt, api_key)
        assert_is_image(data)

        print_stage("saving")
        tmp_path = output_path + ".tmp"
        with open(tmp_path, "wb") as f:
            f.write(data)
        os.replace(tmp_path, output_path)

        print_stage("done", path=output_path)

    except SystemExit:
        raise
    except Exception as e:
        fail(str(e))


if __name__ == "__main__":
    main()
