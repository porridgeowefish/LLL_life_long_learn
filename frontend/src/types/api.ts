// API DTOs — request/response shapes for the Go backend.
// Distinct from domain types when the wire format differs from the canonical
// in-memory shape (e.g. list endpoints wrap arrays in {<name>: [...]}).

import type {
  Agent,
  AgentRuntimeHealth,
  HealthResponse,
  PredecessorFile,
  ProjectMeta,
  ProjectType,
  ProjectState,
  Session,
  ZoneName,
} from './domain';

// Responses
export interface ProjectsListResponse {
  projects: ProjectMeta[];
}

export interface ProjectResponse {
  project: ProjectState;
}

export interface DeleteProjectResponse {
  deleted: true;
  projectId: string;
}

export interface AgentsListResponse {
  agents: Agent[];
}

export type AgentRuntimeSettingsResponse = AgentRuntimeHealth;

export interface SessionsListResponse {
  sessions: Session[];
}

export interface ActiveSessionsResponse {
  sessions: Session[];
}

export interface ZoneResponse {
  zone: ZoneName;
  predecessors: PredecessorFile[];
}

// Requests
export interface CreateProjectRequest {
  title: string;
  projectType: ProjectType;
  slug?: string;
  why?: string;
  current?: string;
  target?: string;
  standard?: string;
}

export interface ProjectTypeAdviceRequest {
  title?: string;
  why?: string;
  current: string;
  target: string;
  standard?: string;
  messages: Array<{
    role: 'user' | 'assistant';
    content: string;
  }>;
}

export interface ProjectTypeAdviceResponse {
  reply: string;
  recommendation?: ProjectType;
  reason?: string;
  tradeoff?: string;
  confidence: 'low' | 'medium' | 'high';
}

export interface DisciplineOverviewResponse {
  title: string;
  content: string;
}

export interface DisciplineOverviewGenerationResponse {
  session: Session;
  runDir: string;
}

export interface InvokeAgentRequest {
  projectId: string;
  zone: ZoneName;
  intent?: string;
  permissionMode?: string;
  sourceRefs?: string[];
  parentPageId?: string;
  practiceAttempt?: number;
  practiceQuestionCount?: number;
}

export interface FollowUpRequest {
  text: string;
  permissionMode?: string;
}

// Re-export the health shape for convenience — it doubles as a DTO.
export type { HealthResponse };
