import { beforeEach, describe, expect, it } from 'vitest';

import { clearDeletedProjectClientData } from './projectDeletion';

describe('clearDeletedProjectClientData', () => {
  beforeEach(() => localStorage.clear());

  it('removes only browser drafts associated with the deleted project', () => {
    localStorage.setItem('lll.draft.practice.target', '{}');
    localStorage.setItem('lll.practice.target.lastAttempt', '2');
    localStorage.setItem('lll.draft.target.summary/notes.md', 'draft');
    localStorage.setItem('lll.draft.other.summary/notes.md', 'keep');

    clearDeletedProjectClientData('target');

    expect(localStorage.getItem('lll.draft.practice.target')).toBeNull();
    expect(localStorage.getItem('lll.practice.target.lastAttempt')).toBeNull();
    expect(localStorage.getItem('lll.draft.target.summary/notes.md')).toBeNull();
    expect(localStorage.getItem('lll.draft.other.summary/notes.md')).toBe('keep');
  });
});
