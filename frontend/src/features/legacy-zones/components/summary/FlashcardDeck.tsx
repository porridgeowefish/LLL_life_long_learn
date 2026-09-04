import { useCallback, useEffect, useMemo, useState } from 'react';
import clsx from 'clsx';

import {
  useFlashcards,
  useGradeFlashcard,
  type CardProgress,
  type Flashcard,
  type Grade,
} from '@/features/legacy-zones/api/flashcards';
import { useMarkdown } from '@/shared/hooks/useMarkdown';
import { EmptyState } from '@/shared/primitive/EmptyState';

import s from './FlashcardDeck.module.css';

const GRADES: { value: Grade; label: string; hint: string; tone: string }[] = [
  { value: 'forgot', label: '忘记了', hint: '重新学习', tone: s.gradeForgot },
  { value: 'fuzzy', label: '有点模糊', hint: '尽快再看', tone: s.gradeFuzzy },
  { value: 'got-it', label: '想起来了', hint: '正常间隔', tone: s.gradeGotIt },
  { value: 'easy', label: '非常清楚', hint: '延长间隔', tone: s.gradeEasy },
];

const CATEGORY_LABELS: Record<string, string> = {
  concept: '核心概念',
  relationship: '概念关系',
  boundary: '边界判断',
  misconception: '误解修正',
  transfer: '迁移应用',
};

type Filter = 'all' | 'wrong' | 'unseen';

export function FlashcardDeck({ projectSlug }: { projectSlug: string }) {
  const { data, isLoading, isError } = useFlashcards(projectSlug);
  const gradeCard = useGradeFlashcard();
  const cards = data?.flashcards ?? [];
  const progress = data?.progress ?? [];
  const [index, setIndex] = useState(0);
  const [flipped, setFlipped] = useState(false);
  const [filter, setFilter] = useState<Filter>('all');

  const progressMap = useMemo(() => {
    const map = new Map<string, CardProgress>();
    for (const item of progress) map.set(item.cardId, item);
    return map;
  }, [progress]);

  const filteredCards = useMemo(() => {
    if (filter === 'unseen') return cards.filter((card) => !progressMap.has(card.id));
    if (filter === 'wrong') {
      return cards.filter((card) => {
        const item = progressMap.get(card.id);
        return item?.lastGrade === 'forgot' || item?.lastGrade === 'fuzzy';
      });
    }
    return cards;
  }, [cards, filter, progressMap]);

  useEffect(() => {
    if (index >= filteredCards.length) setIndex(Math.max(0, filteredCards.length - 1));
  }, [filteredCards.length, index]);

  const current = filteredCards[index];
  const seenCount = cards.filter((card) => progressMap.has(card.id)).length;
  const strongCount = cards.filter((card) => {
    const grade = progressMap.get(card.id)?.lastGrade;
    return grade === 'got-it' || grade === 'easy';
  }).length;

  const move = useCallback((delta: number) => {
    setIndex((value) => Math.min(Math.max(value + delta, 0), Math.max(filteredCards.length - 1, 0)));
    setFlipped(false);
  }, [filteredCards.length]);

  const handleGrade = useCallback((grade: Grade) => {
    if (!current || gradeCard.isPending) return;
    gradeCard.mutate(
      { projectSlug, cardId: current.id, grade },
      { onSuccess: () => move(1) },
    );
  }, [current, gradeCard, move, projectSlug]);

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      const target = event.target;
      if (
        target instanceof Element &&
        target.matches('input, textarea, select, button, a, [contenteditable="true"]')
      ) {
        return;
      }
      if (event.key === 'ArrowLeft') move(-1);
      if (event.key === 'ArrowRight') move(1);
      if (event.key === ' ' || event.key === 'Enter') {
        event.preventDefault();
        setFlipped((value) => !value);
      }
    };
    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  }, [move]);

  if (isLoading) return <div className={s.loading}>正在整理复习卡片…</div>;
  if (isError) {
    return <EmptyState title="闪卡文件无法读取" description="请重新调用总结智能体生成合法的 flashcards.json。" />;
  }
  if (cards.length === 0) {
    return (
      <EmptyState
        title="还没有概念闪卡"
        description="调用总结智能体后，会在这里生成围绕核心概念、关系和边界的复习卡片。"
      />
    );
  }

  return (
    <section className={s.root}>
      <header className={s.deckHead}>
        <div>
          <span className={s.eyebrow}>概念复习</span>
          <h3>把理解重新想起来</h3>
          <p>先在心里作答，再翻面核对推理。空格翻面，方向键切换。</p>
        </div>
        <div className={s.stats} aria-label="复习统计">
          <Stat value={cards.length} label="卡片" />
          <Stat value={seenCount} label="已看" />
          <Stat value={strongCount} label="掌握" />
        </div>
      </header>

      <div className={s.toolbar}>
        <div className={s.filters} aria-label="闪卡筛选">
          {([
            ['all', '全部'],
            ['wrong', '待巩固'],
            ['unseen', '未看'],
          ] as const).map(([value, label]) => (
            <button
              key={value}
              className={clsx(s.filterBtn, filter === value && s.filterActive)}
              onClick={() => {
                setFilter(value);
                setIndex(0);
                setFlipped(false);
              }}
              type="button"
            >
              {label}
              <span>{filterCount(value, cards, progressMap)}</span>
            </button>
          ))}
        </div>
        <div className={s.progressMeta}>
          <span>{filteredCards.length > 0 ? `${index + 1} / ${filteredCards.length}` : '0 / 0'}</span>
          <div className={s.progressTrack}>
            <i style={{ width: `${filteredCards.length ? ((index + 1) / filteredCards.length) * 100 : 0}%` }} />
          </div>
        </div>
      </div>

      {current ? (
        <>
          <FlashcardView card={current} flipped={flipped} onFlip={() => setFlipped((value) => !value)} />

          <div className={s.controls}>
            <button type="button" className={s.navBtn} onClick={() => move(-1)} disabled={index === 0}>
              <span aria-hidden="true">←</span> 上一张
            </button>

            {flipped ? (
              <div className={s.gradeRow}>
                {GRADES.map((grade) => (
                  <button
                    key={grade.value}
                    type="button"
                    className={clsx(s.gradeBtn, grade.tone)}
                    onClick={() => handleGrade(grade.value)}
                    disabled={gradeCard.isPending}
                  >
                    <strong>{grade.label}</strong>
                    <span>{grade.hint}</span>
                  </button>
                ))}
              </div>
            ) : (
              <button type="button" className={s.revealBtn} onClick={() => setFlipped(true)}>
                显示答案
                <span>Space</span>
              </button>
            )}

            <button
              type="button"
              className={s.navBtn}
              onClick={() => move(1)}
              disabled={index >= filteredCards.length - 1}
            >
              下一张 <span aria-hidden="true">→</span>
            </button>
          </div>
        </>
      ) : (
        <div className={s.filteredEmpty}>
          <strong>这一组已经清空</strong>
          <span>换一个筛选继续复习。</span>
        </div>
      )}
    </section>
  );
}

