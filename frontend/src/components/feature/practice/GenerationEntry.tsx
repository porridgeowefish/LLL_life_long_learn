// GenerationEntry — the Practice zone entry card shown when no tasks exist.
// Lets the learner pick a question count (1-10, default 5) and trigger the
// Practice Agent. P-01: counts >10 are not silently truncated — we show a
// reason and require an explicit confirm.

import { useState } from 'react';
import clsx from 'clsx';

import { Button } from '@/components/primitive/Button';
import { Card } from '@/components/primitive/Card';

import s from './GenerationEntry.module.css';

interface GenerationEntryProps {
  onGenerate: (count: number) => void;
  isInvoking?: boolean;
}

export function GenerationEntry({ onGenerate, isInvoking = false }: GenerationEntryProps) {
  const [count, setCount] = useState(5);
  const [confirmOver, setConfirmOver] = useState(false);

  const overLimit = count > 10;

  const handleGenerate = () => {
    if (overLimit && !confirmOver) {
      setConfirmOver(true);
      return;
    }
    onGenerate(count);
  };

  return (
    <Card variant="outlined" className={s.root}>
      <h3 className={s.title}>生成练习题</h3>
      <p className={s.desc}>
        选择题量，生成从一星到五星的渐进题组。客观题答后立即展示答案解析，主观题整批评估。
      </p>

      <div className={s.countRow}>
        <label className={s.label}>题量</label>
        <button
          type="button"
          className={s.stepBtn}
          onClick={() => { setCount((c) => Math.max(1, c - 1)); setConfirmOver(false); }}
          aria-label="减少题量"
        >
          −
        </button>
        <input
          type="number"
          className={s.countInput}
          value={count}
          min={1}
          onChange={(e) => {
            const n = Number(e.target.value);
            setCount(Number.isFinite(n) && n >= 1 ? Math.floor(n) : 1);
            setConfirmOver(false);
          }}
        />
        <button
          type="button"
          className={s.stepBtn}
          onClick={() => { setCount((c) => c + 1); setConfirmOver(false); }}
          aria-label="增加题量"
        >
          +
        </button>
        <span className={s.hint}>默认 5 题，建议 1-10</span>
      </div>

      {overLimit && (
        <div className={clsx(s.warn, confirmOver && s.warnConfirmed)}>
          {confirmOver ? (
            <>
              已确认生成 <strong>{count}</strong> 题。题量过大可能降低作答耐心与反馈质量。
            </>
          ) : (
            <>
              建议题量 1-10。当前 <strong>{count}</strong> 题偏多——点"生成"再次确认，或调小题量。
            </>
          )}
        </div>
      )}

      <div className={s.actions}>
        <Button variant="primary" onClick={handleGenerate} loading={isInvoking}>
          {overLimit && !confirmOver ? '确认生成' : '生成练习题'}
        </Button>
      </div>
    </Card>
  );
}
