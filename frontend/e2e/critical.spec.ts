import { expect, test, type Page, type Route } from '@playwright/test';

async function json(route: Route, body: unknown, status = 200) {
  await route.fulfill({ status, contentType: 'application/json', body: JSON.stringify(body) });
}

async function installStableBackend(page: Page) {
  await page.route('**/api/**', async (route) => {
    const request = route.request();
    const path = new URL(request.url()).pathname;
    if (path === '/api/projects' && request.method() === 'POST') {
      return json(route, { project: { id: 'e2e-project', slug: '浏览器验收', title: '浏览器验收', projectType: 'system-learning', status: 'active' } }, 201);
    }
    if (path === '/api/projects') return json(route, { projects: [] });
    if (path === '/api/health') return json(route, { status: 'ok', workspace: 'E2E', agentRuntime: { runtime: { name: 'fake', available: true } } });
    if (path === '/api/sessions/recent' || path === '/api/sessions/active') return json(route, { sessions: [] });
    if (path === '/api/activity') return json(route, { days: [], totalGrowth: 0 });
    if (path === '/api/folders') return json(route, { folders: [] });
    if (path === '/api/preferences') return json(route, { content: '' });
    return json(route, {});
  });
}

test('creates a system-learning project and enters its workspace', async ({ page }) => {
  await installStableBackend(page);
  await page.goto('/');
  await page.getByRole('button', { name: '新建项目' }).click();
  await page.getByText('系统学习', { exact: true }).click();
  await page.getByLabel('项目标题 *').fill('浏览器验收');
  await page.getByRole('button', { name: '创建项目' }).click();
  await expect(page).toHaveURL(/\/project\/%E6%B5%8F%E8%A7%88%E5%99%A8%E9%AA%8C%E6%94%B6(?:\/|$)/);
});

test('/memory keeps the compatibility redirect to preferences', async ({ page }) => {
  await installStableBackend(page);
  await page.goto('/memory');
  await expect(page).toHaveURL(/\/preferences$/);
  await expect(page.getByRole('heading', { name: '全局学习偏好' })).toBeVisible();
});

test('usage page styles stay inside the usage records', async ({ page }) => {
  await page.setContent('<ul><li data-testid="foreign-list-item">教师回复</li></ul>');
  await page.addStyleTag({ path: 'src/features/usage/UsagePage.module.css' });
  const item = page.getByTestId('foreign-list-item');

  expect(await item.evaluate((element) => getComputedStyle(element).display)).toBe('list-item');
  expect(await item.evaluate((element) => getComputedStyle(element).backgroundColor)).toBe('rgba(0, 0, 0, 0)');
});

test('shell dividers align with the brand rail and workspace header', async ({ page }) => {
  await page.setContent(`
    <style>* { box-sizing: border-box; } body { margin: 0; }</style>
    <header class="topbar">
      <a class="logo"><img class="logoMark"><span>LifeLongLearn</span></a>
      <span class="sep" data-testid="brand-divider"></span>
      <nav class="nav"></nav>
    </header>
    <div style="display:flex;height:300px">
      <aside class="sidebar" data-testid="sidebar"><div class="head" data-testid="navigation-head">导航</div></aside>
      <main style="flex:1"><header class="header" data-testid="workspace-header">教师</header></main>
    </div>
  `);
  await page.addStyleTag({ path: 'src/shared/styles/tokens.css' });
  await page.addStyleTag({ path: 'src/app/layout/Topbar.module.css' });
  await page.addStyleTag({ path: 'src/app/layout/Sidebar.module.css' });
  await page.addStyleTag({ path: 'src/features/learning/components/LearningWorkspace.module.css' });

  const sidebar = await page.getByTestId('sidebar').boundingBox();
  const brandDivider = await page.getByTestId('brand-divider').boundingBox();
  const navigationHead = await page.getByTestId('navigation-head').boundingBox();
  const workspaceHeader = await page.getByTestId('workspace-header').boundingBox();

  expect(sidebar).not.toBeNull();
  expect(brandDivider).not.toBeNull();
  expect(navigationHead).not.toBeNull();
  expect(workspaceHeader).not.toBeNull();
  expect(sidebar!.x + sidebar!.width).toBe(brandDivider!.x);
  expect(navigationHead!.y + navigationHead!.height).toBe(workspaceHeader!.y + workspaceHeader!.height);
});

test('lays out completed teacher rich text without content-visibility skipping', async ({ page }) => {
  await page.route('**/api/**', async (route) => {
    const path = new URL(route.request().url()).pathname;
    if (path === '/api/health') return json(route, { status: 'ok', workspace: 'E2E', learningWorkspace: { failedProjects: [] } });
    if (path === '/api/projects/streaming-layout') {
      return json(route, { project: { id: 'streaming-layout', slug: 'streaming-layout', title: '流式布局验收', projectType: 'system-learning', status: 'active' } });
    }
    if (path === '/api/projects/streaming-layout/conversation') {
      return json(route, {
        conversationId: 'conversation-1', unitId: 'streaming-layout', latestSeq: 1, pageFromSeq: 1, pageThroughSeq: 1,
        hasMore: false, hasPrevious: false, totalMessages: 1, taskLinks: [],
        messages: [{ id: 'teacher-1', role: 'teacher', status: 'completed', createdAt: '', blocks: [{
          id: 'body', type: 'markdown', source: '## 两道小题检验今天的内容\n\n| 目的网段 | 下一跳 |\n|---|---|\n| 0.0.0.0/0 | NAT 网关 |\n| 172.16.0.0/12 | 对等连接 |\n\n1. 你要在广州用 3 个 AZ 做高可用，至少要建几个子网？',
        }] }],
      });
    }
    if (path === '/api/projects/streaming-layout/assistant-tasks') return json(route, { tasks: [] });
    if (path === '/api/projects/streaming-layout/assets') return json(route, { assets: [] });
    if (path === '/api/projects/streaming-layout/sources') return json(route, { sources: [] });
    if (path === '/api/projects/streaming-layout/conversation/responses/active') return route.fulfill({ status: 204 });
    if (path === '/api/settings/ask-ai') return json(route, { default: '', providers: [], bindings: {} });
    return json(route, {});
  });

  await page.goto('/project/streaming-layout/teacher');
  const message = page.locator('article').filter({ hasText: '两道小题检验今天的内容' });
  await expect(message).toBeVisible();
  await expect(message.locator('table')).toHaveCount(1);
  await expect(message.locator('ol > li')).toHaveCount(1);
  expect(await message.evaluate((element) => getComputedStyle(element).contentVisibility)).toBe('visible');
});
