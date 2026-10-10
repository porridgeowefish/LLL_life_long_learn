import { useRetrySource, useSources, type LearningSource } from '@/features/learning/api/learningWorkspace';
import { SourceImportControl } from './SourceImportControl';

import s from './SourcesView.module.css';

// Imported materials automatically participate in the next teacher request.
export function SourcesView({ slug }: { slug: string }) {
  const sources = useSources(slug);
  const retry = useRetrySource(slug);
  const visible = (sources.data ?? []).filter((source) => source.status !== 'tombstoned' && source.status !== 'deleted');

  return (
    <section className={s.sources}>
      <header>
        <div><span>资料库</span><h2>当前文件</h2><p>导入后自动在本地提取文本，可供教师读取。支持 PDF、Markdown、TXT、PPTX、DOCX、XLSX、CSV，每份不超过 50 MB。</p><p>扫描版 PDF 暂不识别图片文字；旧版 DOC 请另存为 DOCX 后导入。</p></div>
        <SourceImportControl slug={slug} />
      </header>
      {sources.isError && <div className={s.error} role="alert">资料列表读取失败。<button type="button" onClick={() => void sources.refetch()}>重试</button></div>}
      {retry.isError && <div className={s.error} role="alert">重试解析失败：{retry.error.message}</div>}
      <section className={s.librarySection}>
        <div className={s.sectionHeading}><div><span>上传资料</span><h3>文件项</h3></div><small>{visible.length} 份</small></div>
        <div className={s.list}>
          {visible.map((source) => <article key={source.sourceId}>
            <div className={s.fileIcon}>{extension(source.displayName)}</div>
            <div className={s.fileDetails}><strong>{source.displayName}</strong><span className={source.status === 'failed' ? s.failure : ''}>{statusLabel(source)}</span>{source.extractionTruncated && <span className={s.warning}>文本过长，仅提取了部分内容；原文件已保留。</span>}</div>
            <div className={s.trailing}><time>{new Date(source.updatedAt).toLocaleString()}</time>{canRetry(source) && <button type="button" className={s.retryButton} disabled={retry.isPending} aria-label={`重试解析 ${source.displayName}`} onClick={() => retry.mutate(source.sourceId)}>{retry.isPending && retry.variables === source.sourceId ? '正在重试…' : '重试解析'}</button>}</div>
          </article>)}
          {!sources.isLoading && !visible.length && <div className={s.empty}>尚未上传资料</div>}
        </div>
      </section>
    </section>
  );
}

function extension(name: string) {
  const value = name.split('.').pop()?.toUpperCase() || 'FILE';
  return value.slice(0, 5);
}

function statusLabel(source: LearningSource) {
  const failures: Record<string, string> = {
    unsupported_legacy_format: '旧版 DOC 暂不支持，请另存为 DOCX 后重新导入。',
    unsupported_format: '暂不支持此格式，请转换为支持的格式后重新导入。',
    invalid_document: '文件格式无效或已损坏，请检查后重新导入。',
    encrypted_document: '文件已加密，请解密后重新导入。',
    resource_limit: '文件内容超出解析资源限制，请拆分后重新导入。',
    no_extractable_text: '无可提取文字，扫描件暂不支持识别。',
    parse_cancelled: '解析被中断，可以重试。',
    parse_failed: '解析未完成，可以重试。',
  };
  if (source.status === 'failed') return `解析失败：${failures[source.failureCode ?? ''] ?? '请检查文件是否损坏、加密或没有可提取文字，再重新导入。'}`;
  if (source.status === 'opaque' && canRetry(source)) return '原文件已保存，可重试本地解析';
  const labels: Record<string, string> = { stored: '已保存，等待解析', processing: '正在本地解析', ready: '解析完成，教师可读取', opaque: '已保存，暂不支持解析' };
  return labels[source.status] ?? source.status;
}

function canRetry(source: LearningSource) {
  return (source.status === 'failed' || source.status === 'opaque') && /\.(pdf|md|markdown|txt|pptx|docx|xlsx|csv)$/i.test(source.displayName);
}
