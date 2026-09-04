import { fireEvent, render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { AssetsView } from './AssetsView';

const assets = {
  intro: { contentKind: 'markdown', content: '# 引入' },
  body: { contentKind: 'explain-pages', content: '完整内容见 `explain/manifest.json`' },
  practice: { contentKind: 'practice-set', content: '```json\n{"tasks":[]}\n```' },
};

vi.mock('@/features/learning/api/learningWorkspace', () => ({
  useAssets: () => ({ data: [
    { key: 'intro', title: '引入' },
    { key: 'body', title: '正文' },
    { key: 'practice', title: '练习' },
  ] }),
  useAsset: (_slug: string, key: keyof typeof assets) => ({ data: {
    meta: { key, title: key === 'body' ? '正文' : key === 'practice' ? '练习' : '引入', editRevision: 1, currentVersionId: 'aver_1' },
    ...assets[key],
  } }),
  useSaveAsset: () => ({ mutate: vi.fn(), isPending: false, isError: false }),
  useGeneratedArtifacts: () => ({ data: [] }),
  useGeneratedArtifactEntry: () => ({ data: undefined }),
  generatedArtifactOpenURL: () => '/generated/open',
}));

vi.mock('@/features/learning/components/ExplainReader', () => ({
  ExplainReader: ({ embedded }: { embedded?: boolean }) => <div data-testid="explain-contract">{embedded ? '内嵌多页正文' : '多页正文'}</div>,
}));

vi.mock('@/features/legacy-zones', () => ({
  PracticeFlow: () => <div data-testid="practice-contract">可作答题目与解析</div>,
}));

vi.mock('./BodyAnnotations', () => ({
  BodyAnnotations: () => <div data-testid="body-markdown">普通正文</div>,
}));

beforeEach(() => {
  vi.clearAllMocks();
  assets.body.contentKind = 'explain-pages';
  assets.body.content = '完整内容见 `explain/manifest.json`';
});

describe('AssetsView structured contracts', () => {
  it('renders migrated multipage body without exposing its manifest pointer', () => {
    render(<AssetsView slug="go语言" />);

    expect(screen.getByTestId('explain-contract')).toHaveTextContent('内嵌多页正文');
    expect(screen.queryByText(/manifest\.json/)).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: '编辑' })).not.toBeInTheDocument();
  });

  it('uses the interactive practice contract instead of displaying stored JSON', () => {
    render(<AssetsView slug="go语言" />);
    fireEvent.click(screen.getByRole('button', { name: '练习' }));

    expect(screen.getByTestId('practice-contract')).toHaveTextContent('可作答题目与解析');
    expect(screen.queryByText(/"tasks"/)).not.toBeInTheDocument();
  });

  it('presents a markdown body through the existing numbered page navigation', () => {
    assets.body.contentKind = 'markdown';
    assets.body.content = '# 正文\n\n导语\n\n## 第一节\n\n第一节内容\n\n## 第二节\n\n第二节内容';
    render(<AssetsView slug="kubernetes" />);

    const navigation = screen.getByRole('tablist', { name: '正文页面' });
    expect(navigation).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: '第一节' })).toHaveAttribute('aria-selected', 'true');
    fireEvent.click(screen.getByRole('button', { name: '下一页' }));
    expect(screen.getByRole('tab', { name: '第二节' })).toHaveAttribute('aria-selected', 'true');
  });
});
