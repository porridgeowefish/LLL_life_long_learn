// AnswerField — type-differentiated answer input for one practice question.
// short-answer → single-line input; essay → multi-line textarea;
// code → monospace textarea. readonly renders as a static block.

import clsx from 'clsx';

import type { PracticeTask } from '@/api/practice';

import s from './AnswerField.module.css';

interface AnswerFieldProps {
  type: PracticeTask['type'];
  value: string;
  onChange: (v: string) => void;
  readonly?: boolean;
}

export function AnswerField({ type, value, onChange, readonly = false }: AnswerFieldProps) {
  if (readonly) {
    return (
      <div className={clsx(s.readonly, type === 'code' && s.code)}>
        {value ? value : <span className={s.placeholder}>（未作答）</span>}
      </div>
    );
  }

  if (type === 'short-answer') {
    return (
      <input
        type="text"
        className={s.shortInput}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder="一句话作答…"
      />
    );
  }

  if (type === 'code') {
    return (
      <textarea
        className={clsx(s.textarea, s.code)}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        rows={12}
        spellCheck={false}
        placeholder="// 在这里写代码…"
      />
    );
  }

  // essay
  return (
    <textarea
      className={s.textarea}
      value={value}
      onChange={(e) => onChange(e.target.value)}
      rows={8}
      placeholder="展开你的论述…"
    />
  );
}
