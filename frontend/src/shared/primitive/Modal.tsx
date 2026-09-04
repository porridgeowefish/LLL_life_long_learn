import { ReactNode } from 'react';
import * as Dialog from '@radix-ui/react-dialog';
import clsx from 'clsx';

import s from './Modal.module.css';

interface ModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title?: ReactNode;
  description?: ReactNode;
  children: ReactNode;
  footer?: ReactNode;
  size?: 'sm' | 'md' | 'lg';
  className?: string;
}

export function Modal({
  open,
  onOpenChange,
  title,
  description,
  children,
  footer,
  size = 'md',
  className,
}: ModalProps) {
  return (
    <Dialog.Root open={open} onOpenChange={onOpenChange}>
      <Dialog.Portal>
        <Dialog.Overlay className={s.overlay} />
        <Dialog.Content className={clsx(s.content, s[`sz-${size}`], className)}>
          {title && <Dialog.Title className={s.title}>{title}</Dialog.Title>}
          {description && (
            <Dialog.Description className={s.description}>{description}</Dialog.Description>
          )}
          <div className={s.body}>{children}</div>
          {footer && <div className={s.footer}>{footer}</div>}
          <Dialog.Close className={s.close} aria-label="关闭">
            ✕
          </Dialog.Close>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
