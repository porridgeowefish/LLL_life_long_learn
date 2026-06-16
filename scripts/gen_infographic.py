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
4. Downloads the image atomically (writes .tmp then os.replace)
5. Prints progress as single-line JSON to stdout

Exit codes:
    0 - Success
    1 - Runtime error (API failure, download error, etc.)
    2 - Missing required arguments or environment
"""

import os
import sys
import json
import urllib.request
from openai import OpenAI


def print_stage(stage: str, **kwargs):
    """Print a single-line JSON progress update."""
    obj = {"stage": stage, **kwargs}
    print(json.dumps(obj, ensure_ascii=False))


def main():
    # Validate argv
    if len(sys.argv) < 3:
        print_stage("error", error="Usage: gen_infographic.py <prompt> <output_path>")
        sys.exit(2)

    final_prompt = sys.argv[1]
    output_path = sys.argv[2]

    # Validate environment
    api_key = os.environ.get("IMAGE_API_KEY")
    base_url = os.environ.get("IMAGE_BASE_URL")
    model = os.environ.get("IMAGE_MODEL", "gpt-image-2")

    if not api_key:
        print_stage("error", error="Missing IMAGE_API_KEY environment variable")
        sys.exit(2)
    if not base_url:
        print_stage("error", error="Missing IMAGE_BASE_URL environment variable")
        sys.exit(2)
    if not final_prompt:
        print_stage("error", error="Prompt cannot be empty")
        sys.exit(2)

    try:
        # Ensure parent directory exists
        parent_dir = os.path.dirname(output_path)
        if parent_dir:
            os.makedirs(parent_dir, exist_ok=True)

        # Initialize OpenAI client
        print_stage("requesting")
        client = OpenAI(api_key=api_key, base_url=base_url)

        # Generate image
        response = client.images.generate(
            model=model,
            prompt=final_prompt,
            quality="high",
            size="1792x1024",
            output_format="png"
        )
        image_url = response.data[0].url

        # Download atomically to .tmp then replace
        print_stage("downloading")
        tmp_path = output_path + ".tmp"
        urllib.request.urlretrieve(image_url, tmp_path)
        os.replace(tmp_path, output_path)

        print_stage("done", path=output_path)

    except Exception as e:
        print_stage("error", error=str(e))
        sys.exit(1)


if __name__ == "__main__":
    main()