function FlashcardView({
  card,
  flipped,
  onFlip,
}: {
  card: Flashcard;
  flipped: boolean;
  onFlip: () => void;
}) {
  const front = useMarkdown(card.front);
  const back = useMarkdown(card.back);
  const category = CATEGORY_LABELS[card.category ?? ''] ?? '理解卡';

  return (
    <div className={s.cardStage}>
      <div className={s.stackSheetOne} />
      <div className={s.stackSheetTwo} />
      <button
        type="button"
        className={clsx(s.card, flipped && s.cardFlipped)}
        onClick={onFlip}
        aria-label={flipped ? '查看问题' : '查看答案'}
      >
        <span className={s.cardIndex}>{card.id}</span>
        <span className={s.category}>{category}</span>
        <div className={s.cardContent}>
          <span className={s.faceLabel}>{flipped ? '理解核对' : '先想一想'}</span>
          <div
            className={clsx(s.cardText, flipped && s.answerText)}
            dangerouslySetInnerHTML={{ __html: flipped ? back.html : front.html }}
          />
        </div>
        <footer className={s.cardFoot}>
          <span>{flipped ? '再次点击返回问题' : '点击卡片翻面'}</span>
          {card.sourceRefs?.[0] && <code>{shortSource(card.sourceRefs[0])}</code>}
        </footer>
      </button>
    </div>
  );
}

function Stat({ value, label }: { value: number; label: string }) {
  return (
    <div className={s.stat}>
      <strong>{value}</strong>
      <span>{label}</span>
    </div>
  );
}

function filterCount(filter: Filter, cards: Flashcard[], progress: Map<string, CardProgress>) {
  if (filter === 'unseen') return cards.filter((card) => !progress.has(card.id)).length;
  if (filter === 'wrong') {
    return cards.filter((card) => {
      const grade = progress.get(card.id)?.lastGrade;
      return grade === 'forgot' || grade === 'fuzzy';
    }).length;
  }
  return cards.length;
}

function shortSource(source: string) {
  const parts = source.split('/');
  return parts[parts.length - 1] ?? source;
}
