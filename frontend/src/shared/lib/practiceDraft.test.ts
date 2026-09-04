import { describe, expect, it } from 'vitest';

import { isEmptyPracticeAnswer } from './practiceDraft';

describe('isEmptyPracticeAnswer', () => {
  it('handles text, boolean, and multiple-choice answers', () => {
    expect(isEmptyPracticeAnswer('')).toBe(true);
    expect(isEmptyPracticeAnswer([])).toBe(true);
    expect(isEmptyPracticeAnswer(false)).toBe(false);
    expect(isEmptyPracticeAnswer(['A'])).toBe(false);
    expect(isEmptyPracticeAnswer('answer')).toBe(false);
  });
});
