// SummaryPage — tab container for Summary zone.
// Two tabs: "闪卡复习" and "学习报告". Report tab uses MarkdownEditor for summary.md.

import { useState } from 'react';
import clsx from 'clsx';

import { FlashcardDeck } from './FlashcardDeck';
import { MarkdownEditor } from '@/components/primitive/MarkdownEditor';

import s from './SummaryPage.module.css';

type Tab = 'flashcards' | 'report';

interface SummaryPageProps {
  projectSlug: string;
}

export function SummaryPage({ projectSlug }: SummaryPageProps) {
  const [tab, setTab] = useState<Tab>('flashcards');

  return (
    <div className={s.root}>
      <div className={s.tabs}>
        <button
          className={clsx(s.tab, tab === 'flashcards' && s.tabActive)}
          onClick={() => setTab('flashcards')}
          type="button"
        >
          闪卡复习
        </button>
        <button
          className={clsx(s.tab, tab === 'report' && s.tabActive)}
          onClick={() => setTab('report')}
          type="button"
        >
          学习报告
        </button>
      </div>

      <div className={s.content}>
        {tab === 'flashcards' && (
          <FlashcardDeck projectSlug={projectSlug} />
        )}
        {tab === 'report' && (
          <MarkdownEditor
            slug={projectSlug}
            relPath="summary/summary.md"
            label="学习总结"
            defaultMode="split"
          />
        )}
      </div>
    </div>
  );
}
