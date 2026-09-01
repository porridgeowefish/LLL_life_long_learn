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

  it('allows system learning without goal fields and leaves unknown levels explicit', async () => {
    createMutate.mockResolvedValueOnce({ project: { slug: 'game-theory' } });
    render(<CreateProjectModal open onOpenChange={vi.fn()} />);

    fireEvent.click(screen.getByRole('radio', { name: /系统学习/ }));
    fireEvent.change(screen.getByLabelText(/项目标题/), { target: { value: '博弈论' } });

    expect(screen.getByRole('radio', { name: '暂不确定' })).toBeChecked();
    expect(screen.getByRole('radio', { name: '暂不设置' })).toBeChecked();
    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: '创建项目' }));
    });

    expect(createMutate.mock.calls.at(-1)?.[0]).toEqual({
      title: '博弈论',
      projectType: 'system-learning',
      why: '',
      current: '暂不确定',
      target: '暂不设置',
      standard: '',
    });
  });

  it('fills editable learning goals from a template', () => {
    render(
      <CreateProjectModal
        open
        onOpenChange={vi.fn()}
        fromDisciplineMap
        fixedProjectType="system-learning"
        initialTitle="静态完全信息博弈"
      />,
    );

    fireEvent.click(screen.getByRole('button', { name: '解决实际问题' }));

    expect(screen.getByLabelText('为什么学这个？')).toHaveValue(
      '希望把「静态完全信息博弈」用于实际问题，能够识别问题并选择合适的方法。',
    );
    expect(screen.getByRole('radio', { name: '能上手用起来' })).toBeChecked();
    expect(screen.getByLabelText('完成标准')).toHaveValue(
      '能运用「静态完全信息博弈」完成一个实际练习，并说明方法选择和结果。',
    );
  });

  it('submits the canonical map topic reference for backend scope snapshotting', async () => {
    createMutate.mockResolvedValueOnce({ project: { slug: 'classical-mechanics' } });
    render(
      <CreateProjectModal
        open
        onOpenChange={vi.fn()}
        fromDisciplineMap
        fixedProjectType="system-learning"
        initialTitle="经典力学"
        initialScopeSource={{
          type: 'discipline-map',
          mapSlug: 'physics',
          topicId: 'classical-mechanics',
        }}
      />,
    );

    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: '确认创建' }));
    });

    expect(createMutate.mock.calls.at(-1)?.[0]).toEqual(expect.objectContaining({
      title: '经典力学',
      projectType: 'system-learning',
      scopeSource: {
        type: 'discipline-map',
        mapSlug: 'physics',
        topicId: 'classical-mechanics',
      },
    }));
  });
});
