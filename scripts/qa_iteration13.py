"""Browser smoke test for the iteration-13 learning workspace.

The server must already be listening on LLL_BASE_URL (default localhost:8787)
and should point WORKSPACE at an isolated test directory.
"""

import os
import tempfile
import uuid
from pathlib import Path

from playwright.sync_api import sync_playwright


BASE_URL = os.environ.get("LLL_BASE_URL", "http://127.0.0.1:8787")
EXPECTED_WORKSPACE = os.environ.get("LLL_EXPECTED_WORKSPACE", "").strip()
SCREENSHOT = Path(os.environ.get(
    "LLL_QA_SCREENSHOT",
    str(Path.cwd() / "frontend-designs" / "v2" / "iteration13-learning-workspace.png"),
))
CHAT_SCREENSHOT = SCREENSHOT.with_name(SCREENSHOT.stem + "-teacher" + SCREENSHOT.suffix)
MODEL_SCREENSHOT = SCREENSHOT.with_name(SCREENSHOT.stem + "-models" + SCREENSHOT.suffix)


def select_text(page, needle: str) -> None:
    page.locator("article").filter(has_text=needle).first.evaluate(
        """(article, needle) => {
          const walker = document.createTreeWalker(article, NodeFilter.SHOW_TEXT);
          while (walker.nextNode()) {
            const node = walker.currentNode;
            const start = node.data.indexOf(needle);
            if (start >= 0) {
              const range = document.createRange();
              range.setStart(node, start);
              range.setEnd(node, start + needle.length);
              const selection = window.getSelection();
              selection.removeAllRanges();
              selection.addRange(range);
              const rect = range.getBoundingClientRect();
              node.parentElement.dispatchEvent(new PointerEvent('pointerup', {
                bubbles: true,
                clientX: rect.right,
                clientY: rect.bottom,
              }));
              return;
            }
          }
          throw new Error(`text not found: ${needle}`);
        }""",
        needle,
    )


def main() -> None:
    errors: list[str] = []
    slug = "qa-" + uuid.uuid4().hex[:12]
    with sync_playwright() as playwright:
        browser = playwright.chromium.launch(headless=True)
        page = browser.new_page(viewport={"width": 1440, "height": 1000})
        page.on("console", lambda message: errors.append(message.text) if message.type == "error" else None)
        page.on("pageerror", lambda error: errors.append(str(error)))

        health = page.request.get(f"{BASE_URL}/api/health")
        assert health.ok, health.text()
        if EXPECTED_WORKSPACE:
            actual_workspace = str(health.json().get("workspace", ""))
            assert Path(actual_workspace).resolve() == Path(EXPECTED_WORKSPACE).resolve(), (
                f"wrong QA workspace: expected {EXPECTED_WORKSPACE!r}, got {actual_workspace!r}"
            )

        created = page.request.post(
            f"{BASE_URL}/api/projects",
            data={"title": "闭包学习", "slug": slug, "projectType": "system-learning"},
        )
        assert created.ok, created.text()
        asset_response = page.request.get(f"{BASE_URL}/api/projects/{slug}/assets/body")
        assert asset_response.ok, asset_response.text()
        asset = asset_response.json()
        saved = page.request.put(
            f"{BASE_URL}/api/projects/{slug}/assets/body",
            data={
                "baseEditRevision": asset["meta"]["editRevision"],
                "content": "# 闭包\n\n闭包是函数与其词法环境的组合。\n\n词法环境让函数离开定义位置后仍能访问变量。",
            },
        )
        assert saved.ok, saved.text()

        page.goto(f"{BASE_URL}/project/{slug}/assets")
        try:
            page.wait_for_load_state("networkidle", timeout=10_000)
        except Exception:
            # The application intentionally keeps one global SSE request open.
            pass
        page.get_by_role("heading", name="闭包学习").wait_for()
        page.get_by_role("heading", name="正文").wait_for()
        assert page.get_by_role("navigation", name="学习单元").get_by_text("教师").count() == 1
        assert page.get_by_role("navigation", name="学习单元").get_by_text("资产").count() == 1
        assert page.get_by_role("navigation", name="学习单元").get_by_text("资料").count() == 1

        select_text(page, "词法环境")
        toolbar = page.get_by_role("toolbar", name="选中文本操作")
        toolbar.wait_for()
        toolbar.get_by_role("button", name="保存批注").click()
        page.get_by_role("complementary", name="正文批注").get_by_text("词法环境", exact=True).wait_for()

        select_text(page, "访问变量")
        toolbar = page.get_by_role("toolbar", name="选中文本操作")
        toolbar.wait_for()
        toolbar.get_by_role("button", name="问 AI").click()
        page.get_by_role("dialog", name="问 AI").wait_for()
        assert "访问变量" in page.get_by_role("dialog", name="问 AI").locator("input").input_value()
        page.get_by_role("button", name="关闭").click()

        page.get_by_role("navigation", name="学习单元").get_by_text("资料").click()
        page.get_by_role("heading", name="教师与助教可引用的材料").wait_for()
        with tempfile.NamedTemporaryFile(suffix=".txt", delete=False) as source_file:
            source_file.write("closure reference".encode("utf-8"))
            source_path = source_file.name
        try:
            page.locator('input[type="file"]').set_input_files(source_path)
            confirm = page.get_by_role("dialog", name="确认资料处理")
            confirm.wait_for()
            assert confirm.get_by_role("button", name="保存并解析").is_disabled()
            confirm.get_by_role("button", name="仅保存原件").click()
            page.get_by_text(Path(source_path).name, exact=True).wait_for()

            page.get_by_role("navigation", name="学习单元").get_by_text("教师").click()
            page.get_by_text("最想先弄懂什么", exact=False).wait_for()
            composer = page.get_by_placeholder("和 闭包学习 的教师继续讨论…")
            composer.wait_for()
            assert composer.bounding_box()["y"] < page.viewport_size["height"]
            assert page.get_by_role("button", name="沉淀").count() == 0
            assert page.get_by_text("上传资料", exact=True).count() == 1
            source_picker = page.locator("details").filter(has_text=Path(source_path).name)
            source_picker.locator("summary").click()
            source_picker.get_by_role("checkbox", name=Path(source_path).name).check()
            assert source_picker.locator("summary").inner_text().strip() == "资料 1"

            CHAT_SCREENSHOT.parent.mkdir(parents=True, exist_ok=True)
            page.screenshot(path=str(CHAT_SCREENSHOT), full_page=True)

            page.goto(f"{BASE_URL}/models")
            page.get_by_role("heading", name="API 模型").wait_for()
            assert page.get_by_text("添加模型", exact=True).count() >= 1
            page.get_by_label("Ask AI 默认模型").wait_for()
            assert page.get_by_role("link", name="Ask AI", exact=True).count() == 0
            page.screenshot(path=str(MODEL_SCREENSHOT), full_page=True)
            page.goto(f"{BASE_URL}/settings")
            page.get_by_role("heading", name="搜索引擎").wait_for()
        finally:
            try:
                Path(source_path).unlink(missing_ok=True)
            except PermissionError:
                # Chromium may retain the upload handle until browser.close().
                pass

        SCREENSHOT.parent.mkdir(parents=True, exist_ok=True)
        page.screenshot(path=str(SCREENSHOT), full_page=True)
        assert not errors, "browser errors: " + " | ".join(errors)
        browser.close()


if __name__ == "__main__":
    main()
