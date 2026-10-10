import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { http } from '@/shared/client';
export interface AutoNotesSettings { enabled: boolean; everyTurns: number; providerId: string; model: string }
export interface NoteJob { id: string; status: 'queued' | 'running' | 'failed' | 'succeeded'; fromTurn: number; throughTurn: number; errorCode?: string; retryable: boolean }
export interface NoteEntry { id: string; jobId: string; content: string; fromTurn: number; throughTurn: number; createdAt: string; providerId: string; model: string }
export interface NotesResource { entries: NoteEntry[]; jobs: NoteJob[] }
export const noteKeys = { settings: ['teacher-settings'] as const, notes: (slug: string) => ['learning-workspace', slug, 'notes'] as const };
export function useTeacherSettings() { return useQuery({ queryKey: noteKeys.settings, queryFn: () => http.get<{ autoNotes: AutoNotesSettings }>('/api/settings/teacher') }); }
export function useSaveTeacherSettings() { const client = useQueryClient(); return useMutation({ mutationFn: (autoNotes: AutoNotesSettings) => http.put<{ autoNotes: AutoNotesSettings }>('/api/settings/teacher', { autoNotes }), onSuccess: data => { client.setQueryData(noteKeys.settings, data); } }); }
export function useNotes(slug: string) { return useQuery({ queryKey: noteKeys.notes(slug), queryFn: () => http.get<NotesResource>(`/api/projects/${encodeURIComponent(slug)}/notes`), enabled: !!slug, refetchInterval: query => query.state.data?.jobs.some(job => job.status === 'queued' || job.status === 'running') ? 2500 : false }); }
export function useRetryNote(slug: string) { const client = useQueryClient(); return useMutation({ mutationFn: (jobId: string) => http.post<NoteJob>(`/api/projects/${encodeURIComponent(slug)}/notes/jobs/${encodeURIComponent(jobId)}/retry`, {}), onSuccess: () => { void client.invalidateQueries({ queryKey: noteKeys.notes(slug) }); } }); }
