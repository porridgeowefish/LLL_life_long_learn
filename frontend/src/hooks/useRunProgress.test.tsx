import { describe, it, expect, beforeEach, vi } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import type { Session } from '@/types/domain';

import { useRunProgress } from './useRunProgress';
import { useSessionStore } from '@/store/slices/session';

let handlers: Record<string, (data: unknown) => void> = {};
let activeSessions: Session[] = [];

vi.mock('@/hooks/useSSE', () => ({
  subscribeToSSE: (name: string, h: (data: unknown) => void) => {
    handlers[name] = h;
    return () => {
      delete handlers[name];
    };
  },
}));
vi.mock('@/api/sessions', () => ({
  useActiveSessions: () => ({ data: { sessions: activeSessions } }),
}));

function sessionOf(id: string, slug: string): Session {
  return { id, projectSlug: slug, zoneName: 'Explain', state: 'running' } as unknown as Session;
}

describe('useRunProgress', () => {
  beforeEach(() => {
    handlers = {};
    activeSessions = [];
    useSessionStore.setState({ activeSessionId: null });
  });

  it('is active as soon as a run is known active, before any event (invoke path)', () => {
    useSessionStore.setState({ activeSessionId: 's1' });
    const { result } = renderHook(() => useRunProgress('myproj'));
    expect(result.current.active).toBe(true);
    expect(result.current.activity).toBeNull(); // the bar falls back to "运行中…"
  });

  it('survives a refresh: active when the backend has an active session for this project, even with no client activeSessionId', () => {
    activeSessions = [sessionOf('s1', 'myproj')];
    useSessionStore.setState({ activeSessionId: null }); // simulate post-refresh
    const { result } = renderHook(() => useRunProgress('myproj'));
    expect(result.current.active).toBe(true);
  });

  it('is inactive when no run is active for this project', () => {
    activeSessions = [sessionOf('s1', 'other')]; // a different project
    const { result } = renderHook(() => useRunProgress('myproj'));
    expect(result.current.active).toBe(false);
  });

  it('artifact-updated sets the activity label', () => {
    useSessionStore.setState({ activeSessionId: 's1' });
    const { result } = renderHook(() => useRunProgress('myproj'));
    act(() => handlers['artifact-updated']({ projectSlug: 'myproj', zone: 'explain' }));
    expect(result.current.activity).toContain('explain');
  });

  it('ignores artifact events for other projects', () => {
    useSessionStore.setState({ activeSessionId: 's1' });
    const { result } = renderHook(() => useRunProgress('myproj'));
    act(() => handlers['artifact-updated']({ projectSlug: 'other', zone: 'explain' }));
    expect(result.current.activity).toBeNull();
  });

  it('hides on session-completed', () => {
    useSessionStore.setState({ activeSessionId: 's1' });
    const { result } = renderHook(() => useRunProgress('myproj'));
    expect(result.current.active).toBe(true);
    act(() => handlers['session-completed']({}));
    expect(result.current.active).toBe(false);
  });

  it('dismiss hides the bar', () => {
    useSessionStore.setState({ activeSessionId: 's1' });
    const { result } = renderHook(() => useRunProgress('myproj'));
    act(() => result.current.dismiss());
    expect(result.current.active).toBe(false);
  });

  it('run-progress correlates by runId and exposes pages', () => {
    useSessionStore.setState({ activeSessionId: 's1' });
    const { result } = renderHook(() => useRunProgress('myproj'));
    act(() => handlers['run-progress']({ runId: 'other', pagesDone: 9, pagesPlanned: 10 }));
    expect(result.current.pagesDone).toBe(0);
    expect(result.current.pagesPlanned).toBe(0);
    act(() => handlers['run-progress']({ runId: 's1', pagesDone: 3, pagesPlanned: 5, activity: 'wrote p3' }));
    expect(result.current.pagesDone).toBe(3);
    expect(result.current.pagesPlanned).toBe(5);
    expect(result.current.activity).toBe('wrote p3');
  });

  it('resets pages when the tracked run changes', () => {
    useSessionStore.setState({ activeSessionId: 's1' });
    const { result } = renderHook(() => useRunProgress('myproj'));
    act(() => handlers['run-progress']({ runId: 's1', pagesDone: 3, pagesPlanned: 5 }));
    expect(result.current.pagesPlanned).toBe(5);
    act(() => useSessionStore.setState({ activeSessionId: 's2' }));
    expect(result.current.pagesDone).toBe(0);
    expect(result.current.pagesPlanned).toBe(0);
  });
});
