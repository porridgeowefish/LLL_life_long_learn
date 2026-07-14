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

vi.mock('react-router-dom', () => ({ useNavigate: () => vi.fn() }));

vi.mock('@/api/projects', () => ({
  useDisciplineOverview: () => ({
    data: {
      title: '物理学',
      content: '# 物理学：学科总览\n\n## 主要研究领域\n\n### 经典力学\n说明\n\n### 热力学\n说明\n\n## 典型应用\n\n### 航天\n说明',
    },
    isLoading: false,
    error: null,
  }),
  useGenerateDisciplineOverview: () => generateHarness,
}));

vi.mock('@/api/folders', () => ({
  useFolders: () => ({ data: { folders: [] } }),
  useSaveFolders: () => ({ mutateAsync: vi.fn() }),
}));

vi.mock('@/components/primitive/MarkdownView', () => ({
  MarkdownView: ({ source, className }: { source: string; className?: string }) => (
    <div className={className}>
      {source.split(/\r?\n/).map((line, index) => {
        if (line.startsWith('### ')) return <h3 key={index}>{line.slice(4)}</h3>;
        if (line.startsWith('## ')) return <h2 key={index}>{line.slice(3)}</h2>;
        return null;
      })}
    </div>
  ),
}));

vi.mock('./CreateProjectModal', () => ({
  CreateProjectModal: ({ open, initialTitle }: { open: boolean; initialTitle: string }) =>
    open ? <div data-testid="deep-dive-prefill">{initialTitle}</div> : null,
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
});

describe('DisciplineOverview', () => {
  beforeEach(() => {
    generateHarness.mutateAsync.mockReset();
    generateHarness.isPending = false;
    generateHarness.error = null;
  });

  it('renders a document outline and opens the prefilled form from the inline body action', async () => {
    render(<DisciplineOverview slug="physics" title="物理学" />);

    expect(screen.getByRole('navigation', { name: '学科总览目录' })).toBeTruthy();
    expect(screen.queryByText('选择一个领域深入学习')).toBeNull();
    const sectionHeading = screen
      .getAllByRole('heading', { level: 2 })
      .find((heading) => heading.textContent?.startsWith('主要研究领域'));
    expect(sectionHeading?.querySelector('button')).toBeNull();

    const classicMechanicsAction = await screen.findByRole('button', { name: '深入学习：经典力学' });
    expect(classicMechanicsAction.closest('h3')?.textContent).toContain('经典力学');
    fireEvent.click(classicMechanicsAction);

    expect(screen.getByTestId('deep-dive-prefill').textContent).toBe('经典力学');
    const aerospaceAction = screen.getByRole('button', { name: '深入学习：航天' });
    expect(aerospaceAction.closest('h3')?.textContent).toContain('航天');
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
