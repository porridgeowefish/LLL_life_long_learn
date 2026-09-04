// Connection slice — SSE connection state. Used by ConnectionBadge to
// render green/red/orange and to gate actions that need a live stream.

import { create } from 'zustand';

export type ConnectionStatus = 'idle' | 'connecting' | 'open' | 'reconnecting' | 'error';

interface ConnectionState {
  status: ConnectionStatus;
  lastEventAt: number | null; // epoch ms
  error: string | null;
  setStatus: (s: ConnectionStatus) => void;
  markEvent: () => void;
  setError: (msg: string | null) => void;
}

export const useConnectionStore = create<ConnectionState>()((set) => ({
  status: 'idle',
  lastEventAt: null,
  error: null,
  setStatus: (status) => set({ status }),
  markEvent: () => set({ lastEventAt: Date.now() }),
  setError: (error) => set({ error }),
}));
