// Application-wide constants. Pull magic strings out of components.

export const API_BASE = ''; // Same-origin in prod, proxied by Vite in dev.

export const HEALTH_POLL_INTERVAL_MS = 30_000;

// SSE event names emitted by backend-go/internal/httpx/sse.go.
export const SSE_EVENTS = {
  hello: 'hello',
  sessionCreated: 'session-created',
  sessionState: 'session-state',
  turnCreated: 'turn-created',
  terminalOutput: 'terminal-output',
  artifactUpdated: 'artifact-updated',
  confusionUpdated: 'confusion-updated',
  sessionCompleted: 'session-completed',
  sessionFailed: 'session-failed',
} as const;

export type SSEEventName = (typeof SSE_EVENTS)[keyof typeof SSE_EVENTS];

// localStorage keys — kept in one place to avoid typo drift across slices.
export const STORAGE_KEYS = {
  uiSidebarCollapsed: 'lll.ui.sidebarCollapsed',
  projectSelectedId: 'lll.project.selectedId',
  projectCurrentZone: 'lll.project.currentZone',
} as const;

// Permission modes for Claude CLI (see backend-go/internal/claudelauncher).
export const PERMISSION_MODES = {
  default: 'default',
  acceptEdits: 'acceptEdits',
  bypassPermissions: 'bypassPermissions',
} as const;

export type PermissionMode = (typeof PERMISSION_MODES)[keyof typeof PERMISSION_MODES];
