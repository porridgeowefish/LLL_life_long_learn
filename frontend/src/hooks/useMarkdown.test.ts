import { describe, expect, it } from 'vitest';

// Test the useMarkdown hook's rendering pipeline by importing the module
// and verifying the exported types/functions exist.
describe('useMarkdown module', () => {
  it('exports useMarkdown as a function', async () => {
    const mod = await import('./useMarkdown');
    expect(typeof mod.useMarkdown).toBe('function');
  });
});
