import { HTMLAttributes, ReactNode } from 'react';
import clsx from 'clsx';

import s from './Tag.module.css';

type Tone = 'neutral' | 'accent' | 'orange' | 'sky' | 'pink' | 'muted';

interface TagProps extends HTMLAttributes<HTMLSpanElement> {
  tone?: Tone;
  children: ReactNode;
}

export function Tag({ tone = 'neutral', className, children, ...rest }: TagProps) {
  return (
    <span className={clsx(s.tag, s[`t-${tone}`], className)} {...rest}>
      {children}
    </span>
  );
}
