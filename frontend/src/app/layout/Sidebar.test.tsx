import { fireEvent, render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { Sidebar } from './Sidebar';

const deleteProject = vi.hoisted(() => vi.fn());

vi.mock('@/features/projects/api/projects', () => ({
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

vi.mock('@/features/projects/api/folders', () => ({
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

vi.mock('@/shared/store', () => ({
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
  beforeEach(() => deleteProject.mockReset());

  it('renders a map as the folder overview and not as a child project row', () => {
    render(<MemoryRouter><Sidebar /></MemoryRouter>);

    const overviewLink = screen.getByRole('link', { name: /概率论与数理统计学科总览/ });
    expect(overviewLink.getAttribute('href')).toBe('/project/probability');
    expect(screen.getByRole('link', { name: /蒙特卡洛模拟/ })).toBeTruthy();
    expect(screen.getByText('总览')).toBeTruthy();
  });

  it('calls the delete command only after the in-app confirmation', () => {
    render(<MemoryRouter><Sidebar /></MemoryRouter>);

    fireEvent.click(screen.getByRole('button', { name: '项目操作：蒙特卡洛模拟' }));
    expect(screen.getByText('移动到文件夹')).toBeTruthy();
    fireEvent.click(screen.getByRole('menuitem', { name: '删除学习单元' }));

    expect(deleteProject).not.toHaveBeenCalled();
    expect(screen.getByRole('heading', { name: '删除学习单元？' })).toBeTruthy();
    fireEvent.click(screen.getByRole('button', { name: '确认删除' }));
    expect(deleteProject).toHaveBeenCalledWith('monte-carlo', expect.any(Object));
  });

  it('does not call the delete command when the dialog is cancelled', () => {
    render(<MemoryRouter><Sidebar /></MemoryRouter>);

    fireEvent.click(screen.getByRole('button', { name: '项目操作：蒙特卡洛模拟' }));
    fireEvent.click(screen.getByRole('menuitem', { name: '删除学习单元' }));
    fireEvent.click(screen.getByRole('button', { name: '取消' }));

    expect(deleteProject).not.toHaveBeenCalled();
  });

  it('confirms deletion of a discipline project through the same in-app surface', () => {
    render(<MemoryRouter><Sidebar /></MemoryRouter>);

    fireEvent.click(screen.getByRole('button', { name: '项目操作：概率论与数理统计学科总览' }));
    fireEvent.click(screen.getByRole('menuitem', { name: '删除学科项目' }));

    expect(screen.getByRole('heading', { name: '删除学科项目？' })).toBeTruthy();
    fireEvent.click(screen.getByRole('button', { name: '确认删除' }));
    expect(deleteProject).toHaveBeenCalledWith('probability', expect.any(Object));
  });
});
