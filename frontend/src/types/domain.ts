// Domain types — canonical LLL entity shapes. Mirror the Go side at
// backend-go/internal/workspace/workspace.go and agentregistry/registry.go.
// Keep these the single source of truth; api.ts DTOs extend / pick from them.

export type ZoneName = 'Intro' | 'Explain' | 'Practice' | 'Extend' | 'Summary';

export const ALL_ZONES: ZoneName[] = [
  'Intro',
  'Explain',
  'Practice',
  'Extend',
  'Summary',
];

// Chinese display names — zone ID stays English internally, only the
// user-visible label changes. Mirrors the sidebar timeline labels.
export const ZONE_DISPLAY: Record<ZoneName, string> = {
  Intro: '引入',
  Explain: '讲解',
  Practice: '练习',
  Extend: '拓展',
  Summary: '总结',
};

// ZoneName → canonical file each zone writes to (mirrors zoneFilenames in
// backend-go/internal/workspace/workspace.go:38).
export const ZONE_FILENAME: Record<ZoneName, string> = {
  Intro: 'output.md',
  Explain: 'output.md',
  Practice: 'tasks.json',
  Extend: 'prompts.md',
  Summary: 'summary.md',
};

export interface ArtifactRef {
  zoneName: ZoneName;
  filename: string;
  sessionId?: string;
  runDirRel?: string;
  writtenAt: string; // ISO timestamp
}

export interface ProjectState {
  id: string;
  title: string;
  slug: string;
  parentProjectId?: string;
  status: string;
  activeZone: ZoneName | '';
  createdAt: string;
  updatedAt: string;
  childProjectIds?: string[];
  lastArtifacts?: ArtifactRef[];
  generatedZones?: ZoneName[];
}

export interface ProjectMeta {
  id: string;
  slug: string;
  title: string;
  parentProjectId?: string;
  hasSubprojects: boolean;
}

export interface PredecessorFile {
  zoneName: ZoneName;
  path: string; // absolute
  relPath: string; // relative to project root
  exists: boolean;
}

export type SessionStatus =
  | 'preparing'
  | 'launching'
  | 'running'
  | 'completed'
  | 'failed'
  | 'cancelled';

export interface SessionTurn {
  id: number;
  ordinal: number;
  type: 'user' | 'assistant' | 'system';
  content: string;
  createdAt: string;
  runDirRel?: string;
}

export interface Session {
  id: string;
  projectSlug: string;
  zoneName: ZoneName;
  agentId: string;
  state: SessionStatus | 'awaiting-follow-up';
  createdAt: string;
  finishedAt?: string;
  runDirRel?: string;
  promptPath?: string;
  turns: SessionTurn[];
  exitCode?: number;
  lastMessage?: string;
}

export interface OutputTarget {
  zone: ZoneName;
  filename: string;
}

export interface AgentPrimitives {
  required?: string[];
  optional?: string[];
}

export interface Agent {
  id: string;
  name: string;
  icon: string; // single emoji or empty
  description?: string;
  userStory: string;
  primitives: AgentPrimitives;
  allowedZones: ZoneName[];
  charterPath: string;
  defaultOutputTargets: OutputTarget[];
  charterText?: string; // loaded by registry; optional on the wire
}

export interface HealthStats {
  projects: number;
  sessions: number;
  turns: number;
  activeSessions: number;
}

export interface HealthResponse {
  ok: boolean;
  workspace: string;
  claude: {
    bin: string;
    available: boolean;
  };
  stats: HealthStats;
}
