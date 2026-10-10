import { useEffect, useState } from 'react';

import type { SourceCitation } from '@/features/learning/api/learningWorkspace';
import { API_BASE } from '@/shared/lib/constants';
import { MarkdownView } from '@/shared/primitive/MarkdownView';

import s from './SourceCitations.module.css';

interface EvidencePreview extends SourceCitation {
  quote: string;
  previous?: string;
  next?: string;
}

export function SourceCitations({ slug, citations, sourceNames, selected, onSelect }: {
  slug: string;
  citations: SourceCitation[];
  sourceNames: Map<string, string>;
  selected: string;
  onSelect: (id: string) => void;
}) {
  const citation = citations.find(item => item.evidenceId === selected);
  const [preview, setPreview] = useState<EvidencePreview | null>(null);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const [direction, setDirection] = useState<{ selectedId: string; origin: string; value: 'previous' | 'next' } | null>(null);

  useEffect(() => {
    setPreview(null);
    setError('');
    if (!citation) { setLoading(false); return; }
    const controller = new AbortController();
    const evidenceId = direction?.selectedId === selected ? direction.origin : citation.evidenceId;
    const query = direction?.selectedId === selected ? `?direction=${direction.value}` : '';
    setLoading(true);
    void fetch(`${API_BASE}/api/projects/${encodeURIComponent(slug)}/sources/${encodeURIComponent(citation.sourceId)}/evidence/${encodeURIComponent(evidenceId)}${query}`, { signal: controller.signal })
      .then(async response => {
        if (!response.ok) {
          if (response.status === 410) throw new Error('这份资料已删除。');
          if (response.status === 409) throw new Error('资料修订已变化，请重新查阅。');
          throw new Error('暂时无法读取这段原文。');
        }
        return await response.json() as EvidencePreview;
      })
      .then(value => { if (!controller.signal.aborted) setPreview(value); })
      .catch(cause => { if (!controller.signal.aborted) setError((cause as Error).message); })
      .finally(() => { if (!controller.signal.aborted) setLoading(false); });
    return () => controller.abort();
  }, [citation, direction, selected, slug]);

  if (!citations.length) return null;
  return (
    <section className={s.sources} aria-label="本次查阅来源">
      <div className={s.links}>
        {citations.map((item, index) => <button key={item.evidenceId} type="button" aria-expanded={selected === item.evidenceId} onClick={() => { setDirection(null); onSelect(selected === item.evidenceId ? '' : item.evidenceId); }}>
          {index + 1} · {sourceNames.get(item.sourceId) ?? '资料'} · {item.displayRange}
        </button>)}
      </div>
      {citation && <div className={s.preview} aria-label="引用原文">
        <header><strong>{sourceNames.get(citation.sourceId) ?? '资料'} · {preview?.displayRange ?? citation.displayRange}</strong><button type="button" aria-label="关闭原文" onClick={() => onSelect('')}>×</button></header>
        {loading && <p role="status">正在读取原文…</p>}
        {error && <p role="alert">{error}</p>}
        {preview && <>
          <MarkdownView source={preview.quote} />
          {(preview.previous || preview.next) && <footer>
            {preview.previous && <button type="button" onClick={() => setDirection({ selectedId: selected, origin: preview.evidenceId, value: 'previous' })}>前一段</button>}
            {preview.next && <button type="button" onClick={() => setDirection({ selectedId: selected, origin: preview.evidenceId, value: 'next' })}>后一段</button>}
          </footer>}
        </>}
      </div>}
    </section>
  );
}
