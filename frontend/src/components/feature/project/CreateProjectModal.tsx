import { useEffect } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';

import { useCreateProject } from '@/api/projects';
import { Modal } from '@/components/primitive/Modal';
import { Button } from '@/components/primitive/Button';

import s from './CreateProjectModal.module.css';

// Form schema — title + why are required, current/target default to endpoints.
// "当前水平" describes exposure (where the learner is now); "目标水平" describes
// mastery (where they want to land). Two distinct ladders — previously shared,
// which made the two selectors render identical options.
const CURRENT_LEVELS = ['未接触', '了解概念', '动手做过', '能独立完成'] as const;
const TARGET_LEVELS = ['看懂原理', '能上手用起来', '能独立解决问题', '融会贯通·能教别人'] as const;

const schema = z.object({
  title: z.string().trim().min(1, '请填写项目标题'),
  why: z.string().trim().min(1, '请填写"为什么学这个"'),
  current: z.enum(CURRENT_LEVELS).default('未接触'),
  target: z.enum(TARGET_LEVELS).default('融会贯通·能教别人'),
  standard: z.string().trim().optional().default(''),
});

type FormValues = z.infer<typeof schema>;

interface CreateProjectModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCreated?: (projectSlug: string) => void;
}

export function CreateProjectModal({ open, onOpenChange, onCreated }: CreateProjectModalProps) {
  const create = useCreateProject();
  const {
    register,
    handleSubmit,
    reset,
    formState: { errors, isSubmitting },
  } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      title: '',
      why: '',
      current: '未接触',
      target: '融会贯通·能教别人',
      standard: '',
    },
  });

  // Reset form whenever the modal closes — keep state clean for the next open.
  useEffect(() => {
    if (!open) reset();
  }, [open, reset]);

  const onSubmit = handleSubmit(async (values) => {
    const res = await create.mutateAsync(values);
    onOpenChange(false);
    onCreated?.(res.project.slug);
  });

  return (
    <Modal
      open={open}
      onOpenChange={onOpenChange}
      title="新建学习项目"
      description="填写背景信息让智能体调用时拥有更完整的上下文"
      size="md"
    >
      <form className={s.form} onSubmit={onSubmit}>
        <div className={s.field}>
          <label htmlFor="cp-title">
            项目标题 <span className={s.req}>*</span>
          </label>
          <input
            id="cp-title"
            type="text"
            placeholder="例如：Rust 所有权模型"
            autoComplete="off"
            {...register('title')}
          />
          <div className={s.hint}>用作项目目录名（自动 slugify），后续可改显示标题。</div>
          {errors.title && <div className={s.error}>{errors.title.message}</div>}
        </div>

        <div className={s.field}>
          <label htmlFor="cp-why">
            为什么学这个？ <span className={s.req}>*</span>
          </label>
          <textarea
            id="cp-why"
            rows={3}
            placeholder="一句话说明动机。例如：理解所有权是写好 Rust 的前提，避免在 borrow checker 前束手无策。"
            {...register('why')}
          />
          <div className={s.hint}>写入 project.md 的「Why this topic matters」章节，会成为智能体调用时的上下文。</div>
          {errors.why && <div className={s.error}>{errors.why.message}</div>}
        </div>

        <div className={s.fieldRow}>
          <div className={s.field}>
            <label>当前水平</label>
            <div className={s.radioRow}>
              {CURRENT_LEVELS.map((lvl) => (
                <label key={lvl} className={s.radio}>
                  <input type="radio" value={lvl} {...register('current')} />
                  <span>{lvl}</span>
                </label>
              ))}
            </div>
          </div>
        </div>

        <div className={s.fieldRow}>
          <div className={s.field}>
            <label>目标水平</label>
            <div className={s.radioRow}>
              {TARGET_LEVELS.map((lvl) => (
                <label key={lvl} className={s.radio}>
                  <input type="radio" value={lvl} {...register('target')} />
                  <span>{lvl}</span>
                </label>
              ))}
            </div>
          </div>
        </div>

        <div className={s.field}>
          <label htmlFor="cp-standard">完成标准</label>
          <textarea
            id="cp-standard"
            rows={2}
            placeholder='怎样的成果算"学完"？例如：能独立解释三条所有权规则并写出 5 行无编译错误的借用代码。'
            {...register('standard')}
          />
          <div className={s.hint}>可选；用于判断何时进入下一阶段。</div>
        </div>

        {create.error && (
          <div className={s.error}>创建失败：{(create.error as Error).message}</div>
        )}

        <div className={s.actions}>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            取消
          </Button>
          <Button type="submit" variant="primary" loading={isSubmitting}>
            创建项目
          </Button>
        </div>
      </form>
    </Modal>
  );
}
