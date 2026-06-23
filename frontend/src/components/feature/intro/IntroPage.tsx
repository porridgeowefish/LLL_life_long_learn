import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { ArrowRightIcon } from '@radix-ui/react-icons';

import { useIntroAssessment, type PrerequisiteAssessment } from '@/api/learningArtifacts';
import { useCreateSubproject } from '@/api/projects';
import { Button } from '@/components/primitive/Button';
import { Modal } from '@/components/primitive/Modal';
import { OutputViewer } from '@/components/feature/project/OutputViewer';

import s from './IntroPage.module.css';

interface IntroPageProps {
  projectSlug: string;
}

const LEVEL_LABEL: Record<string, string> = {
  unknown: '尚未确认',
  none: '未接触',
  basic: '基础了解',
  working: '可以运用',
  solid: '掌握扎实',
};

const STATUS_LABEL: Record<string, string> = {
  ready: '已就绪',
  weak: '建议补齐',
  missing: '需要补齐',
};

export function IntroPage({ projectSlug }: IntroPageProps) {
  const navigate = useNavigate();
  const assessment = useIntroAssessment(projectSlug);
  const createSubproject = useCreateSubproject(projectSlug);
  const [selected, setSelected] = useState<PrerequisiteAssessment | null>(null);

  const gaps = assessment.data?.prerequisites?.filter(
    (item) => item.status === 'weak' || item.status === 'missing',
  ) ?? [];

  const handleCreate = async () => {
    if (!selected) return;
    const result = await createSubproject.mutateAsync(selected.projectDraft);
    setSelected(null);
    navigate(`/project/${result.project.slug}/Intro`);
  };

  return (
    <div className={s.layout}>
      <OutputViewer slug={projectSlug} zone="Intro" />

      {assessment.data && (
        <section className={s.assessment}>
          <div className={s.assessmentHead}>
            <div className={s.headingGroup}>
              <span className={s.eyebrow}>学习起点</span>
              <h2>前置知识诊断</h2>
            </div>
            <div className={s.count}>
              <strong>{gaps.length}</strong>
              <span>个待补缺口</span>
            </div>
          </div>

          <div className={s.baseline}>
            <span>当前背景</span>
            <p>{assessment.data.baseline}</p>
          </div>

          <div className={s.grid}>
            {assessment.data.prerequisites?.map((item, index) => (
              <article key={item.id} className={`${s.card} ${s[item.status]}`}>
                <div className={s.cardLead}>
                  <span className={s.index}>{String(index + 1).padStart(2, '0')}</span>
                  <div className={s.titleGroup}>
                    <h3>{item.title}</h3>
                    <div className={s.meta}>
                      <span className={s.status}>{STATUS_LABEL[item.status]}</span>
                      <span className={s.level}>{LEVEL_LABEL[item.assessedLevel]}</span>
                    </div>
                  </div>
                </div>

                <div className={s.cardBody}>
                  <p className={s.impact}>{item.impact}</p>
                  <p className={s.evidence}>
                    <span>判断依据</span>
                    {item.evidence}
                  </p>
                </div>

                <div className={s.cardAction}>
                  {item.status === 'weak' || item.status === 'missing' ? (
                    <Button
                      size="sm"
                      variant={item.status === 'missing' ? 'primary' : 'outline'}
                      iconRight={<ArrowRightIcon aria-hidden="true" />}
                      onClick={() => setSelected(item)}
                    >
                      单独学习
                    </Button>
                  ) : (
                    <span className={s.readyNote}>可直接进入讲解</span>
                  )}
                </div>
              </article>
            ))}
          </div>
        </section>
      )}

      <Modal
        open={!!selected}
        onOpenChange={(open) => { if (!open) setSelected(null); }}
        title="创建前置学习项目"
        description="将作为当前项目的子项目创建，内容已经根据诊断结果预填。"
        footer={
          <>
            <Button variant="outline" onClick={() => setSelected(null)}>取消</Button>
            <Button variant="primary" onClick={handleCreate} loading={createSubproject.isPending}>
              创建并进入
            </Button>
          </>
        }
      >
        {selected && (
          <div className={s.preview}>
            <strong>{selected.projectDraft.title}</strong>
            <p>{selected.projectDraft.why}</p>
            <dl>
              <dt>当前水平</dt><dd>{selected.projectDraft.current}</dd>
              <dt>目标水平</dt><dd>{selected.projectDraft.target}</dd>
              <dt>完成标准</dt><dd>{selected.projectDraft.standard}</dd>
            </dl>
          </div>
        )}
      </Modal>
    </div>
  );
}
