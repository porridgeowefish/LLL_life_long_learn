import { ReactNode } from 'react';
import clsx from 'clsx';

import s from './EmptyState.module.css';

interface EmptyStateProps {
  title: ReactNode;
  description?: ReactNode;
  action?: ReactNode;
  className?: string;
}

export function EmptyState({ title, description, action, className }: EmptyStateProps) {
  return (
    <div className={clsx(s.empty, className)}>
      <h3 className={s.title}>{title}</h3>
      {description && <p className={s.description}>{description}</p>}
      {action && <div className={s.action}>{action}</div>}
    </div>
  );
}
