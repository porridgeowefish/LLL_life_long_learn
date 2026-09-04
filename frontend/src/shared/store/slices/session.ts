// Session slice — active session id + SSE-driven output buffer. NOT
// persisted (outputBuffer can grow large; on reconnect the detail
// endpoint serves the canonical state).

import { create } from 'zustand';

interface SessionState {
  activeSessionId: string | null;
  // outputBuffer accumulates SSE terminal-output deltas for the active
  // session. Reset whenever activeSessionId changes.
  outputBuffer: string;
  // The terminal-output stream emits chunked text deltas; if backend ever
  // emits a full-snapshot event we can replace the buffer wholesale.
  setActiveSession: (id: string | null) => void;
  appendDelta: (chunk: string) => void;
  replaceBuffer: (text: string) => void;
  clearBuffer: () => void;
}

export const useSessionStore = create<SessionState>()((set) => ({
  activeSessionId: null,
  outputBuffer: '',
  setActiveSession: (id) =>
    set({
      activeSessionId: id,
      // Switching session invalidates any prior buffer content.
      outputBuffer: '',
    }),
  appendDelta: (chunk) =>
    set((s) => ({ outputBuffer: s.outputBuffer + chunk })),
  replaceBuffer: (text) => set({ outputBuffer: text }),
  clearBuffer: () => set({ outputBuffer: '' }),
}));
