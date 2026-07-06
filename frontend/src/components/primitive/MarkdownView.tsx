import { useEffect, useRef } from 'react';

import { useMarkdown } from '@/hooks/useMarkdown';

export function MarkdownView({ source, className }: { source: string; className?: string }) {
  const { html, mermaid } = useMarkdown(source);
  const hostRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!hostRef.current || mermaid.length === 0) return;
    let cancelled = false;
    void import('mermaid').then(async (mod) => {
      if (cancelled) return;
      for (const block of mermaid) {
        const target = hostRef.current?.querySelector(`[data-mermaid-id="${block.id}"]`);
        if (!target) continue;
        try {
          const result = await mod.default.render(`${block.id}-mdview-svg`, block.code);
          (target as HTMLElement).innerHTML = result.svg;
        } catch (err) {
          (target as HTMLElement).textContent = `Mermaid render error: ${(err as Error).message}`;
        }
      }
    });
    return () => {
      cancelled = true;
    };
  }, [html, mermaid]);

  return (
    <div ref={hostRef} className={className} dangerouslySetInnerHTML={{ __html: html }} />
  );
}
