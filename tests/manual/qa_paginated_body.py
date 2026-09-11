"""Read-only browser QA for the paginated Markdown body asset."""

import json
import os
from urllib.parse import quote

from playwright.sync_api import sync_playwright


BASE_URL = os.environ.get("LLL_BASE_URL", "http://127.0.0.1:8787")
SLUG = os.environ.get("LLL_QA_PROJECT", "腾讯云解决方案")
SCREENSHOT = os.environ.get("LLL_QA_SCREENSHOT", "")


def main() -> None:
    errors: list[str] = []
    with sync_playwright() as playwright:
        browser = playwright.chromium.launch(headless=True)
        page = browser.new_page(viewport={"width": 1600, "height": 1000})
        page.on("console", lambda message: errors.append(message.text) if message.type == "error" else None)
        page.on("pageerror", lambda error: errors.append(str(error)))
        page.goto(f"{BASE_URL}/project/{quote(SLUG, safe='')}/assets", wait_until="domcontentloaded")
        page.get_by_role("tablist", name="正文页面").wait_for(state="visible")

        navigation = page.get_by_role("tablist", name="正文页面")
        navigation.wait_for()
        tabs = navigation.get_by_role("tab")
        page_count = tabs.count()
        assert page_count > 1, "long Markdown body was not paginated"
        assert tabs.nth(0).get_attribute("aria-selected") == "true"
        first_title = tabs.nth(0).get_attribute("aria-label")
        second_title = tabs.nth(1).get_attribute("aria-label")
        first_heading = page.locator("h2", has_text=first_title).first
        second_heading = page.locator("h2", has_text=second_title).first
        assert first_heading.is_visible()
        assert not second_heading.is_visible()

        page.get_by_role("button", name="下一页").click()
        assert tabs.nth(1).get_attribute("aria-selected") == "true"
        assert not first_heading.is_visible()
        assert second_heading.is_visible()

        dimensions = navigation.evaluate("""node => {
          const documentCard = node.closest('article');
          const layout = documentCard?.parentElement;
          return {
            viewport: window.innerWidth,
            card: documentCard?.getBoundingClientRect().width || 0,
            layout: layout?.getBoundingClientRect().width || 0
          };
        }""")
        assert dimensions["layout"] >= dimensions["viewport"] * 0.72, dimensions
        assert dimensions["card"] >= dimensions["layout"] * 0.68, dimensions
        if SCREENSHOT:
            page.screenshot(path=SCREENSHOT, full_page=False)
        browser.close()

    assert not errors, errors
    print(json.dumps({
        "project": SLUG,
        "pages": page_count,
        "firstPage": first_title,
        "secondPage": second_title,
        "dimensions": dimensions,
        "browserErrors": errors,
    }, ensure_ascii=False))


if __name__ == "__main__":
    main()
