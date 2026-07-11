// Centralized TanStack Query key factory. Keeping keys in one place
// prevents typo drift and makes invalidations predictable.
//
// Conventions:
//   ['scope']              — list
//   ['scope', id]          — detail
//   ['scope', id, sub]     — sub-resource

export const qk = {
  health: () => ['health'] as const,
  settings: {
    agentRuntime: () => ['settings', 'agent-runtime'] as const,
    appearance: () => ['settings', 'appearance'] as const,
  },

  projects: {
    all: () => ['projects'] as const,
    detail: (slug: string) => ['projects', slug] as const,
    tree: (slug: string) => ['projects', slug, 'tree'] as const,
    zone: (slug: string, zone: string) => ['projects', slug, 'zones', zone] as const,
  },

  agents: {
    all: () => ['agents'] as const,
  },

  sessions: {
    recent: (limit?: number) => ['sessions', 'recent', limit ?? 10] as const,
    active: () => ['sessions', 'active'] as const,
    detail: (id: string) => ['sessions', id] as const,
  },

  memory: {
    file: (projectSlug: string, filename: string) =>
      ['memory', projectSlug, filename] as const,
  },

  files: {
    raw: (projectSlug: string, relPath: string) =>
      ['files', projectSlug, relPath] as const,
  },

  confusions: {
    all: (projectSlug: string) => ['confusions', projectSlug] as const,
    detail: (projectSlug: string, id: string) =>
      ['confusions', projectSlug, id] as const,
  },

  folders: {
    all: () => ['folders'] as const,
  },
} as const;
