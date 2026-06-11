import { useEffect, useState } from 'react';

import { useMemoryFile, useSaveMemory } from '@/api/memory';
import { Button } from '@/components/primitive/Button';
import { Card } from '@/components/primitive/Card';
import { Tag } from '@/components/primitive/Tag';

import s from './MemoryEditor.module.css';

interface MemoryEditorProps {
  slug: string;
  title: string;
  filename?: string;
}

export function MemoryEditor({ slug, title, filename = 'project-memory.md' }: MemoryEditorProps) {
  const { data, isLoading } = useMemoryFile(slug, filename);
  const save = useSaveMemory();

  const [draft, setDraft] = useState('');
  const [dirty, setDirty] = useState(false);

  // Sync loaded content into local draft (once per load).
  useEffect(() => {
    if (typeof data === 'string') {
      setDraft(data);
      setDirty(false);
    }
  }, [data]);

  const handleSave = async () => {
    await save.mutateAsync({ slug, filename, body: draft });
    setDirty(false);
  };

  return (
    <Card variant="outlined" className={s.card} id={`mem-${slug}`}>
      <header className={s.head}>
        <h4 className={s.title}>
          {title}
          <Tag tone="muted" className={s.tag}>
            memory/{filename}
          </Tag>
        </h4>
        <Button
          size="sm"
          variant="outline"
          onClick={handleSave}
          loading={save.isPending}
          disabled={!dirty}
        >
          {dirty ? '保存修改' : '已保存'}
        </Button>
      </header>
      <p className={s.hint}>Learner-owned：编辑后点击"保存修改"覆盖磁盘文件。</p>
      <textarea
        className={s.body}
        value={isLoading ? '加载中…' : draft}
        onChange={(e) => {
          setDraft(e.target.value);
          setDirty(true);
        }}
        rows={10}
        spellCheck={false}
      />
      {save.error && (
        <div className={s.error}>保存失败：{(save.error as Error).message}</div>
      )}
    </Card>
  );
}
