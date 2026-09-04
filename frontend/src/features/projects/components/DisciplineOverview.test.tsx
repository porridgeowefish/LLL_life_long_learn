import { fireEvent, render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import {
  DisciplineOverview,
  extractDisciplineOutline,
  extractDisciplineTopics,
} from './DisciplineOverview';

const generateHarness = vi.hoisted(() => ({
  mutateAsync: vi.fn(),
  isPending: false,
  error: null as Error | null,
}));

const savePlanHarness = vi.hoisted(() => ({
  mutate: vi.fn(),
  isPending: false,
}));

vi.mock('react-router-dom', () => ({ useNavigate: () => vi.fn() }));

vi.mock('@/features/projects/api/projects', () => ({
  useDisciplineOverview: () => ({
    data: {
      title: '物理学',
      content: '# 物理学：学科总览\n\n## 主要研究领域与知识架构\n\n### 力与运动\n\n#### 经典力学\n说明\n\n#### 热力学\n说明\n\n## 典型应用\n说明',
    },
    isLoading: false,
    error: null,
  }),
  useDisciplineTopics: () => ({
    data: {
      schemaVersion: 1,
      topics: [
        {
          id: 'classical-mechanics',
          title: '经典力学',
          chapterTitle: '力与运动',
          goal: '解释宏观低速物体的运动规律',
          inScope: ['牛顿运动定律'],
          outOfScope: ['热现象'],
          prerequisites: ['向量'],
          ownedConcepts: ['惯性参考系'],
          reusedConcepts: ['微积分'],
        },
        {
          id: 'thermodynamics',
          title: '热力学',
          chapterTitle: '力与运动',
          goal: '解释宏观热现象',
          inScope: ['状态量'],
          outOfScope: ['运动方程'],
          prerequisites: ['代数'],
          ownedConcepts: ['熵'],
          reusedConcepts: ['微积分'],
        },
      ],
      updatedAt: '2026-07-28T00:00:00Z',
    },
    isLoading: false,
    error: null,
  }),
  useDisciplineLearningPlan: () => ({
    data: {
      schemaVersion: 1,
      items: [{ id: 'task-1', topicTitle: '经典力学', status: 'planned', addedAt: '2026-07-18T00:00:00Z' }],
      updatedAt: '2026-07-18T00:00:00Z',
    },
    isLoading: false,
    error: null,
  }),
  useSaveDisciplineLearningPlan: () => savePlanHarness,
  useGenerateDisciplineOverview: () => generateHarness,
}));

vi.mock('@/features/projects/api/folders', () => ({
  useFolders: () => ({ data: { folders: [] } }),
  useSaveFolders: () => ({ mutateAsync: vi.fn() }),
}));

vi.mock('@/shared/primitive/MarkdownView', () => ({
  MarkdownView: ({ source, className }: { source: string; className?: string }) => (
    <div className={className}>
      {source.split(/\r?\n/).map((line, index) => {
        if (line.startsWith('### ')) return <h3 key={index}>{line.slice(4)}</h3>;
        if (line.startsWith('#### ')) return <h4 key={index}>{line.slice(5)}</h4>;
        if (line.startsWith('## ')) return <h2 key={index}>{line.slice(3)}</h2>;
        return line ? <p key={index}>{line}</p> : null;
      })}
    </div>
  ),
}));

vi.mock('./CreateProjectModal', () => ({
  CreateProjectModal: ({
    open,
    initialTitle,
    initialScopeSource,
  }: {
    open: boolean;
    initialTitle: string;
    initialScopeSource?: { mapSlug: string; topicId: string };
  }) =>
    open ? (
      <div
        data-testid="deep-dive-prefill"
        data-map-slug={initialScopeSource?.mapSlug}
        data-topic-id={initialScopeSource?.topicId}
      >
        {initialTitle}
      </div>
    ) : null,
}));

describe('extractDisciplineTopics', () => {
  it('extracts every level-three key concept or branch as an inline deep-dive topic', () => {
    const markdown = '## 主要研究领域\n### 经典力学\n### 热力学\n## 典型应用\n### 航天';
    expect(extractDisciplineTopics(markdown)).toEqual(['经典力学', '热力学', '航天']);
  });

  it('builds a numbered Word-style outline from the same heading hierarchy', () => {
    const outline = extractDisciplineOutline('## 学科边界\n## 主要研究领域\n### 经典力学\n### 热力学\n## 典型应用');
    expect(outline.map(({ number, title, actionable }) => ({ number, title, actionable }))).toEqual([
      { number: '1', title: '学科边界', actionable: false },
      { number: '2', title: '主要研究领域', actionable: false },
      { number: '2.1', title: '经典力学', actionable: true },
      { number: '2.2', title: '热力学', actionable: true },
      { number: '3', title: '典型应用', actionable: false },
    ]);
  });

  it('builds a three-level outline and makes only fine-grained H4 topics actionable', () => {
    const outline = extractDisciplineOutline([
      '## 主要研究领域与知识架构',
      '### 数学基础',
      '#### 线性代数',
      '#### 概率论',
      '### 机器学习',
      '#### 监督学习',
      '## 其他信息',
      '### 阅读提示',
    ].join('\n'));

    expect(outline.map(({ number, title, actionable }) => ({ number, title, actionable }))).toEqual([
      { number: '1', title: '主要研究领域与知识架构', actionable: false },
      { number: '1.1', title: '数学基础', actionable: false },
      { number: '1.1.1', title: '线性代数', actionable: true },
      { number: '1.1.2', title: '概率论', actionable: true },
      { number: '1.2', title: '机器学习', actionable: false },
      { number: '1.2.1', title: '监督学习', actionable: true },
      { number: '2', title: '其他信息', actionable: false },
      { number: '2.1', title: '阅读提示', actionable: false },
    ]);
    expect(extractDisciplineTopics('## 知识架构\n### 数学基础\n#### 线性代数\n#### 概率论')).toEqual([
      '线性代数',
      '概率论',
    ]);
  });
});

describe('DisciplineOverview', () => {
  beforeEach(() => {
    generateHarness.mutateAsync.mockReset();
    generateHarness.isPending = false;
    generateHarness.error = null;
    savePlanHarness.mutate.mockReset();
    savePlanHarness.isPending = false;
  });

  it('renders a document outline and opens the prefilled form from the inline body action', async () => {
    render(<DisciplineOverview slug="physics" title="物理学" />);

    expect(screen.getByRole('navigation', { name: '学科总览目录' })).toBeTruthy();
    expect(screen.queryByText('选择一个领域深入学习')).toBeNull();
    const sectionHeading = screen
      .getAllByRole('heading', { level: 2 })
      .find((heading) => heading.textContent?.startsWith('主要研究领域'));
    expect(sectionHeading?.querySelector('button')).toBeNull();

    const chapterHeading = screen.getByRole('heading', { level: 3, name: '力与运动' });
    expect(chapterHeading.querySelector('button')).toBeNull();

    const classicMechanicsAction = await screen.findByRole('button', { name: '深入学习：经典力学' });
    expect(classicMechanicsAction.closest('h4')?.textContent).toContain('经典力学');
    fireEvent.click(classicMechanicsAction);

    expect(screen.getByTestId('deep-dive-prefill').textContent).toBe('经典力学');
    expect(screen.getByTestId('deep-dive-prefill').getAttribute('data-map-slug')).toBe('physics');
    expect(screen.getByTestId('deep-dive-prefill').getAttribute('data-topic-id')).toBe('classical-mechanics');
    expect(screen.getByRole('button', { name: '已加入计划：经典力学' })).toBeDisabled();
    fireEvent.click(screen.getByRole('button', { name: '加入计划：热力学' }));
    expect(savePlanHarness.mutate).toHaveBeenCalledWith(expect.objectContaining({
      items: expect.arrayContaining([expect.objectContaining({
        topicId: 'thermodynamics',
        topicTitle: '热力学',
        status: 'planned',
      })]),
    }));
  });

  it('switches to the independent learning-plan page from the discipline-map top bar', () => {
    render(<DisciplineOverview slug="physics" title="物理学" />);

    expect(screen.getByRole('tab', { name: '学科总览' }).getAttribute('aria-selected')).toBe('true');
    fireEvent.click(screen.getByRole('tab', { name: '学习计划' }));

    expect(screen.getByRole('tab', { name: '学习计划' }).getAttribute('aria-selected')).toBe('true');
    expect(screen.getByRole('heading', { name: '学习任务清单' })).toBeTruthy();
    expect(screen.getByText('经典力学')).toBeTruthy();
    expect(screen.getByRole('button', { name: '开始' })).toBeTruthy();
    expect(screen.queryByRole('navigation', { name: '学科总览目录' })).toBeNull();
  });

  it('keeps overview generation visibly in progress beside the action', () => {
    generateHarness.isPending = true;

    render(<DisciplineOverview slug="physics" title="物理学" />);

    expect(screen.getByRole('button', { name: '正在启动 Agent CLI' })).toBeDisabled();
    expect(screen.getByRole('status').textContent).toContain('正在创建会话并打开所选 Agent CLI');
  });

  it('reports the explicit visible CLI launch instead of hidden model progress', async () => {
    generateHarness.mutateAsync.mockResolvedValueOnce({
      session: { id: 'session-map-1' },
      runDir: 'runs/example-encyclopedia',
    });
    render(<DisciplineOverview slug="physics" title="物理学" />);

    fireEvent.click(screen.getByRole('button', { name: '重新调用百科 Agent' }));

    expect((await screen.findByRole('status')).textContent).toContain('百科 Agent CLI 已在可见终端启动');
    expect(screen.queryByText(/20–90 秒/)).toBeNull();
  });
});
