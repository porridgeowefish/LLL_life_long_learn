import { forwardRef, useEffect, useImperativeHandle, useRef } from 'react';

import { useMarkdown } from '@/hooks/useMarkdown';
import { mountMermaidBlocks } from '@/lib/mermaidRenderer';

export const MarkdownView = forwardRef<HTMLDivElement, { source: string; className?: string }>(function MarkdownView({ source, className }, forwardedRef) {
  const { html, mermaid } = useMarkdown(source);
  const hostRef = useRef<HTMLDivElement>(null);
  useImperativeHandle(forwardedRef, () => hostRef.current as HTMLDivElement, []);

  useEffect(() => {
    if (!hostRef.current || mermaid.length === 0) return;
    return mountMermaidBlocks(hostRef.current, mermaid, 'markdown-view');
  }, [html, mermaid]);

  return (
    <div ref={hostRef} className={className} dangerouslySetInnerHTML={{ __html: html }} />
  );
});
