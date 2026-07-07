import { ExtendKnowledgeFlower } from './ExtendKnowledgeFlower';
import s from './ExtendPage.module.css';

interface ExtendPageProps {
  projectSlug: string;
  projectTitle: string;
}

export function ExtendPage({ projectSlug, projectTitle }: ExtendPageProps) {
  return (
    <div className={s.root}>
      <ExtendKnowledgeFlower projectSlug={projectSlug} projectTitle={projectTitle} />
    </div>
  );
}
