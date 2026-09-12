import { useTeacherUsage } from './api';
import { UsageHeatmap } from './UsageHeatmap';
import s from './UsagePage.module.css';

export function UsagePage() {
  const usage = useTeacherUsage();
  return <main className={s.page}>
    <header><span>USAGE</span><h1>教师 Token 用量</h1><p>按一段持久教师对话统计。当前仅记录模型服务实际返回的教师用量；助教用量暂不展示。</p></header>
    <UsageHeatmap />
    {usage.isLoading && <p className={s.muted}>正在读取用量…</p>}
    {usage.isError && <p className={s.error}>读取用量失败，请稍后重试。</p>}
    {!usage.isLoading && !usage.isError && !usage.data?.conversations.length && <p className={s.empty}>还没有可统计的教师用量。完成一次有 provider usage 返回的对话后会出现在这里。</p>}
    <section className={s.list}>
      {(usage.data?.conversations ?? []).map((conversation) => <article key={conversation.conversationId}>
        <header><div><span>学习单元</span><h2>{conversation.title}</h2><small>{conversation.projectSlug}</small></div><strong>输入 {conversation.inputTokens.toLocaleString()} · 输出 {conversation.outputTokens.toLocaleString()}</strong></header>
        <details><summary>{conversation.turns.length} 个教师回合</summary><ol>{conversation.turns.map((turn) => <li key={turn.responseId}><time>{new Date(turn.occurredAt).toLocaleString()}</time><span>输入 {turn.inputTokens.toLocaleString()} · 输出 {turn.outputTokens.toLocaleString()}</span>{turn.providerId && <small>{turn.providerId}</small>}</li>)}</ol></details>
      </article>)}
    </section>
  </main>;
}
