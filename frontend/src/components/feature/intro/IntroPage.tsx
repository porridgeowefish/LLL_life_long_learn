import { useEffect, useMemo, useState } from 'react';
import { ClipboardCopyIcon } from '@radix-ui/react-icons';

import {
  useIntroAssessment,
  useIntroSurvey,
  useSaveIntroSurvey,
  type IntroSurvey,
  type IntroSurveyQuestion,
} from '@/api/learningArtifacts';
import { Button } from '@/components/primitive/Button';
import { EmptyState } from '@/components/primitive/EmptyState';
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
  weak: '学习时留意',
  missing: '需要先认识',
};

function buildSurveyPayload(questions: IntroSurveyQuestion[], answers: Record<string, string>): IntroSurvey {
  return {
    schemaVersion: 1,
    updatedAt: new Date().toISOString(),
    questions: questions.map((q) => ({
      ...q,
      answer: (answers[q.id] ?? q.answer ?? '').trim(),
    })),
  };
}

function buildCopyText(survey: IntroSurvey) {
  const body = survey.questions
    .map((q, index) => {
      const answer = (q.answer ?? '').trim() || '跳过 / 未获得证据';
      return `${index + 1}. ${q.label}\n问题：${q.prompt}\n我的回答：${answer}`;
    })
    .join('\n\n');

  return [
    '以下是我在 Intro 能力调查页填写的回答。请基于这些具体证据继续生成 intro/output.md 和 intro/assessment.json。',
    '规则：不知道或跳过的部分只记为“未获得证据”，不要推断为我已经掌握；请给出两个具体样例、为什么值得学、3 个入口问题、前置知识地图和一条学习路线建议。',
    '',
    body,
  ].join('\n');
}

export function IntroPage({ projectSlug }: IntroPageProps) {
  const assessment = useIntroAssessment(projectSlug);
  const survey = useIntroSurvey(projectSlug);
  const saveSurvey = useSaveIntroSurvey(projectSlug);
  const [answers, setAnswers] = useState<Record<string, string>>({});
  const [copyState, setCopyState] = useState<'idle' | 'copied' | 'failed'>('idle');

  const surveyQuestions = useMemo(() => {
    return survey.data?.questions ?? [];
  }, [survey.data]);

  useEffect(() => {
    if (!survey.data?.questions?.length) return;
    setAnswers(Object.fromEntries(survey.data.questions.map((q) => [q.id, q.answer ?? ''])));
  }, [survey.data]);

  const gaps = assessment.data?.prerequisites?.filter(
    (item) => item.status === 'weak' || item.status === 'missing',
  ) ?? [];
  const answeredCount = surveyQuestions.filter((q) => (answers[q.id] ?? q.answer ?? '').trim()).length;
  const hasSurvey = surveyQuestions.length > 0;

  const handleSaveSurvey = async () => {
    const payload = buildSurveyPayload(surveyQuestions, answers);
    await saveSurvey.mutateAsync(payload);
    return payload;
  };

  const handleCopySurvey = async () => {
    try {
      const payload = await handleSaveSurvey();
      await navigator.clipboard.writeText(buildCopyText(payload));
      setCopyState('copied');
      window.setTimeout(() => setCopyState('idle'), 2400);
    } catch {
      setCopyState('failed');
    }
  };

  const surveySection = (
    <section className={s.survey}>
      <div className={s.surveyHead}>
        <div className={s.headingGroup}>
          <span className={s.eyebrow}>Intro 调查</span>
          <h2>能力边界校准</h2>
        </div>
        <div className={s.surveyActions}>
          <span className={s.progressText}>
            {hasSurvey ? `${answeredCount}/${surveyQuestions.length} 已填写` : '等待生成调查页'}
          </span>
          <Button
            variant="outline"
            size="sm"
            onClick={handleSaveSurvey}
            loading={saveSurvey.isPending}
            disabled={!hasSurvey}
          >
            保存
          </Button>
          <Button
            variant="primary"
            size="sm"
            iconLeft={<ClipboardCopyIcon aria-hidden="true" />}
            onClick={handleCopySurvey}
            loading={saveSurvey.isPending}
            disabled={!hasSurvey}
          >
            复制给同一对话
          </Button>
        </div>
      </div>

      {hasSurvey ? (
        <div className={s.questionList}>
          {surveyQuestions.map((question, index) => (
            <label key={question.id} className={s.questionCard}>
              <span className={s.questionIndex}>{String(index + 1).padStart(2, '0')}</span>
              <span className={s.questionBody}>
                <strong>{question.label}</strong>
                <span>{question.prompt}</span>
                <textarea
                  value={answers[question.id] ?? question.answer ?? ''}
                  onChange={(e) => {
                    setCopyState('idle');
                    setAnswers((current) => ({ ...current, [question.id]: e.target.value }));
                  }}
                  placeholder="写你的回答；也可以写“跳过”或“不知道”。"
                  rows={4}
                />
              </span>
            </label>
          ))}
        </div>
      ) : (
        <EmptyState
          title="还没有调查页"
          description={
            <>
              调用 Intro Agent 后，它会先生成 <code>intro/survey.json</code>。这里会把题目渲染成可填写页面，而不是让你在命令行里回答。
            </>
          }
        />
      )}

      <div className={s.surveyFoot}>
        {saveSurvey.isError && <span className={s.errorText}>保存失败：{(saveSurvey.error as Error).message}</span>}
        {copyState === 'copied' && <span className={s.successText}>已保存并复制，可以贴回当前 Agent 对话。</span>}
        {copyState === 'failed' && <span className={s.errorText}>复制失败，请先保存后手动复制页面内容。</span>}
      </div>
    </section>
  );

  const introSection = <OutputViewer slug={projectSlug} zone="Intro" />;

  return (
    <div className={s.layout}>
      {introSection}

      {assessment.data && (
        <section className={s.assessment}>
          <div className={s.assessmentHead}>
            <div className={s.headingGroup}>
              <span className={s.eyebrow}>学习起点</span>
              <h2>前置知识诊断</h2>
            </div>
            <div className={s.count}>
              <strong>{gaps.length}</strong>
              <span>个需要留意</span>
            </div>
          </div>

          <div className={s.baseline}>
            <span>当前背景</span>
            <p>{assessment.data.baseline}</p>
          </div>

          <div className={s.grid}>
            {assessment.data.prerequisites?.map((item) => (
              <article key={item.id} className={`${s.card} ${s[item.status]}`}>
                <div className={s.cardLead}>
                  <div className={s.titleGroup}>
                    <h3>{item.title}</h3>
                    <div className={s.meta}>
                      <span className={s.status}>{STATUS_LABEL[item.status]}</span>
                      <span className={s.level}>{LEVEL_LABEL[item.assessedLevel]}</span>
                    </div>
                  </div>
                </div>

                <div className={s.cardBody}>
                  <p className={s.summary}>{item.summary?.trim() || item.impact}</p>
                  {item.summary?.trim() && (
                    <p className={s.impact}>
                      <span>与当前学习的关系</span>
                      {item.impact}
                    </p>
                  )}
                  <p className={s.evidence}>
                    <span>判断依据</span>
                    {item.evidence}
                  </p>
                </div>
              </article>
            ))}
          </div>
        </section>
      )}

      {surveySection}
    </div>
  );
}
