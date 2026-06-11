import { ButtonHTMLAttributes, forwardRef, ReactNode } from 'react';
import clsx from 'clsx';

import s from './Button.module.css';

type Variant = 'primary' | 'outline' | 'ghost' | 'danger';
type Size = 'sm' | 'md';

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: Variant;
  size?: Size;
  iconLeft?: ReactNode;
  iconRight?: ReactNode;
  loading?: boolean;
}

export const Button = forwardRef<HTMLButtonElement, ButtonProps>(function Button(
  {
    variant = 'outline',
    size = 'md',
    iconLeft,
    iconRight,
    loading = false,
    className,
    children,
    disabled,
    ...rest
  },
  ref,
) {
  return (
    <button
      ref={ref}
      className={clsx(s.btn, s[`v-${variant}`], s[`s-${size}`], loading && s.loading, className)}
      disabled={disabled || loading}
      {...rest}
    >
      {iconLeft && <span className={s.iconLeft}>{iconLeft}</span>}
      <span className={s.label}>{children}</span>
      {iconRight && <span className={s.iconRight}>{iconRight}</span>}
    </button>
  );
});
