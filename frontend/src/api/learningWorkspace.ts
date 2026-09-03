import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useMemo } from 'react';

import { ApiError, http } from './client';

export interface MessageBlock {
  id: string;
  type: 'markdown' | 'reasoning-summary' | 'attachment';
  source?: string;
  artifactRef?: string;
}

export interface ConversationMessage {
  id: string;
  role: 'learner' | 'teacher' | 'system';
  status: 'completed' | 'failed' | 'interrupted';
  blocks: MessageBlock[];
  createdAt: string;
}

export interface TaskLink {
  messageId: string;
  taskId: string;
  toolCallId?: string;
}

export interface ConversationProjection {
  conversationId: string;
  unitId: string;
  latestSeq: number;
  pageFromSeq: number;
  pageThroughSeq: number;
  hasMore: boolean;
  hasPrevious: boolean;
  totalMessages: number;
  messages: ConversationMessage[];
  taskLinks: TaskLink[];
}

export interface AssistantTask {
  id: string;
  type: 'consolidate' | 'verify' | 'produce-material' | 'source-processing';
  objective: string;
  status: 'queued' | 'running' | 'succeeded' | 'partial' | 'failed' | 'cancelled';
  phase?: 'preparing' | 'executing' | 'validating' | 'committing';
  result?: { summary: string; deliverables?: string[]; assetUpdates?: Record<string, string> };
  failure?: { code: string; message: string; suggestion?: string };
  createdAt: string;
  updatedAt: string;
}

export interface AssetMeta {
  assetId: string;
  key: 'intro' | 'body' | 'practice';
  title: string;
  currentVersionId: string;
  editRevision: number;
  conversationCursor: number;
  updatedAt: string;
}

export interface LearningAsset {
  meta: AssetMeta;
  content: string;
  contentKind: 'markdown' | 'explain-pages' | 'practice-set';
}

export interface GeneratedArtifactSummary {
  artifactId: string;
  kind: string;
  title: string;
  description: string;
  entryPoint?: string;
  entryMediaType?: string;
  fileCount: number;
  createdAt: string;
}

export interface BodyAnnotationMessage {
  id: string;
  role: 'learner' | 'assistant';
  content: string;
  status: 'completed' | 'interrupted' | 'failed';
  createdAt: string;
}

export interface BodyAnnotation {
  schemaVersion: number;
  annotationId: string;
  assetId: string;
  assetVersionId: string;
  quoteSnapshot: string;
  anchors: { start: number; end: number; prefix?: string; suffix?: string };
  note?: string;
  status: 'open' | 'asked' | 'resolved';
  detached?: boolean;
  createdAt: string;
  updatedAt: string;
  ask?: { messages: BodyAnnotationMessage[]; summary?: string; summaryState?: string };
}

export interface LearningSource {
  sourceId: string;
  displayName: string;
  status: 'stored' | 'processing' | 'ready' | 'opaque' | 'failed' | 'tombstoned' | 'deleted';
  currentRevisionId: string;
  failureCode?: string;
  createdAt: string;
  updatedAt: string;
}

export interface SourceFileRef {
  key?: string;
  path: string;
  filename?: string;
  mediaType: string;
  bytes: number;
  sha256: string;
}

export interface SourceRevision {
  revisionId: string;
  sourceId: string;
  original: SourceFileRef;
  derivedFiles: SourceFileRef[];
  parseTaskId?: string;
  createdAt: string;
}

export type TeacherStreamFrame =
  | { type: 'turn-accepted'; data: { learnerMessageId: string; responseId: string } }
  | { type: 'message-started'; data: { teacherMessageId: string } }
  | { type: 'text-delta' | 'reasoning-summary-delta'; data: { blockId: string; delta: string } }
  | { type: 'task-accepted'; data: { toolCallId: string; taskId: string; status: string } }
  | { type: 'tool-rejected'; data: { toolCallId: string; code: string; existingTaskId?: string } }
  | { type: 'message-completed'; data: { messageId: string; latestSeq: number } }
  | { type: 'message-failed'; data: { messageId?: string; code: string; partialPreserved: boolean } }
  | { type: string; data: Record<string, unknown> };

