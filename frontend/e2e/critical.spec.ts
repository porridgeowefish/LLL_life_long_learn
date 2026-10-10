import { expect, test, type Page, type Route } from '@playwright/test';

test.beforeEach(({page}) => {page.on('pageerror', error => console.log('[PAGEERROR]', error.message));});

async function json(route: Route, body: unknown, status = 200) {
  await route.fulfill({ status, contentType: 'application/json', body: JSON.stringify(body) });
}

async function installStableBackend(page: Page) {
  await page.route(url => url.pathname.startsWith('/api/'), async (route) => {
    const request = route.request();
    const path = new URL(request.url()).pathname;
    if (path === '/api/projects' && request.method() === 'POST') {
      return json(route, { project: { id: 'e2e-project', slug: '浏览器验收', title: '浏览器验收', projectType: 'system-learning', status: 'active' } }, 201);
    }
    if (path === '/api/projects') return json(route, { projects: [] });
    if (path === '/api/health') return json(route, { status: 'ok', workspace: 'E2E', agentRuntime: { runtime: { name: 'fake', available: true } } });
    if (path === '/api/sessions/recent' || path === '/api/sessions/active') return json(route, { sessions: [] });
    if (path === '/api/activity') return json(route, { summary: { days: [], totalGrowth: 0 } });
    if (path === '/api/folders') return json(route, { folders: [] });
    if (path === '/api/preferences') return json(route, { content: '' });
    if (path === '/api/settings/ask-ai') return json(route, {default:'',searchEngine:'google',providers:[],bindings:{}});
    if (path === '/api/settings/teacher') return json(route, {autoNotes:{enabled:true,everyTurns:3,providerId:'',model:''}});
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

test('review asset tables keep the document table treatment', async ({ page }) => {
  await page.route('http://127.0.0.1:4173/api/**', async (route) => {
    const path = new URL(route.request().url()).pathname;
    if (path === '/api/health') return json(route, { status: 'ok', workspace: 'E2E', learningWorkspace: { failedProjects: [] } });
    if (path === '/api/projects/body-table') {
      return json(route, { project: { id: 'body-table', slug: 'body-table', title: '正文表格验收', projectType: 'system-learning', status: 'active' } });
    }
    if (path === '/api/projects/body-table/assets') {
      return json(route, { assets: [
                { assetId: 'review', key: 'review', title: '复习稿', currentVersionId: 'body-v1', editRevision: 1, conversationCursor: 0, updatedAt: '' },
        { assetId: 'practice', key: 'practice', title: '练习', currentVersionId: 'practice-v1', editRevision: 1, conversationCursor: 0, updatedAt: '' },
      ] });
    }
    if (path === '/api/projects/body-table/assets/review') {
      return json(route, { meta: { assetId: 'review', key: 'review', title: '复习稿', currentVersionId: 'body-v1', editRevision: 1, conversationCursor: 0, updatedAt: '' }, contentKind: 'markdown', content: '# 容器编排\n\n## 副本与 Deployment\n\n| 组件 | 职责 |\n| --- | --- |\n| Deployment | 管理副本 |' });
    }
        if (path === '/api/projects/body-table/assets/review/annotations') return json(route, { annotations: [] });
    if (path === '/api/projects') return json(route, { projects: [] });
    if (path === '/api/folders') return json(route, { folders: [] });
    if (path === '/api/preferences') return json(route, { content: '' });
    if (path === '/api/settings/ask-ai') return json(route, {default:'',searchEngine:'google',providers:[],bindings:{}});
    if (path === '/api/settings/teacher') return json(route, {autoNotes:{enabled:true,everyTurns:3,providerId:'',model:''}});
    if (path === '/api/sessions/recent' || path === '/api/sessions/active') return json(route, { sessions: [] });
    if (path === '/api/settings/ask-ai') return json(route, { default: '', providers: [], bindings: {} });
    return json(route, {});
  });
  await page.goto('/project/body-table/assets');

  const table = page.locator('table').first();
  await expect(table).toBeVisible();
  const cell = table.locator('td').first();
  const heading = table.locator('th').first();
  expect(await cell.evaluate((element) => getComputedStyle(element).borderTopStyle)).toBe('solid');
  expect(await cell.evaluate((element) => getComputedStyle(element).paddingTop)).toBe('9px');
  expect(await heading.evaluate((element) => getComputedStyle(element).backgroundColor)).not.toBe('rgba(0, 0, 0, 0)');
});

test('home uses available width and keeps heatmap height under a long map list', async ({ page }) => {
  const projects = Array.from({length:40},(_,index)=>({id:`map-${index}`,slug:`map-${index}`,title:`学科地图 ${index}`,projectType:'discipline-map',overviewAvailable:false}));
  projects.push({id:'hidden-chat',slug:'hidden-chat',title:'隐藏对话项目',projectType:'system-learning',overviewAvailable:false});
  const days = Array.from({length:182},(_,index)=>({date:new Date(Date.UTC(2026,3,1+index)).toISOString().slice(0,10),activity:index%5,growth:0,actions:1,events:[]}));
  await page.route(url => url.pathname.startsWith('/api/'),async route=>{const path=new URL(route.request().url()).pathname;if(path==='/api/projects')return json(route,{projects});if(path==='/api/activity')return json(route,{summary:{days,rangeStart:days[0].date,rangeEnd:days.at(-1)!.date,currentStreak:2,longestStreak:3,activeDays:140,totalActions:182,totalGrowth:0}});if(path==='/api/folders')return json(route,{folders:[]});if(path==='/api/health')return json(route,{ok:true,workspace:'E2E',claude:{available:false},learningWorkspace:{failedProjects:[]}});return json(route,{});});
  for(const width of [1440,1920,2560]) {
    await page.setViewportSize({width,height:1000});await page.goto('/');await page.waitForLoadState('networkidle');
    const main=page.getByRole('main');await expect(main.getByRole('heading',{name:'学科地图 0',exact:true})).toBeVisible();await expect(main.getByText('隐藏对话项目',{exact:true})).toHaveCount(0);
    await expect(main.getByText('本周投入',{exact:true})).toHaveCount(0);
    const mainBox=await main.boundingBox();const scrollBox=await main.locator(':scope > div > div').first().boundingBox();expect(Math.abs(mainBox!.x+mainBox!.width-scrollBox!.x-scrollBox!.width)).toBeLessThan(2);
    await expect(main.getByRole('gridcell')).toHaveCount(182);
    const grid=await main.getByRole('grid').boundingBox();expect(grid!.height).toBeGreaterThan(100);
    expect(await page.evaluate(()=>document.documentElement.scrollWidth)).toBeLessThanOrEqual(width);
    if(width===1920)await page.screenshot({path:'../.artifacts/iteration18/home-1920.png'});
  }
});

test('mobile sidebar reaches model configuration and home by keyboard',async ({page})=>{
 await installStableBackend(page);await page.setViewportSize({width:620,height:900});await page.goto('/');await page.waitForLoadState('networkidle');
 await page.getByRole('button',{name:'打开侧栏'}).click();await page.getByRole('link',{name:'API 模型',exact:true}).click();await expect(page).toHaveURL(/models$/);
 await page.getByRole('button',{name:'打开侧栏'}).click();const home=page.getByRole('link',{name:'主页',exact:true});await home.focus();await home.press('Enter');await expect(page.getByRole('heading',{name:'学习主页'})).toBeVisible();
 expect(await page.evaluate(()=>document.documentElement.scrollWidth)).toBeLessThanOrEqual(620);
 await page.screenshot({path:'../.artifacts/iteration18/home-mobile.png'});
});

test('teacher notes settings are styled and save a valid interval on desktop and mobile', async ({ page }) => {
  await installStableBackend(page);
  let settings = { enabled: true, everyTurns: 3, providerId: '', model: '' };
  await page.route(url => url.pathname === '/api/settings/teacher', async route => {
    if (route.request().method() === 'PUT') settings = route.request().postDataJSON().autoNotes;
    return json(route, { autoNotes: settings });
  });
  for (const width of [1440, 620]) {
    await page.setViewportSize({ width, height: 1000 });
    await page.goto('/settings');
    const notes = page.getByRole('region', { name: '教师自动笔记' });
    await expect(notes).toBeVisible();
    const title = notes.getByRole('heading', { name: '教师自动笔记' });
    expect(await title.evaluate(el => parseFloat(getComputedStyle(el).fontSize))).toBeLessThanOrEqual(20);
    expect(await notes.evaluate(el => getComputedStyle(el).borderTopStyle)).toBe('solid');
    const interval = notes.getByLabel('记录间隔');
    await interval.fill('5');
    await notes.getByRole('button', { name: '保存自动笔记' }).click();
    await expect(notes.getByRole('status')).toContainText('已保存');
    expect(settings.everyTurns).toBe(5);
    await interval.fill('0');
    await expect(notes.getByRole('button', { name: '保存自动笔记' })).toBeDisabled();
    await expect(notes.getByRole('status')).not.toContainText('已保存');
    await interval.fill('3');
    expect(await page.evaluate(() => document.documentElement.scrollWidth)).toBeLessThanOrEqual(width);
    await notes.screenshot({ path: `../.artifacts/iteration18/notes-settings-${width}.png` });
  }
});

test('token bars keep selected details and open the project by keyboard',async ({page})=>{
 await installStableBackend(page);
 await page.route(url=>url.pathname==='/api/usage/projects',route=>json(route,{projects:[{projectSlug:'usage-topic',title:'用量验收项目',inputTokens:120,outputTokens:80,noteInputTokens:20,noteOutputTokens:10}]}));
 await page.route(url=>url.pathname==='/api/projects/usage-topic',route=>json(route,{project:{id:'usage-topic',slug:'usage-topic',title:'用量验收项目',projectType:'system-learning'}}));
 await page.goto('/usage');const bar=page.getByRole('button',{name:/用量验收项目.*230/});await bar.focus();await expect(page.getByRole('tooltip')).toContainText('笔记输入 20');await bar.press('Enter');
 const link=page.getByRole('link',{name:'进入项目'});await link.focus();await expect(page.getByRole('tooltip')).toContainText('用量验收项目');await page.screenshot({path:'../.artifacts/iteration18/usage-chart.png'});await link.press('Enter');await expect(page).toHaveURL(/project\/usage-topic/);
});

test('model key stays complete through reveal save and reload',async ({page})=>{
 await installStableBackend(page);const key='sk-test-'+ 'abcdefghij'.repeat(40);let saved=false;
 await page.route(url=>url.pathname==='/api/settings/ask-ai',async route=>{
  if(route.request().method()==='PUT'){expect(route.request().postDataJSON().providers[0].apiKey).toBe(key);saved=true;}
  return json(route,{default:'p',providers:[{id:'p',kind:'openai',name:'验收模型',baseURL:'https://example.invalid/v1',model:'test-model',apiKey:key}],bindings:{}});
 });
 await page.goto('/models');const input=page.getByRole('textbox',{name:'API Key'});const secret=page.getByLabel('API Key',{exact:true});await expect(secret).toHaveAttribute('type','password');await expect(secret).toHaveValue(key);await page.getByRole('button',{name:'显示 API Key'}).click();await expect(input).toHaveValue(key);await page.getByRole('button',{name:'保存配置',exact:true}).click();await expect.poll(()=>saved).toBe(true);await page.reload();await expect(secret).toHaveValue(key);await expect(secret).toHaveAttribute('type','password');
});

test('lays out completed teacher rich text without content-visibility skipping', async ({ page }) => {
  await page.route(url => url.pathname.startsWith('/api/'), async (route) => {
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
