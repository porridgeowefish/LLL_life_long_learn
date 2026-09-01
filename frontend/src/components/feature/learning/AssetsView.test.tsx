import { fireEvent, render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { AssetsView } from './AssetsView';

const assets = {
  intro: { contentKind: 'markdown', content: '# 引入' },
  body: { contentKind: 'explain-pages', content: '完整内容见 `explain/manifest.json`' },
  practice: { contentKind: 'practice-set', content: '```json\n{"tasks":[]}\n```' },
} as const;

vi.mock('@/api/learningWorkspace', () => ({
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

vi.mock('@/components/feature/explain/ExplainReader', () => ({
  ExplainReader: ({ embedded }: { embedded?: boolean }) => <div data-testid="explain-contract">{embedded ? '内嵌多页正文' : '多页正文'}</div>,
}));

vi.mock('@/components/feature/practice/PracticeFlow', () => ({
  PracticeFlow: () => <div data-testid="practice-contract">可作答题目与解析</div>,
}));

vi.mock('./BodyAnnotations', () => ({
  BodyAnnotations: () => <div data-testid="body-markdown">普通正文</div>,
}));

beforeEach(() => vi.clearAllMocks());

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
});
