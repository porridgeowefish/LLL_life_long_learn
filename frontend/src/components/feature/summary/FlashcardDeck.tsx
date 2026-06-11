// FlashcardDeck — flip cards with grade buttons and navigation.

import { useState, useMemo, useCallback } from 'react';
import clsx from 'clsx';

import { useFlashcards, useGradeFlashcard, type CardProgress, type Grade } from '@/api/flashcards';
import { Button } from '@/components/primitive/Button';
import { EmptyState } from '@/components/primitive/EmptyState';

import s from './FlashcardDeck.module.css';

const GRADES: { value: Grade; label: string; color: string }[] = [
  { value: 'forgot', label: '忘记', color: '#c44' },
  { value: 'fuzzy', label: '模糊', color: '#c90' },
  { value: 'got-it', label: '掌握', color: 'var(--accent)' },
  { value: 'easy', label: '太简单', color: '#6a9' },
];

interface FlashcardDeckProps {
  projectSlug: string;
}

export function FlashcardDeck({ projectSlug }: FlashcardDeckProps) {
  const { data, isLoading } = useFlashcards(projectSlug);
  const gradeCard = useGradeFlashcard();

  const cards = data?.flashcards ?? [];
  const progress = data?.progress ?? [];
  const [index, setIndex] = useState(0);
  const [flipped, setFlipped] = useState(false);
  const [filter, setFilter] = useState<'all' | 'wrong' | 'unseen'>('all');

  const progressMap = useMemo(() => {
    const m = new Map<string, CardProgress>();
    for (const p of progress) m.set(p.cardId, p);
    return m;
  }, [progress]);

  const filteredCards = useMemo(() => {
    if (filter === 'all') return cards;
    if (filter === 'unseen') return cards.filter((c) => !progressMap.has(c.id));
    if (filter === 'wrong') return cards.filter((c) => {
      const p = progressMap.get(c.id);
      return p && (p.lastGrade === 'forgot' || p.lastGrade === 'fuzzy');
    });
    return cards;
  }, [cards, filter, progressMap]);

  const current = filteredCards[index];

  const handleGrade = useCallback((grade: Grade) => {
    if (!current) return;
    gradeCard.mutate({ projectSlug, cardId: current.id, grade });
    setFlipped(false);
    if (index < filteredCards.length - 1) {
      setIndex(index + 1);
    }
  }, [current, gradeCard, projectSlug, index, filteredCards.length]);

  const handlePrev = () => {
    if (index > 0) setIndex(index - 1);
    setFlipped(false);
  };

  const handleNext = () => {
    if (index < filteredCards.length - 1) setIndex(index + 1);
    setFlipped(false);
  };

  if (isLoading) return <div className={s.loading}>加载中…</div>;

  if (cards.length === 0) {
    return (
      <EmptyState
        title="闪卡尚未生成"
        description={<>调用 Summary Agent 生成闪卡后，在这里复习。</>}
      />
    );
  }

  return (
    <div className={s.root}>
      <header className={s.head}>
        <div className={s.filters}>
          {(['all', 'wrong', 'unseen'] as const).map((f) => (
            <button
              key={f}
              className={clsx(s.filterBtn, filter === f && s.filterActive)}
              onClick={() => { setFilter(f); setIndex(0); setFlipped(false); }}
              type="button"
            >
              {f === 'all' ? '全部' : f === 'wrong' ? '答错' : '未看'}
            </button>
          ))}
        </div>
        <span className={s.counter}>
          {filteredCards.length > 0 ? `${index + 1} / ${filteredCards.length}` : '0 / 0'}
        </span>
      </header>

      {filteredCards.length === 0 && (
        <div className={s.empty}>当前筛选无卡片。</div>
      )}

      {current && (
        <div className={s.cardArea}>
          <div
            className={clsx(s.card, flipped && s.cardFlipped)}
            onClick={() => setFlipped(!flipped)}
            role="button"
            tabIndex={0}
          >
            <div className={s.cardFront}>
              <div className={s.cardLabel}>问题</div>
              <div className={s.cardText}>{current.front}</div>
              <div className={s.cardHint}>点击翻转</div>
            </div>
            <div className={s.cardBack}>
              <div className={s.cardLabel}>答案</div>
              <div className={s.cardText}>{current.back}</div>
            </div>
          </div>
        </div>
      )}

      {current && flipped && (
        <div className={s.gradeRow}>
          {GRADES.map((g) => (
            <button
              key={g.value}
              className={s.gradeBtn}
              style={{ borderColor: g.color, color: g.color }}
              onClick={() => handleGrade(g.value)}
              type="button"
            >
              {g.label}
            </button>
          ))}
        </div>
      )}

      <div className={s.nav}>
        <Button variant="outline" size="sm" onClick={handlePrev} disabled={index === 0}>
          ← 上一张
        </Button>
        <Button variant="outline" size="sm" onClick={handleNext} disabled={index >= filteredCards.length - 1}>
          下一张 →
        </Button>
      </div>
    </div>
  );
}