export const learningWorkspaceKeys = {
  conversation: (slug: string) => ['learning-workspace', slug, 'conversation'] as const,
  tasks: (slug: string) => ['learning-workspace', slug, 'tasks'] as const,
  assets: (slug: string) => ['learning-workspace', slug, 'assets'] as const,
  asset: (slug: string, key: string) => ['learning-workspace', slug, 'assets', key] as const,
  generated: (slug: string) => ['learning-workspace', slug, 'generated'] as const,
  generatedEntry: (slug: string, artifactId: string) => ['learning-workspace', slug, 'generated', artifactId, 'entry'] as const,
  annotations: (slug: string) => ['learning-workspace', slug, 'annotations'] as const,
  sources: (slug: string) => ['learning-workspace', slug, 'sources'] as const,
  source: (slug: string, sourceId: string) => ['learning-workspace', slug, 'sources', sourceId] as const,
};

const keys = learningWorkspaceKeys;

export function useConversation(slug: string) {
  const query = useInfiniteQuery({
    queryKey: keys.conversation(slug),
    queryFn: ({ pageParam }) => http.get<ConversationProjection>(`/api/projects/${encodeURIComponent(slug)}/conversation?limit=40&beforeSeq=${pageParam}`),
    initialPageParam: 0,
    getNextPageParam: (page) => page.hasPrevious && page.pageFromSeq > 0 ? page.pageFromSeq : undefined,
    enabled: Boolean(slug),
  });
  const data = useMemo(() => {
    const pages = query.data?.pages;
    if (!pages?.length) return undefined;
    const newest = pages[0];
    const messages = new Map<string, ConversationMessage>();
    const taskLinks = new Map<string, TaskLink>();
    for (const page of [...pages].reverse()) {
      for (const message of page.messages ?? []) messages.set(message.id, { ...message, blocks: message.blocks ?? [] });
      for (const link of page.taskLinks ?? []) taskLinks.set(`${link.messageId}:${link.taskId}`, link);
    }
    return {
      ...newest,
      pageFromSeq: pages[pages.length - 1].pageFromSeq,
      hasMore: Boolean(query.hasNextPage),
      hasPrevious: Boolean(query.hasNextPage),
      messages: [...messages.values()],
      taskLinks: [...taskLinks.values()],
    } satisfies ConversationProjection;
  }, [query.data?.pages, query.hasNextPage]);
  return {
    ...query,
    data,
    hasPrevious: Boolean(query.hasNextPage),
    loadPrevious: query.fetchNextPage,
    isLoadingPrevious: query.isFetchingNextPage,
  };
}

export function useAssistantTasks(slug: string) {
  return useQuery({ queryKey: keys.tasks(slug), queryFn: async () => (await http.get<{ tasks: AssistantTask[] }>(`/api/projects/${encodeURIComponent(slug)}/assistant-tasks`)).tasks, enabled: Boolean(slug), refetchInterval: (query) => query.state.data?.some((task) => task.status === 'queued' || task.status === 'running') ? 2500 : false });
}

export function useAssets(slug: string) {
  return useQuery({ queryKey: keys.assets(slug), queryFn: async () => (await http.get<{ assets: AssetMeta[] }>(`/api/projects/${encodeURIComponent(slug)}/assets`)).assets, enabled: Boolean(slug) });
}

export function useAsset(slug: string, key: string) {
  return useQuery({ queryKey: keys.asset(slug, key), queryFn: () => http.get<LearningAsset>(`/api/projects/${encodeURIComponent(slug)}/assets/${key}`), enabled: Boolean(slug && key) });
}

export function useGeneratedArtifacts(slug: string) {
  return useQuery({
    queryKey: keys.generated(slug),
    queryFn: async () => (await http.get<{ artifacts: GeneratedArtifactSummary[] }>(`/api/projects/${encodeURIComponent(slug)}/generated`)).artifacts,
    enabled: Boolean(slug),
  });
}

