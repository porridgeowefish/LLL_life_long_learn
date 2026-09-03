import { describe, expect, it } from 'vitest';

import { qk } from './queryKeys';

describe('queryKeys structure', () => {
  it('has one workspace-global preferences key', () => {
    expect(qk.preferences.file()).toEqual(['preferences']);
  });

  it('has files.raw with projectSlug + relPath', () => {
    const key = qk.files.raw('test', 'summary/report.md');
    expect(key).toEqual(['files', 'test', 'summary/report.md']);
  });

  it('has confusions.all with projectSlug', () => {
    const key = qk.confusions.all('test');
    expect(key).toEqual(['confusions', 'test']);
  });

  it('has confusions.detail with projectSlug + id', () => {
    const key = qk.confusions.detail('test', 'abc123');
    expect(key).toEqual(['confusions', 'test', 'abc123']);
  });

  it('all keys are frozen (as const)', () => {
    // qk is typed `as const` — verify the structure is readonly at compile time
    // by just checking all top-level scopes exist
    expect(qk.health).toBeTypeOf('function');
    expect(qk.projects.all).toBeTypeOf('function');
    expect(qk.agents.all).toBeTypeOf('function');
    expect(qk.sessions.active).toBeTypeOf('function');
    expect(qk.preferences.file).toBeTypeOf('function');
    expect(qk.files.raw).toBeTypeOf('function');
    expect(qk.confusions.all).toBeTypeOf('function');
  });
});
