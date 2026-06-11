import { InputHTMLAttributes } from 'react';
import clsx from 'clsx';

import { Icon } from './Icon';
import s from './SearchInput.module.css';

interface SearchInputProps extends Omit<InputHTMLAttributes<HTMLInputElement>, 'type'> {
  onClear?: () => void;
}

export function SearchInput({ className, onClear, value, ...rest }: SearchInputProps) {
  return (
    <div className={clsx(s.wrap, className)}>
      <Icon name="zap" size={14} className={s.lead} />
      <input
        type="search"
        className={s.input}
        value={value}
        {...rest}
      />
      {value && onClear && (
        <button
          type="button"
          className={s.clear}
          onClick={onClear}
          aria-label="清除搜索"
        >
          <Icon name="x" size={14} />
        </button>
      )}
    </div>
  );
}
