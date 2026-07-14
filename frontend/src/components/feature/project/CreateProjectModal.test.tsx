import { act, fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import { CreateProjectModal } from './CreateProjectModal';

const createMutate = vi.fn();
const adviceMutate = vi.fn();

vi.mock('@/api/projects', () => ({
  useCreateProject: () => ({ mutateAsync: createMutate, error: null, isPending: false }),
  useProjectTypeAdvice: () => ({ mutateAsync: adviceMutate, reset: vi.fn(), error: null, isPending: false }),
}));

vi.mock('@/components/primitive/Modal', () => ({
  Modal: ({ open, children, footer }: { open: boolean; children: React.ReactNode; footer?: React.ReactNode }) =>
    open ? <div>{children}{footer}</div> : null,
}));

describe('CreateProjectModal', () => {
  it('requires an explicit visible project type choice', async () => {
    render(<CreateProjectModal open onOpenChange={vi.fn()} />);

    expect(screen.getByRole('radio', { name: /学科地图/ })).not.toBeChecked();
    expect(screen.getByRole('radio', { name: /系统学习/ })).not.toBeChecked();
    expect(screen.getByRole('button', { name: '问问 AI，帮我选择项目形态' })).toBeTruthy();

    fireEvent.change(screen.getByLabelText(/项目标题/), { target: { value: '博弈论' } });
    fireEvent.click(screen.getByRole('button', { name: '创建项目' }));

    expect(await screen.findByText('请选择学科地图或系统学习')).toBeTruthy();
    expect(createMutate).not.toHaveBeenCalled();
  });

  it('keeps discipline-map creation short and does not require learning-level fields', async () => {
    createMutate.mockResolvedValueOnce({ project: { slug: 'game-theory' } });
    const onOpenChange = vi.fn();
    render(<CreateProjectModal open onOpenChange={onOpenChange} />);

    fireEvent.click(screen.getByRole('radio', { name: /学科地图/ }));
    fireEvent.change(screen.getByLabelText(/项目标题/), { target: { value: '博弈论' } });

    expect(screen.queryByLabelText(/为什么学这个/)).toBeNull();
    expect(screen.queryByText('当前水平')).toBeNull();
    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: '创建项目' }));
    });

    expect(onOpenChange).toHaveBeenCalledWith(false);
    expect(createMutate.mock.calls.at(-1)?.[0]).toEqual({
      title: '博弈论',
      projectType: 'discipline-map',
    });
  });

  it('re-seeds the title when a map topic opens the form', () => {
    const onOpenChange = vi.fn();
    const { rerender } = render(
      <CreateProjectModal open={false} onOpenChange={onOpenChange} fromDisciplineMap fixedProjectType="system-learning" />,
    );

    rerender(
      <CreateProjectModal
        open
        onOpenChange={onOpenChange}
        fromDisciplineMap
        fixedProjectType="system-learning"
        initialTitle="经典力学"
      />,
    );

    expect((screen.getByLabelText(/项目标题/) as HTMLInputElement).value).toBe('经典力学');
  });
});
