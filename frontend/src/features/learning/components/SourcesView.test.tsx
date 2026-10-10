import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { SourcesView } from './SourcesView';

const apiMocks = vi.hoisted(() => ({
  upload: vi.fn(),
  tombstone: vi.fn(),
  permanent: vi.fn(),
  retry: vi.fn(),
  sources: [{ sourceId: 'source_ready', displayName: 'notes.pdf', status: 'ready', updatedAt: '2026-09-04T00:00:00Z' }] as Array<Record<string, unknown>>,
}));

vi.mock('@/features/learning/api/learningWorkspace', () => ({
  MAX_SOURCE_FILE_BYTES: 50 * 1024 * 1024,
  useSources: () => ({ data: apiMocks.sources, isLoading: false }),
  useUploadSource: () => ({ mutateAsync: apiMocks.upload, isPending: false, isError: false }),
  useTombstoneSource: () => ({ mutate: apiMocks.tombstone }),
  usePermanentlyDeleteSource: () => ({ mutate: apiMocks.permanent }),
  useRetrySource: () => ({ mutate: apiMocks.retry, isPending: false }),
}));

describe('SourcesView local imports', () => {
  beforeEach(() => {
    apiMocks.upload.mockReset();
    apiMocks.tombstone.mockReset();
    apiMocks.permanent.mockReset();
    apiMocks.retry.mockReset();
    apiMocks.sources = [{ sourceId: 'source_ready', displayName: 'notes.pdf', status: 'ready', updatedAt: '2026-09-04T00:00:00Z' }];
  });

  it('imports multiple files immediately and reports partial failure', async () => {
    apiMocks.upload.mockRejectedValueOnce(new Error('文件已损坏')).mockResolvedValueOnce({ source: { status: 'processing' } });
    const { container } = render(<SourcesView slug="calculus" />);
    const input = container.querySelector('input[type="file"]') as HTMLInputElement;
    const broken = new File(['content'], 'broken.pdf', { type: 'application/pdf' });
    const good = new File(['# note'], 'note.md', { type: 'text/markdown' });

    fireEvent.change(input, { target: { files: [broken, good] } });
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    await waitFor(() => expect(screen.getByRole('status')).toHaveTextContent('已保存 1 份 · 1 份导入失败'));
    expect(screen.getByText('文件已损坏')).toBeInTheDocument();
    expect(screen.getByText('已保存，正在自动解析')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '导入资料' })).toBeEnabled();
    expect(apiMocks.upload).toHaveBeenNthCalledWith(1, { file: broken });
    expect(apiMocks.upload).toHaveBeenNthCalledWith(2, { file: good });
  });

  it('rejects oversized files and explains unsupported old Word format', async () => {
    const { container } = render(<SourcesView slug="calculus" />);
    const file = new File(['content'], 'too-large.pdf');
    Object.defineProperty(file, 'size', { value: 50 * 1024 * 1024 + 1 });
    fireEvent.change(container.querySelector('input[type="file"]')!, { target: { files: [file, new File(['old'], 'legacy.doc')] } });
    expect(await screen.findByText('文件超过 50 MB，请拆分后导入')).toBeInTheDocument();
    expect(screen.getByText('请另存为 DOCX 后导入')).toBeInTheDocument();
    expect(apiMocks.upload).not.toHaveBeenCalled();
  });

  it('explains failed extraction and makes truncated extraction visible', () => {
    apiMocks.sources = [{ sourceId: 'empty', displayName: 'scan.pdf', status: 'failed', failureCode: 'no_extractable_text', updatedAt: '' }, { sourceId: 'long', displayName: 'large.docx', status: 'ready', extractionTruncated: true, updatedAt: '' }];
    render(<SourcesView slug="calculus" />);
    expect(screen.getByText(/无可提取文字，扫描件暂不支持识别/)).toBeInTheDocument();
    expect(screen.getByText(/仅提取了部分内容/)).toBeInTheDocument();
  });

  it('retries the selected failed or formerly opaque supported document', () => {
    apiMocks.sources = [
      { sourceId: 'failed-pdf', displayName: 'failed.pdf', status: 'failed', failureCode: 'parse_cancelled', updatedAt: '' },
      { sourceId: 'old-docx', displayName: 'old.docx', status: 'opaque', updatedAt: '' },
      { sourceId: 'legacy-doc', displayName: 'legacy.doc', status: 'failed', failureCode: 'unsupported_legacy_format', updatedAt: '' },
    ];
    render(<SourcesView slug="calculus" />);
    fireEvent.click(screen.getByRole('button', { name: '重试解析 failed.pdf' }));
    expect(apiMocks.retry).toHaveBeenCalledWith('failed-pdf');
    fireEvent.click(screen.getByRole('button', { name: '重试解析 old.docx' }));
    expect(apiMocks.retry).toHaveBeenCalledWith('old-docx');
    expect(screen.queryByRole('button', { name: '重试解析 legacy.doc' })).not.toBeInTheDocument();
    expect(screen.getByText(/旧版 DOC 暂不支持，请另存为 DOCX/)).toBeInTheDocument();
  });

  it('shows only source file records without parsed previews or assistant assets', () => {
    render(<SourcesView slug="calculus" />);
    expect(screen.getByText('notes.pdf')).toBeInTheDocument();
    expect(screen.queryByText('查看解析内容')).not.toBeInTheDocument();
    expect(screen.queryByText('助教生成资料')).not.toBeInTheDocument();
  });
});
