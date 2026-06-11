import { CSSProperties } from 'react';
import clsx from 'clsx';

import s from './Icon.module.css';

// Icon names available in /public/img/icons.svg (the v2 SVG sprite).
// Keep this list in sync with the sprite; unknown names silently render
// nothing rather than throwing, so a missing icon doesn't break the page.
export type IconName =
  | 'terminal'
  | 'check'
  | 'circle'
  | 'plus'
  | 'folder'
  | 'clock'
  | 'brain'
  | 'bot'
  | 'zap'
  | 'x';

interface IconProps {
  name: IconName;
  size?: number;
  className?: string;
  style?: CSSProperties;
  title?: string;
}

export function Icon({ name, size = 18, className, style, title }: IconProps) {
  return (
    <svg
      className={clsx(s.ico, className)}
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={1.6}
      style={style}
      role={title ? 'img' : 'presentation'}
      aria-label={title}
    >
      <use href={`/img/icons.svg#${name}`} />
    </svg>
  );
}
