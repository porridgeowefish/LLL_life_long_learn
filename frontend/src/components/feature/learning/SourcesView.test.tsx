import { fireEvent, render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { SourcesView } from './SourcesView';

const apiMocks = vi.hoisted(() => ({
  upload: vi.fn(),
  tombstone: vi.fn(),
  permanent: vi.fn(),
}));

vi.mock('@/api/learningWorkspace', () => ({
  useSources: () => ({ data: [], isLoading: false }),
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

  it('saves an original without granting cloud parsing permission', () => {
    const { container } = render(<SourcesView slug="calculus" />);
    const input = container.querySelector('input[type="file"]') as HTMLInputElement;
    const file = new File(['derivative'], 'notes.md', { type: 'text/markdown' });

    fireEvent.change(input, { target: { files: [file] } });
    expect(screen.getByRole('dialog', { name: '确认资料处理' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '保存并解析' })).toBeDisabled();

    fireEvent.click(screen.getByRole('button', { name: '仅保存原件' }));
    expect(apiMocks.upload).toHaveBeenCalledWith(
      { file, parseApproved: false, cloudDisclosureAccepted: false },
      expect.objectContaining({ onSuccess: expect.any(Function) }),
    );
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
});
