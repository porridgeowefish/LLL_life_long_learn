// Browser-only project data is not covered by the server-side directory
// deletion. Clear every known per-project draft after the DELETE succeeds.
export function clearDeletedProjectClientData(slug: string): void {
  try {
    localStorage.removeItem(`lll.draft.practice.${slug}`);
    localStorage.removeItem(`lll.practice.${slug}.lastAttempt`);
    const markdownDraftPrefix = `lll.draft.${slug}.`;
    for (let index = localStorage.length - 1; index >= 0; index -= 1) {
      const key = localStorage.key(index);
      if (key?.startsWith(markdownDraftPrefix)) {
        localStorage.removeItem(key);
      }
    }
  } catch {
    // Storage may be unavailable in privacy-restricted browser contexts.
  }
}
