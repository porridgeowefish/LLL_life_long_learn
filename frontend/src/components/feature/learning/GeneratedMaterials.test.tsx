import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import { GeneratedMaterials } from './GeneratedMaterials';

vi.mock('@/api/learningWorkspace', () => ({
  useGeneratedArtifacts: () => ({ data: [{
    artifactId: 'artifact_summary', kind: 'summary', title: '分布式计算总结',
    description: '对话沉淀的核心概念', entryPoint: 'files/summary.md',
    entryMediaType: 'text/markdown', fileCount: 1, createdAt: '2026-09-02T00:00:00Z',
  }], isLoading: false }),
  useGeneratedArtifactEntry: () => ({ data: '# 一致性与可用性', isLoading: false }),
  generatedArtifactOpenURL: () => '/api/generated/artifact_summary/open',
}));

describe('GeneratedMaterials', () => {
  it('renders a committed assistant deliverable as readable material', () => {
    render(<GeneratedMaterials slug="distributed-systems" />);
    expect(screen.getByRole('complementary', { name: '助教生成资料' })).toHaveTextContent('分布式计算总结');
    expect(screen.getByRole('heading', { name: '一致性与可用性' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: '打开原始成果' })).toHaveAttribute('href', '/api/generated/artifact_summary/open');
  });
});
