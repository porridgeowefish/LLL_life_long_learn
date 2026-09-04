import { fireEvent, render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { SourcesView } from './SourcesView';

const apiMocks = vi.hoisted(() => ({
  upload: vi.fn(),
  tombstone: vi.fn(),
  permanent: vi.fn(),
}));

vi.mock('@/features/learning/api/learningWorkspace', () => ({
  useSources: () => ({ data: [{ sourceId: 'source_ready', displayName: 'notes.pdf', status: 'ready', updatedAt: '2026-09-04T00:00:00Z' }], isLoading: false }),
  useUploadSource: () => ({ mutate: apiMocks.upload, isPending: false, isError: false }),
  useTombstoneSource: () => ({ mutate: apiMocks.tombstone }),
  usePermanentlyDeleteSource: () => ({ mutate: apiMocks.permanent }),
}));

describe('SourcesView upload confirmation', () => {
  beforeEach(() => {
    apiMocks.upload.mockReset();
    apiMocks.tombstone.mockReset();
    apiMocks.permanent.mockReset();
  });

  it('only enables parsing after the learner accepts the cloud disclosure', () => {
    const { container } = render(<SourcesView slug="calculus" />);
    const input = container.querySelector('input[type="file"]') as HTMLInputElement;
    const file = new File(['content'], 'textbook.pdf', { type: 'application/pdf' });

    fireEvent.change(input, { target: { files: [file] } });
    const parse = screen.getByRole('button', { name: '保存并解析' });
    fireEvent.click(screen.getByRole('checkbox', { name: /我了解并同意/ }));
    expect(parse).toBeEnabled();

    fireEvent.click(parse);
    expect(apiMocks.upload).toHaveBeenCalledWith(
      { file, parseApproved: true, cloudDisclosureAccepted: true },
      expect.objectContaining({ onSuccess: expect.any(Function) }),
    );
  });

  it('shows only source file records without parsed previews or assistant assets', () => {
    render(<SourcesView slug="calculus" />);
    expect(screen.getByText('notes.pdf')).toBeInTheDocument();
    expect(screen.queryByText('查看解析内容')).not.toBeInTheDocument();
    expect(screen.queryByText('助教生成资料')).not.toBeInTheDocument();
  });
});
