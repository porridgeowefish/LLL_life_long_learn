// practiceDraft — localStorage persistence for the Practice exam flow.
// Stores per-project drafts keyed by task fingerprint so a regenerate
// (new taskIds) does not bleed old answers into new questions (P-04/P-05).
//
// Two keys per project:
//   lll.draft.practice.{slug}            — { generatedAt, taskIds, drafts }
//   lll.practice.{slug}.lastAttempt      — last submitted attempt number

export interface DraftEntry {
  answer: string;
  selfAssess: number; // 0 = not yet self-assessed
}

export interface PracticeDraft {
  generatedAt: string;
  taskIds: string[];
  drafts: Record<string, DraftEntry>;
}

const DRAFT_PREFIX = 'lll.draft.practice';
const ATTEMPT_PREFIX = 'lll.practice';

function draftKey(slug: string) {
  return `${DRAFT_PREFIX}.${slug}`;
}

function attemptKey(slug: string) {
  return `${ATTEMPT_PREFIX}.${slug}.lastAttempt`;
}

/** Load a stored draft. Returns null if missing or corrupt. */
export function loadDraft(slug: string): PracticeDraft | null {
  try {
    const raw = localStorage.getItem(draftKey(slug));
    if (!raw) return null;
    const parsed = JSON.parse(raw) as PracticeDraft;
    if (!parsed || !Array.isArray(parsed.taskIds)) return null;
    return parsed;
  } catch {
    return null;
  }
}

/** Persist the full draft object. */
export function saveDraft(slug: string, draft: PracticeDraft): void {
  try {
    localStorage.setItem(draftKey(slug), JSON.stringify(draft));
  } catch {
    // quota exceeded — silently ignore (best-effort persistence)
  }
}

/** Remove the draft (e.g. after a successful submit). */
export function clearDraft(slug: string): void {
  try {
    localStorage.removeItem(draftKey(slug));
  } catch {
    // ignore
  }
}

/** True if the stored draft's fingerprint matches the current task batch. */
export function draftMatches(
  draft: PracticeDraft | null,
  taskIds: string[],
  generatedAt: string | undefined,
): boolean {
  if (!draft || !generatedAt) return false;
  if (draft.generatedAt !== generatedAt) return false;
  if (draft.taskIds.length !== taskIds.length) return false;
  return taskIds.every((id, i) => draft.taskIds[i] === id);
}

/** Read the last submitted attempt number (null if never submitted). */
export function getLastAttempt(slug: string): number | null {
  try {
    const raw = localStorage.getItem(attemptKey(slug));
    if (!raw) return null;
    const n = Number(raw);
    return Number.isFinite(n) && n > 0 ? n : null;
  } catch {
    return null;
  }
}

/** Record the last submitted attempt number. */
export function setLastAttempt(slug: string, attempt: number): void {
  try {
    localStorage.setItem(attemptKey(slug), String(attempt));
  } catch {
    // ignore
  }
}
