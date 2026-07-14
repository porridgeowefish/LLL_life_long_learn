import { fireEvent, render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { describe, expect, it, vi } from 'vitest';

import { Sidebar } from './Sidebar';

const deleteProject = vi.hoisted(() => vi.fn());

vi.mock('@/api/projects', () => ({
  useDeleteProject: () => ({
    mutate: deleteProject,
    isError: false,
  }),
  useProjects: () => ({
    isLoading: false,
    data: {
      projects: [
        { slug: 'probability', title: '概率论与数理统计', projectType: 'discipline-map' },
        { slug: 'monte-carlo', title: '蒙特卡洛模拟', projectType: 'system-learning' },
      ],
    },
  }),
}));

vi.mock('@/api/folders', () => ({
  useFolders: () => ({
    data: {
      folders: [{
        id: 'math',
        name: '概率论与数理统计',
        mapProjectSlug: 'probability',
        slugOrder: ['monte-carlo'],
      }],
    },
  }),
  useSaveFolders: () => ({ mutate: vi.fn(), isPending: false, isError: false }),
}));

vi.mock('@/store', () => ({
  useUiStore: (selector: (state: object) => unknown) => selector({
    collapsedFolderIds: [],
    toggleFolderCollapsed: vi.fn(),
  }),
  useProjectStore: (selector: (state: object) => unknown) => selector({
    selectedProjectId: null,
    selectProject: vi.fn(),
  }),
}));

describe('Sidebar discipline maps', () => {
  it('renders a map as the folder overview and not as a child project row', () => {
    render(<MemoryRouter><Sidebar /></MemoryRouter>);

    const overviewLink = screen.getByRole('link', { name: /概率论与数理统计/ });
    expect(overviewLink.getAttribute('href')).toBe('/project/probability');
    expect(screen.getByRole('link', { name: /蒙特卡洛模拟/ })).toBeTruthy();
    expect(screen.queryByText('地图')).toBeNull();
  });

  it('calls the delete command only after one confirmation', () => {
    const confirm = vi.spyOn(window, 'confirm').mockReturnValue(true);
    render(<MemoryRouter><Sidebar /></MemoryRouter>);

    fireEvent.click(screen.getByRole('button', { name: /项目操作/ }));
    fireEvent.click(screen.getByRole('menuitem', { name: '删除学习' }));

    expect(confirm).toHaveBeenCalledTimes(1);
    expect(deleteProject).toHaveBeenCalledWith('monte-carlo', expect.any(Object));
    confirm.mockRestore();
  });

  it('does not call the delete command when confirmation is cancelled', () => {
    deleteProject.mockClear();
    const confirm = vi.spyOn(window, 'confirm').mockReturnValue(false);
    render(<MemoryRouter><Sidebar /></MemoryRouter>);

    fireEvent.click(screen.getByRole('button', { name: /项目操作/ }));
    fireEvent.click(screen.getByRole('menuitem', { name: '删除学习' }));

    expect(confirm).toHaveBeenCalledTimes(1);
    expect(deleteProject).not.toHaveBeenCalled();
    confirm.mockRestore();
  });
});
