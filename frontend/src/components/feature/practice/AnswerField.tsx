import clsx from 'clsx';

import type { PracticeAnswer, PracticeTask } from '@/api/practice';

import s from './AnswerField.module.css';

interface AnswerFieldProps {
  task: PracticeTask;
  value: PracticeAnswer;
  onChange: (value: PracticeAnswer) => void;
  readonly?: boolean;
}

export function AnswerField({ task, value, onChange, readonly = false }: AnswerFieldProps) {
  if (readonly) {
    const shown = Array.isArray(value) ? value.join('、') : typeof value === 'boolean' ? (value ? '正确' : '错误') : value;
    return (
      <div className={clsx(s.readonly, task.type === 'code' && s.code)}>
        {shown === '' ? <span className={s.placeholder}>（未作答）</span> : String(shown)}
      </div>
    );
  }

  if (task.type === 'true-false') {
    return (
      <div className={s.choiceList}>
        {[true, false].map((option) => (
          <label key={String(option)} className={s.choice}>
            <input
              type="radio"
              checked={value === option}
              onChange={() => onChange(option)}
            />
            <span>{option ? '正确' : '错误'}</span>
          </label>
        ))}
      </div>
    );
  }

  if (task.type === 'single-choice' || task.type === 'multiple-choice') {
    const selected = Array.isArray(value) ? value : [];
    return (
      <div className={s.choiceList}>
        {(task.options ?? []).map((option) => {
          const checked = task.type === 'single-choice'
            ? value === option.id
            : selected.includes(option.id);
          return (
            <label key={option.id} className={clsx(s.choice, checked && s.choiceActive)}>
              <input
                type={task.type === 'single-choice' ? 'radio' : 'checkbox'}
                checked={checked}
                onChange={() => {
                  if (task.type === 'single-choice') {
                    onChange(option.id);
                  } else {
                    onChange(
                      checked
                        ? selected.filter((id) => id !== option.id)
                        : [...selected, option.id],
                    );
                  }
                }}
              />
              <strong>{option.id}</strong>
              <span>{option.text}</span>
            </label>
          );
        })}
      </div>
    );
  }

  const textValue = typeof value === 'string' ? value : '';
  if (task.type === 'short-answer') {
    return (
      <input
        type="text"
        className={s.shortInput}
        value={textValue}
        onChange={(event) => onChange(event.target.value)}
        placeholder="一句话作答…"
      />
    );
  }

  return (
    <textarea
      className={clsx(s.textarea, task.type === 'code' && s.code)}
      value={textValue}
      onChange={(event) => onChange(event.target.value)}
      rows={task.type === 'code' ? 12 : 8}
      spellCheck={task.type !== 'code'}
      placeholder={task.type === 'code' ? '// 在这里写代码…' : '展开你的论述…'}
    />
  );
}