function generatedFileURL(slug: string, artifact: GeneratedArtifactSummary) {
  if (!artifact.entryPoint) return '';
  const path = artifact.entryPoint.split('/').map(encodeURIComponent).join('/');
  return `/api/projects/${encodeURIComponent(slug)}/generated/${encodeURIComponent(artifact.artifactId)}/files/${path}`;
}

export function generatedArtifactOpenURL(slug: string, artifactId: string) {
  return `/api/projects/${encodeURIComponent(slug)}/generated/${encodeURIComponent(artifactId)}/open`;
}

export function useGeneratedArtifactEntry(slug: string, artifact?: GeneratedArtifactSummary) {
  const previewable = artifact?.entryMediaType === 'text/markdown' || artifact?.entryMediaType === 'text/plain';
  return useQuery({
    queryKey: keys.generatedEntry(slug, artifact?.artifactId ?? ''),
    queryFn: () => http.get<string>(generatedFileURL(slug, artifact!), { rawText: true }),
    enabled: Boolean(slug && artifact?.entryPoint && previewable),
  });
}

export function useSaveAsset(slug: string, key: string) {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (input: { baseEditRevision: number; content: string }) => http.put<LearningAsset>(`/api/projects/${encodeURIComponent(slug)}/assets/${key}`, input),
    onSuccess: (asset) => {
      client.setQueryData(keys.asset(slug, key), asset);
      void client.invalidateQueries({ queryKey: keys.assets(slug) });
    },
  });
}

export function useBodyAnnotations(slug: string) {
  return useQuery({
    queryKey: keys.annotations(slug),
    queryFn: async () => (await http.get<{ annotations: BodyAnnotation[] }>(`/api/projects/${encodeURIComponent(slug)}/assets/body/annotations`)).annotations,
    enabled: Boolean(slug),
  });
}

export function useCreateBodyAnnotation(slug: string) {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (input: { assetVersionId: string; quoteSnapshot: string; anchors: BodyAnnotation['anchors']; note?: string }) =>
      http.post<{ annotation: BodyAnnotation }>(`/api/projects/${encodeURIComponent(slug)}/assets/body/annotations`, input),
    onSuccess: () => void client.invalidateQueries({ queryKey: keys.annotations(slug) }),
  });
}

export function useDeleteBodyAnnotation(slug: string) {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (annotationId: string) => http.delete(`/api/projects/${encodeURIComponent(slug)}/assets/body/annotations/${encodeURIComponent(annotationId)}`),
    onSuccess: () => void client.invalidateQueries({ queryKey: keys.annotations(slug) }),
  });
}

export function useSources(slug: string) {
  return useQuery({ queryKey: keys.sources(slug), queryFn: async () => (await http.get<{ sources: LearningSource[] }>(`/api/projects/${encodeURIComponent(slug)}/sources`)).sources, enabled: Boolean(slug), refetchInterval: (query) => query.state.data?.some((source) => source.status === 'processing') ? 2500 : false });
}

export function useSource(slug: string, sourceId: string) {
  return useQuery({
    queryKey: keys.source(slug, sourceId),
    queryFn: () => http.get<{ source: LearningSource; revision: SourceRevision }>(`/api/projects/${encodeURIComponent(slug)}/sources/${encodeURIComponent(sourceId)}`),
    enabled: Boolean(slug && sourceId),
  });
}

export function sourceFileURL(slug: string, sourceId: string, revisionId: string, fileKey: string) {
  return `/api/projects/${encodeURIComponent(slug)}/sources/${encodeURIComponent(sourceId)}/revisions/${encodeURIComponent(revisionId)}/files/${encodeURIComponent(fileKey)}`;
}

