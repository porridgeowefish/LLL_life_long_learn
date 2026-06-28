// API DTOs — request/response shapes for the Go backend.
// Distinct from domain types when the wire format differs from the canonical
// in-memory shape (e.g. list endpoints wrap arrays in {<name>: [...]}).

import type {
  Agent,
  AgentRuntimeHealth,
  HealthResponse,
  PredecessorFile,
  ProjectMeta,
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
  slug?: string;
  parentProjectId?: string;
  why?: string;
  current?: string;
  target?: string;
  standard?: string;
}

export interface CreateSubprojectRequest {
  title: string;
  slug?: string;
  why?: string;
  current?: string;
  target?: string;
  standard?: string;
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
