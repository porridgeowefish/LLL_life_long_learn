import { describe, expect, it, beforeEach } from 'vitest';

import { useProjectStore } from './project';

describe('useProjectStore', () => {
  beforeEach(() => {
    useProjectStore.setState({
      selectedProjectId: null,
      currentZone: null,
      expandedPaths: [],
    });
  });

  it('selects a project and zone', () => {
    useProjectStore.getState().selectProject('rust-ownership');
    useProjectStore.getState().setZone('Explain');
    const s = useProjectStore.getState();
    expect(s.selectedProjectId).toBe('rust-ownership');
    expect(s.currentZone).toBe('Explain');
  });

  it('toggles expandedPaths additively then removes on next toggle', () => {
    useProjectStore.getState().togglePath('intro/');
    useProjectStore.getState().togglePath('explain/');
    expect(useProjectStore.getState().expandedPaths).toEqual(['intro/', 'explain/']);
    useProjectStore.getState().togglePath('intro/');
    expect(useProjectStore.getState().expandedPaths).toEqual(['explain/']);
  });
});