export function useUploadSource(slug: string) {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async ({ file, parseApproved, cloudDisclosureAccepted }: { file: File; parseApproved: boolean; cloudDisclosureAccepted: boolean }) => {
      const form = new FormData();
      form.set('operationId', crypto.randomUUID());
      form.set('displayName', file.name);
      form.set('parseApproved', String(parseApproved));
      form.set('cloudDisclosureAccepted', String(cloudDisclosureAccepted));
      form.set('file', file);
      const response = await fetch(`/api/projects/${encodeURIComponent(slug)}/sources`, { method: 'POST', body: form });
      const payload = await response.json().catch(() => null);
      if (!response.ok) throw new ApiError(response.status, '资料上传失败', payload);
      return payload as { source: LearningSource; task?: AssistantTask };
    },
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: keys.sources(slug) });
      void client.invalidateQueries({ queryKey: keys.tasks(slug) });
    },
  });
}

export function useTombstoneSource(slug: string) {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (sourceId: string) => http.delete(`/api/projects/${encodeURIComponent(slug)}/sources/${encodeURIComponent(sourceId)}`),
    onSuccess: () => void client.invalidateQueries({ queryKey: keys.sources(slug) }),
  });
}

export function usePermanentlyDeleteSource(slug: string) {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (sourceId: string) => http.post(`/api/projects/${encodeURIComponent(slug)}/sources/${encodeURIComponent(sourceId)}/permanent-delete`, { operationId: crypto.randomUUID(), confirmation: 'PERMANENT_DELETE' }),
    onSuccess: () => void client.invalidateQueries({ queryKey: keys.sources(slug) }),
  });
}

export async function streamTeacherTurn(slug: string, input: { operationId: string; content: string; attachmentRefs?: string[]; providerId?: string }, signal: AbortSignal, onFrame: (frame: TeacherStreamFrame) => void) {
  const response = await fetch(`/api/projects/${encodeURIComponent(slug)}/conversation/turns`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(input), signal });
  if (!response.ok || !response.body) throw new ApiError(response.status, '教师响应无法开始', await response.text());
  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  let buffer = '';
  while (true) {
    const { value, done } = await reader.read();
    buffer += decoder.decode(value, { stream: !done });
    const chunks = buffer.split(/\r?\n\r?\n/);
    buffer = chunks.pop() ?? '';
    for (const chunk of chunks) {
      let type = '';
      let data = '';
      for (const line of chunk.split(/\r?\n/)) {
        if (line.startsWith('event:')) type = line.slice(6).trim();
        if (line.startsWith('data:')) data += line.slice(5).trim();
      }
      if (type && data) onFrame({ type, data: JSON.parse(data) } as TeacherStreamFrame);
    }
    if (done) break;
  }
}

// Reconnects to a teacher response that is still running in the backend. The
// backend owns the generation; this request is only a disposable subscriber.
export async function resumeTeacherTurn(slug: string, signal: AbortSignal, onFrame: (frame: TeacherStreamFrame) => void) {
  const response = await fetch(`/api/projects/${encodeURIComponent(slug)}/conversation/responses/active`, { signal });
  if (response.status === 204) return false;
  if (!response.ok || !response.body) throw new ApiError(response.status, '教师响应无法恢复', await response.text());
  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  let buffer = '';
  while (true) {
    const { value, done } = await reader.read();
    buffer += decoder.decode(value, { stream: !done });
    const chunks = buffer.split(/\r?\n\r?\n/);
    buffer = chunks.pop() ?? '';
    for (const chunk of chunks) {
      let type = '';
      let data = '';
      for (const line of chunk.split(/\r?\n/)) {
        if (line.startsWith('event:')) type = line.slice(6).trim();
        if (line.startsWith('data:')) data += line.slice(5).trim();
      }
      if (type && data) onFrame({ type, data: JSON.parse(data) } as TeacherStreamFrame);
    }
    if (done) break;
  }
  return true;
}

export async function stopTeacherResponse(slug: string, responseId: string) {
  return http.post(`/api/projects/${encodeURIComponent(slug)}/conversation/responses/${encodeURIComponent(responseId)}/stop`, { operationId: crypto.randomUUID() });
}

export function invalidateLearningWorkspace(client: ReturnType<typeof useQueryClient>, slug: string) {
  void client.invalidateQueries({ queryKey: ['learning-workspace', slug] });
}
