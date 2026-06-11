// ConfusionPanel — lists / filters confusion markers for the Explain zone.
// Allows "统一提问" batch action (marks selected confusions as "asked").

import { useState } from 'react';
import clsx from 'clsx';

import { useConfusions, useDeleteConfusion, useUpdateConfusion, type Confusion, type ConfusionState } from '@/api/confusions';
import { Button } from '@/components/primitive/Button';

import s from './ConfusionPanel.module.css';

const STATE_LABELS: Record<ConfusionState, string> = {
  open: '待提问',
  asked: '已提问',
  resolved: '已解决',
  deleted: '已删除',
};

type Filter = 'all' | ConfusionState;

interface ConfusionPanelProps {
  projectSlug: string;
  /** Called with selected confusion IDs when learner clicks "统一提问". */
  onBatchAsk?: (ids: string[]) => void;
  className?: string;
}

export function ConfusionPanel({ projectSlug, onBatchAsk, className }: ConfusionPanelProps) {
  const [filter, setFilter] = useState<Filter>('all');
  const [selected, setSelected] = useState<Set<string>>(new Set());

  const stateFilter = filter === 'all' ? undefined : filter;
  const { data: confusions, isLoading } = useConfusions(projectSlug, stateFilter);
  const deleteConfusion = useDeleteConfusion();
  const updateConfusion = useUpdateConfusion();

  const toggleSelect = (id: string) => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  const handleBatchAsk = () => {
    if (selected.size === 0) return;
    // Mark each selected confusion as "asked".
    for (const id of selected) {
      updateConfusion.mutate({ projectSlug, confusionId: id, patch: { state: 'asked' } });
    }
    onBatchAsk?.(Array.from(selected));
    setSelected(new Set());
  };

  const openCount = confusions?.filter((c) => c.state === 'open').length ?? 0;

  return (
    <div className={clsx(s.root, className)}>
      <header className={s.head}>
        <h4 className={s.title}>困惑标记</h4>
        <div className={s.filters}>
          {(['all', 'open', 'asked', 'resolved'] as const).map((f) => (
            <button
              key={f}
              className={clsx(s.filterBtn, filter === f && s.filterActive)}
              onClick={() => setFilter(f)}
            >
              {f === 'all' ? '全部' : STATE_LABELS[f]}
            </button>
          ))}
        </div>
      </header>

      {isLoading && <div className={s.empty}>加载中…</div>}

      {!isLoading && (!confusions || confusions.length === 0) && (
        <div className={s.empty}>暂无困惑标记。在讲解内容中选择文字 → "标困惑" 添加。</div>
      )}

      {confusions?.map((c) => (
        <ConfusionItem
          key={c.id}
          confusion={c}
          selected={selected.has(c.id)}
          onToggle={() => toggleSelect(c.id)}
          onDelete={() => deleteConfusion.mutate({ projectSlug, confusionId: c.id })}
        />
      ))}

      {openCount > 0 && (
        <footer className={s.foot}>
          <Button
            variant="primary"
            size="sm"
            disabled={selected.size === 0}
            onClick={handleBatchAsk}
          >
            统一提问 ({selected.size})
          </Button>
          <span className={s.footHint}>选择困惑后批量提问给 Explain Agent</span>
        </footer>
      )}
    </div>
  );
}

function ConfusionItem({
  confusion,
  selected,
  onToggle,
  onDelete,
}: {
  confusion: Confusion;
  selected: boolean;
  onToggle: () => void;
  onDelete: () => void;
}) {
  return (
    <div className={clsx(s.item, selected && s.itemSelected)}>
      <label className={s.itemCheck}>
        <input
          type="checkbox"
          checked={selected}
          onChange={onToggle}
          disabled={confusion.state !== 'open'}
        />
      </label>
      <div className={s.itemBody}>
        <blockquote className={s.quote}>{confusion.quoteSnapshot}</blockquote>
        {confusion.notes && <p className={s.notes}>{confusion.notes}</p>}
        <div className={s.meta}>
          <span className={s.stateTag}>{STATE_LABELS[confusion.state]}</span>
        </div>
      </div>
      <button
        className={s.deleteBtn}
        onClick={onDelete}
        title="删除"
        type="button"
      >
        ×
      </button>
    </div>
  );
}
