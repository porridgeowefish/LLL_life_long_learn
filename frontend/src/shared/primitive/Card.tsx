import { HTMLAttributes, ReactNode } from 'react';
import clsx from 'clsx';

import s from './Card.module.css';

interface CardProps extends HTMLAttributes<HTMLDivElement> {
  variant?: 'flat' | 'outlined' | 'accent';
  children: ReactNode;
}

export function Card({ variant = 'outlined', className, children, ...rest }: CardProps) {
  return (
    <div className={clsx(s.card, s[`v-${variant}`], className)} {...rest}>
      {children}
    </div>
  );
}

interface CardSectionProps extends HTMLAttributes<HTMLDivElement> {
  children: ReactNode;
}

export function CardHeader({ className, children, ...rest }: CardSectionProps) {
  return (
    <div className={clsx(s.head, className)} {...rest}>
      {children}
    </div>
  );
}

export function CardBody({ className, children, ...rest }: CardSectionProps) {
  return (
    <div className={clsx(s.body, className)} {...rest}>
      {children}
    </div>
  );
}
