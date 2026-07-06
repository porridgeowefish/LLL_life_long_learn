import { describe, it, expect, beforeEach, vi } from 'vitest';
import { renderHook, act } from '@testing-library/react';

import { useRunProgress } from './useRunProgress';
import { useSessionStore } from '@/store/slices/session';

let handlers: Record<string, (data: unknown) => void> = {};
vi.mock('@/hooks/useSSE', () => ({
  subscribeToSSE: (name: string, h: (data: unknown) => void) => {
    handlers[name] = h;
    return () => {
      delete handlers[name];
    };
  },
}));

describe('useRunProgress', () => {
  beforeEach(() => {
    handlers = {};
    useSessionStore.setState({ activeSessionId: null });
  });

  it('is inactive until an artifact event arrives for this slug', () => {
    useSessionStore.setState({ activeSessionId: 's1' });
    const { result } = renderHook(() => useRunProgress('myproj'));
    expect(result.current.active).toBe(false);
    act(() => handlers['artifact-updated']({ projectSlug: 'myproj', zone: 'explain' }));
    expect(result.current.active).toBe(true);
    expect(result.current.activity).toContain('explain');
  });

  it('ignores artifact events for other projects', () => {
    useSessionStore.setState({ activeSessionId: 's1' });
    const { result } = renderHook(() => useRunProgress('myproj'));
    act(() => handlers['artifact-updated']({ projectSlug: 'other', zone: 'explain' }));
    expect(result.current.active).toBe(false);
  });

  it('hides on session-completed', () => {
    useSessionStore.setState({ activeSessionId: 's1' });
    const { result } = renderHook(() => useRunProgress('myproj'));
    act(() => handlers['artifact-updated']({ projectSlug: 'myproj', zone: 'explain' }));
    expect(result.current.active).toBe(true);
    act(() => handlers['session-completed']({}));
    expect(result.current.active).toBe(false);
  });

  it('dismiss hides the bar', () => {
    useSessionStore.setState({ activeSessionId: 's1' });
    const { result } = renderHook(() => useRunProgress('myproj'));
    act(() => handlers['artifact-updated']({ projectSlug: 'myproj', zone: 'explain' }));
    act(() => result.current.dismiss());
    expect(result.current.active).toBe(false);
  });
});
