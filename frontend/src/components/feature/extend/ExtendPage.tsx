// ExtendPage — OutputViewer + MarkdownEditor for extend/relation-notes.md.
// The Extend agent's output is read-only (output.md); the learner can write
// their own relation notes in a separate editor.

import { OutputViewer } from '@/components/feature/project/OutputViewer';
import { MarkdownEditor } from '@/components/primitive/MarkdownEditor';

import s from './ExtendPage.module.css';

interface ExtendPageProps {
  projectSlug: string;
}

export function ExtendPage({ projectSlug }: ExtendPageProps) {
  return (
    <div className={s.root}>
      <section className={s.section}>
        <h3 className={s.heading}>拓展内容</h3>
        <OutputViewer slug={projectSlug} zone="Extend" />
      </section>

      <section className={s.section}>
        <h3 className={s.heading}>关联笔记</h3>
        <MarkdownEditor
          slug={projectSlug}
          relPath="extend/relation-notes.md"
          label="关联笔记"
          defaultMode="edit"
        />
      </section>
    </div>
  );
}
