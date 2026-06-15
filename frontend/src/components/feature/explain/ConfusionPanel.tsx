import { useMemo, useState } from 'react';
import { ChevronLeftIcon, ChevronRightIcon, CopyIcon, TrashIcon } from '@radix-ui/react-icons';
import clsx from 'clsx';

import { useConfusions, useDeleteConfusion, type Confusion } from '@/api/confusions';
import { Button } from '@/components/primitive/Button';
import { Modal } from '@/components/primitive/Modal';

import s from './ConfusionPanel.module.css';

interface ConfusionPanelProps {
  projectSlug: string;
  collapsed?: boolean;
  onCollapsedChange?: (collapsed: boolean) => void;
  className?: string;
}

export function ConfusionPanel({
  projectSlug,
  collapsed = false,
  onCollapsedChange,
  className,
}: ConfusionPanelProps) {
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [deleteTarget, setDeleteTarget] = useState<Confusion | null>(null);
  const [copied, setCopied] = useState(false);
  const { data: confusions = [], isLoading } = useConfusions(projectSlug);
  const deleteConfusion = useDeleteConfusion();
  const visible = useMemo(
    () => confusions.filter((item) => item.state !== 'deleted'),
    [confusions],
  );

  const toggleSelect = (id: string) => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  const copySelected = async () => {
    const text = visible
      .filter((item) => selected.has(item.id))
      .map((item, index) => `${index + 1}. ${item.quoteSnapshot}`)
      .join('\n\n');
    if (!text) return;
    await copyText(text);
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1600);
  };

  if (collapsed) {
    return (
      <div className={clsx(s.collapsed, className)}>
        <button
          type="button"
          className={s.collapseButton}
          onClick={() => onCollapsedChange?.(false)}
          aria-label="展开摘要"
          title="展开摘要"
        >
          <ChevronLeftIcon />
        </button>
        <span className={s.collapsedLabel}>摘要</span>
        {visible.length > 0 && <span className={s.count}>{visible.length}</span>}
      </div>
    );
  }

  return (
    <div className={clsx(s.root, className)}>
      <header className={s.head}>
        <div>
          <h4 className={s.title}>摘要</h4>
          <p className={s.subtitle}>选中教程文字即可保存</p>
        </div>
        <button
          type="button"
          className={s.collapseButton}
          onClick={() => onCollapsedChange?.(true)}
          aria-label="收起摘要"
          title="收起摘要"
        >
          <ChevronRightIcon />
        </button>
      </header>

      <div className={s.list}>
        {isLoading && <div className={s.empty}>加载中…</div>}
        {!isLoading && visible.length === 0 && (
          <div className={s.empty}>还没有摘要。用鼠标选中讲解正文，点击“保存摘要”。</div>
        )}
        {visible.map((item) => (
          <SummaryItem
            key={item.id}
            summary={item}
            selected={selected.has(item.id)}
            onToggle={() => toggleSelect(item.id)}
            onDelete={() => setDeleteTarget(item)}
          />
        ))}
      </div>

      {visible.length > 0 && (
        <footer className={s.foot}>
          <Button
            variant="primary"
            size="sm"
            disabled={selected.size === 0}
            iconLeft={<CopyIcon />}
            onClick={copySelected}
          >
            {copied ? '已复制' : `复制 (${selected.size})`}
          </Button>
        </footer>
      )}

      <Modal
        open={deleteTarget !== null}
        onOpenChange={(open) => {
          if (!open) setDeleteTarget(null);
        }}
        title="删除这条摘要？"
        description="删除后，教程正文中的黄色高亮也会消失。"
        size="sm"
        footer={(
          <>
            <Button variant="ghost" onClick={() => setDeleteTarget(null)}>取消</Button>
            <Button
              variant="danger"
              loading={deleteConfusion.isPending}
              onClick={async () => {
                if (!deleteTarget) return;
                await deleteConfusion.mutateAsync({
                  projectSlug,
                  confusionId: deleteTarget.id,
                });
                setSelected((prev) => {
                  const next = new Set(prev);
                  next.delete(deleteTarget.id);
                  return next;
                });
                setDeleteTarget(null);
              }}
            >
              删除
            </Button>
          </>
        )}
      >
        <blockquote className={s.deletePreview}>{deleteTarget?.quoteSnapshot}</blockquote>
      </Modal>
    </div>
  );
}

function SummaryItem({
  summary,
  selected,
  onToggle,
  onDelete,
}: {
  summary: Confusion;
  selected: boolean;
  onToggle: () => void;
  onDelete: () => void;
}) {
  return (
    <div className={clsx(s.item, selected && s.itemSelected)}>
      <label className={s.itemCheck}>
        <input type="checkbox" checked={selected} onChange={onToggle} />
      </label>
      <blockquote className={s.quote}>{summary.quoteSnapshot}</blockquote>
      <button
        className={s.deleteButton}
        onClick={onDelete}
        title="删除摘要"
        aria-label="删除摘要"
        type="button"
      >
        <TrashIcon />
      </button>
    </div>
  );
}

async function copyText(text: string) {
  if (navigator.clipboard?.writeText) {
    await navigator.clipboard.writeText(text);
    return;
  }
  const textarea = document.createElement('textarea');
  textarea.value = text;
  textarea.style.position = 'fixed';
  textarea.style.opacity = '0';
  document.body.appendChild(textarea);
  textarea.select();
  document.execCommand('copy');
  textarea.remove();
}
