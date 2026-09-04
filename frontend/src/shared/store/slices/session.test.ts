import { describe, expect, it } from 'vitest';

import { useSessionStore } from './session';

describe('useSessionStore', () => {
  it('appends deltas to the outputBuffer', () => {
    useSessionStore.getState().clearBuffer();
    useSessionStore.getState().appendDelta('hello ');
    useSessionStore.getState().appendDelta('world');
    expect(useSessionStore.getState().outputBuffer).toBe('hello world');
  });

  it('clears the buffer when active session changes', () => {
    useSessionStore.getState().appendDelta('leftover');
    useSessionStore.getState().setActiveSession('sess-2');
    expect(useSessionStore.getState().outputBuffer).toBe('');
    expect(useSessionStore.getState().activeSessionId).toBe('sess-2');
  });

  it('replaceBuffer replaces the entire buffer', () => {
    useSessionStore.getState().appendDelta('a');
    useSessionStore.getState().replaceBuffer('full snapshot');
    expect(useSessionStore.getState().outputBuffer).toBe('full snapshot');
  });
});
