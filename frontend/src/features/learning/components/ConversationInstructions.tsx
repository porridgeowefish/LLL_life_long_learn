import { useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { http } from '@/shared/client';
import { Modal } from '@/shared/primitive/Modal';
import { Button } from '@/shared/primitive/Button';
import s from './ConversationInstructions.module.css';

export function ConversationInstructions({ slug }: { slug: string }) {
  const [open, setOpen] = useState(false);
  const [draft, setDraft] = useState<string | null>(null);
  const queryClient = useQueryClient();
  const key = ['learning-workspace', slug, 'instructions'];
  const endpoint = `/api/projects/${encodeURIComponent(slug)}/conversation/instructions`;
  const query = useQuery({ queryKey: key, queryFn: () => http.get<{ systemPrompt: string }>(endpoint), enabled: open });
  const save = useMutation({
    mutationFn: (systemPrompt: string) => http.put<{ systemPrompt: string }>(endpoint, { systemPrompt }),
    onSuccess: (value) => { queryClient.setQueryData(key, value); setOpen(false); setDraft(null); },
  });
  const changeOpen = (value: boolean) => { if (save.isPending) return; setOpen(value); setDraft(null); save.reset(); };
  return <>
    <button type="button" className={s.trigger} onClick={() => changeOpen(true)}>附加信息</button>
    <Modal open={open} onOpenChange={changeOpen} title="当前对话的附加信息" description="有无特殊要求？可以不填。保存后持续用于本对话，刷新或重启后仍保留。" footer={<>
      <Button variant="outline" disabled={save.isPending} onClick={() => changeOpen(false)}>取消</Button>
      <Button variant="primary" loading={save.isPending} disabled={!query.data || query.isFetching} onClick={() => save.mutate(draft ?? query.data?.systemPrompt ?? '')}>保存</Button>
    </>}>
      {query.isLoading ? <p>正在读取附加信息…</p> : query.error ? <p role="alert">读取失败：{query.error.message}</p> : <>
        <label className={s.label} htmlFor="conversation-instructions">特殊要求 / 附加信息（选填）</label>
        <textarea id="conversation-instructions" className={s.input} rows={6} value={draft ?? query.data?.systemPrompt ?? ''} onChange={event => setDraft(event.target.value)} placeholder="例如：多用几何图示，先讲直觉，再讲公式。" />
        <p className={s.hint}>可以随时修改或清空。这是当前对话的系统提示补充，保存后从下一次回答生效。</p>
      </>}
      {save.error && <p role="alert">保存失败：{save.error.message}</p>}
    </Modal>
  </>;
}
