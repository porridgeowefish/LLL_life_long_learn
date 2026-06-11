// Project slice — what the user has selected. Persisted so refresh keeps
// them on the same project / zone. The actual project data lives in
// TanStack Query cache (server state); this slice just tracks which one
// the user is currently looking at.

import { create } from 'zustand';
import { persist } from 'zustand/middleware';

import { STORAGE_KEYS } from '@/lib/constants';
import type { ZoneName } from '@/types/domain';

interface ProjectState {
  selectedProjectId: string | null;
  currentZone: ZoneName | null;
  expandedPaths: string[]; // for FileTree expansion state
  selectProject: (id: string | null) => void;
  setZone: (z: ZoneName | null) => void;
  togglePath: (path: string) => void;
}

export const useProjectStore = create<ProjectState>()(
  persist(
    (set) => ({
      selectedProjectId: null,
      currentZone: null,
      expandedPaths: [],
      selectProject: (id) => set({ selectedProjectId: id }),
      setZone: (z) => set({ currentZone: z }),
      togglePath: (path) =>
        set((s) => ({
          expandedPaths: s.expandedPaths.includes(path)
            ? s.expandedPaths.filter((p) => p !== path)
            : [...s.expandedPaths, path],
        })),
    }),
    {
      name: 'lll.project',
      partialize: (s) => ({
        selectedProjectId: s.selectedProjectId,
        currentZone: s.currentZone,
      }),
    },
  ),
);

// Convenience: bind storage keys to the documented constants for grep-ability.
void STORAGE_KEYS.projectSelectedId;
void STORAGE_KEYS.projectCurrentZone;
