import { render, screen } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { ProjectPage } from './ProjectPage';

const useProject = vi.fn();
const useHealth = vi.fn();

vi.mock('@/api/projects', () => ({ useProject: (...args: unknown[]) => useProject(...args) }));
vi.mock('@/api/health', () => ({ useHealth: () => useHealth() }));
vi.mock('@/components/feature/learning/LearningWorkspace', () => ({
  LearningWorkspace: () => <div>teacher-workspace</div>,
}));
vi.mock('@/components/feature/project/LegacyProjectView', () => ({
  LegacyProjectView: () => <div>legacy-reader</div>,
}));

const project = {
  slug: 'topic',
  title: '主题',
  projectType: 'system-learning',
  activeZone: 'Explain',
  generatedZones: [],
};

function renderPage() {
  render(
    <MemoryRouter initialEntries={['/project/topic']}>
      <Routes><Route path="/project/:id" element={<ProjectPage />} /></Routes>
    </MemoryRouter>,
  );
}

describe('ProjectPage migration cutover', () => {
  beforeEach(() => {
    useProject.mockReturnValue({ data: { project }, isLoading: false, error: null });
  });

  it('uses the legacy reader only for a project whose migration failed', () => {
    useHealth.mockReturnValue({
      data: { learningWorkspace: { mode: 'legacy', failedProjects: ['topic'] } },
      isLoading: false,
    });
    renderPage();
    expect(screen.getByText('legacy-reader')).toBeInTheDocument();
  });

  it('uses the teacher workspace for a migrated learning unit', () => {
    useHealth.mockReturnValue({
      data: { learningWorkspace: { mode: 'legacy', failedProjects: ['another-topic'] } },
      isLoading: false,
    });
    renderPage();
    expect(screen.getByText('teacher-workspace')).toBeInTheDocument();
  });

  it('treats a legacy null failed-project list as empty', () => {
    useHealth.mockReturnValue({
      data: { learningWorkspace: { mode: 'teacher', failedProjects: null } },
      isLoading: false,
    });
    renderPage();
    expect(screen.getByText('teacher-workspace')).toBeInTheDocument();
  });
});
