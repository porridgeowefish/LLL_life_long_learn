import { useEffect, useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';

import { useCreateProject } from '@/api/projects';
import { Modal } from '@/components/primitive/Modal';
import { Button } from '@/components/primitive/Button';
import { Icon } from '@/components/primitive/Icon';
import type { ProjectType } from '@/types/domain';
import { ProjectTypeAdvisorDialog } from './ProjectTypeAdvisorDialog';

import s from './CreateProjectModal.module.css';

// Form schema — title is always required; why and the learning ladders apply
// only to system-learning projects.
// "当前水平" describes exposure (where the learner is now); "目标水平" describes
// mastery (where they want to land). Two distinct ladders — previously shared,
// which made the two selectors render identical options.
const CURRENT_LEVELS = ['未接触', '了解概念', '动手做过', '能独立完成'] as const;
const TARGET_LEVELS = ['看懂原理', '能上手用起来', '能独立解决问题', '融会贯通·能教别人'] as const;

const schema = z.object({
  projectType: z.enum(['discipline-map', 'system-learning'], {
    required_error: '请选择学科地图或系统学习',
    invalid_type_error: '请选择学科地图或系统学习',
  }),
  title: z.string().trim().min(1, '请填写项目标题'),
  why: z.string().trim().optional().default(''),
  current: z.enum(CURRENT_LEVELS).default('未接触'),
  target: z.enum(TARGET_LEVELS).default('融会贯通·能教别人'),
  standard: z.string().trim().optional().default(''),
}).superRefine((values, context) => {
  if (values.projectType === 'system-learning' && !values.why.trim()) {
    context.addIssue({
      code: z.ZodIssueCode.custom,
      path: ['why'],
      message: '请填写“为什么学这个”',
    });
  }
});

type FormValues = z.infer<typeof schema>;

interface CreateProjectModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCreated?: (projectSlug: string) => void;
  fromDisciplineMap?: boolean;
  fixedProjectType?: ProjectType;
  initialTitle?: string;
}

export function CreateProjectModal({
  open,
  onOpenChange,
  onCreated,
  fromDisciplineMap = false,
  fixedProjectType,
  initialTitle = '',
}: CreateProjectModalProps) {
  const create = useCreateProject();
  const [advisorOpen, setAdvisorOpen] = useState(false);
  const {
    register,
    handleSubmit,
    reset,
    getValues,
    setValue,
    watch,
    formState: { errors, isSubmitting },
  } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      title: initialTitle,
      projectType: fixedProjectType,
      why: '',
      current: '未接触',
      target: '融会贯通·能教别人',
      standard: '',
    },
  });

  // Re-seed on open so a topic selected from the map is actually prefilled.
  useEffect(() => {
    if (open) {
      reset({
        title: initialTitle,
        projectType: fixedProjectType,
        why: '',
        current: '未接触',
        target: '融会贯通·能教别人',
        standard: '',
      });
    } else {
      setAdvisorOpen(false);
    }
  }, [fixedProjectType, initialTitle, open, reset]);

  const onSubmit = handleSubmit(async (values) => {
    const projectType = fromDisciplineMap ? 'system-learning' : values.projectType;
    const payload = projectType === 'discipline-map'
      ? { title: values.title, projectType }
      : { ...values, projectType };
    const res = await create.mutateAsync(payload);
    onOpenChange(false);
    onCreated?.(res.project.slug);
  });

  const selectedType = watch('projectType');

  const formId = fromDisciplineMap ? 'deep-dive-project-form' : 'create-project-form';
  const currentDraft = getValues();

  return (
    <Modal
      open={open}
      onOpenChange={onOpenChange}
      title={fromDisciplineMap ? '创建深入学习项目' : '新建项目'}
      description={fromDisciplineMap ? '创建一个普通、平级的系统学习项目；之后仍可在侧栏自由分类' : '先选择当前需要的是建立地图，还是进入系统学习'}
      size="md"
      footer={(
        <>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            取消
          </Button>
          <Button
            type="submit"
            form={formId}
            variant="primary"
            loading={isSubmitting || create.isPending}
          >
            {fromDisciplineMap ? '确认创建' : '创建项目'}
          </Button>
        </>
      )}
    >
      <form id={formId} className={s.form} onSubmit={onSubmit}>
        {!fixedProjectType && (
          <div className={s.field}>
            <div className={s.typeHeader}>
              <label>项目形态 <span className={s.req}>*</span></label>
              <button
                type="button"
                className={s.advisorTrigger}
                onClick={() => setAdvisorOpen(true)}
                aria-label="问问 AI，帮我选择项目形态"
              >
                <Icon name="bot" size={15} />
                <span>问问 AI</span>
                <small>选择顾问</small>
              </button>
            </div>
            <div className={s.typeGrid}>
              <label className={s.typeCard}>
                <input type="radio" value="discipline-map" {...register('projectType')} />
                <strong>学科地图</strong>
                <span>先全面了解一门学科的边界、领域与路线。</span>
              </label>
              <label className={s.typeCard}>
                <input type="radio" value="system-learning" {...register('projectType')} />
                <strong>系统学习</strong>
                <span>围绕具体主题进入引入、讲解、练习、拓展和总结。</span>
              </label>
            </div>
            {errors.projectType && <div className={s.error}>{errors.projectType.message}</div>}
          </div>
        )}
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

        {selectedType === 'system-learning' && (
          <div className={s.learningFields}>
            <div className={s.field}>
              <label htmlFor="cp-why">
                为什么学这个？ <span className={s.req}>*</span>
              </label>
              <textarea
                id="cp-why"
                rows={3}
                placeholder="一句话说明动机。例如：希望理解统计推断，能判断现实数据中的不确定性。"
                {...register('why')}
              />
              <div className={s.hint}>这段背景会成为学习智能体理解你目标的上下文。</div>
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
                placeholder='怎样的成果算“学完”？例如：能独立解释三条核心规则并完成一个小练习。'
                {...register('standard')}
              />
              <div className={s.hint}>可选；用于判断何时进入下一阶段。</div>
            </div>
          </div>
        )}

        {create.error && (
          <div className={s.error}>创建失败：{(create.error as Error).message}</div>
        )}

      </form>
      {!fixedProjectType && (
        <ProjectTypeAdvisorDialog
          open={advisorOpen}
          onOpenChange={setAdvisorOpen}
          draft={{
            title: currentDraft.title,
            why: currentDraft.why,
            current: currentDraft.current,
            target: currentDraft.target,
            standard: currentDraft.standard,
          }}
          onAdopt={(projectType) => setValue('projectType', projectType, { shouldValidate: true })}
        />
      )}
    </Modal>
  );
}
