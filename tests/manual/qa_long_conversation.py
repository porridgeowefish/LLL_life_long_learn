"""Read-only browser QA for long teacher conversations and generated materials."""

import json
import os
import time
from urllib.parse import quote

from playwright.sync_api import sync_playwright


BASE_URL = os.environ.get("LLL_BASE_URL", "http://127.0.0.1:8787")
SLUG = os.environ.get("LLL_QA_PROJECT", "腾讯云解决方案")


def main() -> None:
    conversation_responses: list[dict] = []
    errors: list[str] = []
    encoded = quote(SLUG, safe="")
    with sync_playwright() as playwright:
        browser = playwright.chromium.launch(headless=True)
        page = browser.new_page(viewport={"width": 1440, "height": 1000})
        page.on("console", lambda message: errors.append(message.text) if message.type == "error" else None)
        page.on("pageerror", lambda error: errors.append(str(error)))

        def capture(response) -> None:
            if "/conversation?" not in response.url:
                return
            try:
                payload = response.json()
            except Exception as cause:
                payload = {"decodeError": str(cause)}
            conversation_responses.append({"url": response.url, "status": response.status, "payload": payload})

        page.on("response", capture)
        page.goto(f"{BASE_URL}/project/{encoded}/assets", wait_until="domcontentloaded")
        page.get_by_role("navigation", name="学习单元").get_by_text("教师", exact=True).wait_for()
        started = time.perf_counter()
        page.get_by_role("navigation", name="学习单元").get_by_text("教师", exact=True).click()
        page.get_by_role("log", name="教师对话").wait_for()
        page.get_by_placeholder(f"和 {SLUG} 的教师继续讨论…").wait_for()
        page.wait_for_timeout(800)
        elapsed_ms = round((time.perf_counter() - started) * 1000)
        transcript = page.get_by_role("log", name="教师对话")
        scroll = transcript.evaluate("el => ({ top: el.scrollTop, height: el.scrollHeight, client: el.clientHeight })")
        assert scroll["height"] - scroll["top"] - scroll["client"] <= 2, scroll

        assert len(conversation_responses) == 1, f"expected one initial conversation request, got {len(conversation_responses)}"
        response = conversation_responses[0]
        assert response["status"] == 200, response
        assert "beforeSeq=0" in response["url"] and "limit=40" in response["url"], response["url"]
        payload = response["payload"]
        assert len(payload.get("messages", [])) <= 40, len(payload.get("messages", []))

        generated = page.request.get(f"{BASE_URL}/api/projects/{encoded}/generated")
        assert generated.ok, generated.text()
        artifacts = generated.json().get("artifacts", [])
        page.get_by_role("navigation", name="学习单元").get_by_text("资料", exact=True).click()
        page.get_by_text("助教生成资料", exact=True).wait_for()
        preview_chars = 0
        if artifacts:
            page.get_by_text(artifacts[0]["title"], exact=True).first.wait_for()
            page.get_by_role("heading", name=artifacts[0]["title"], exact=True).wait_for()
            preview = page.locator("article").filter(has=page.get_by_role("heading", name=artifacts[0]["title"], exact=True))
            page.wait_for_function("node => (node.textContent || '').length > 500", arg=preview.element_handle())
            preview_chars = len(preview.inner_text())

        browser.close()

    assert not errors, f"browser errors: {errors}"
    print(json.dumps({
        "project": SLUG,
        "teacherOpenMs": elapsed_ms,
        "conversationRequests": len(conversation_responses),
        "initialMessages": len(payload.get("messages", [])),
        "totalMessages": payload.get("totalMessages"),
        "hasPrevious": payload.get("hasPrevious", False),
        "generatedMaterials": len(artifacts),
        "generatedPreviewChars": preview_chars,
        "scroll": scroll,
        "browserErrors": errors,
    }, ensure_ascii=False))


if __name__ == "__main__":
    main()
