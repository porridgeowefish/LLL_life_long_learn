// UI slice — transient UI state shared across pages (sidebar collapse,
// drawer open/close, modal open/close). Persisted to localStorage so a
// page refresh keeps the user's chosen layout.

import { create } from 'zustand';
import { persist } from 'zustand/middleware';

import { STORAGE_KEYS } from '@/lib/constants';

interface UiState {
  sidebarCollapsed: boolean;
  summaryPanelCollapsed: boolean;
  drawerOpen: boolean;
  createProjectModalOpen: boolean;
  setSidebarCollapsed: (v: boolean) => void;
  setSummaryPanelCollapsed: (v: boolean) => void;
  setDrawerOpen: (v: boolean) => void;
  toggleDrawer: () => void;
  openCreateProjectModal: () => void;
  closeCreateProjectModal: () => void;
}

export const useUiStore = create<UiState>()(
  persist(
    (set) => ({
      sidebarCollapsed: false,
      summaryPanelCollapsed: false,
      drawerOpen: false,
      createProjectModalOpen: false,
      setSidebarCollapsed: (v) => set({ sidebarCollapsed: v }),
      setSummaryPanelCollapsed: (v) => set({ summaryPanelCollapsed: v }),
      setDrawerOpen: (v) => set({ drawerOpen: v }),
      toggleDrawer: () => set((s) => ({ drawerOpen: !s.drawerOpen })),
      openCreateProjectModal: () => set({ createProjectModalOpen: true }),
      closeCreateProjectModal: () => set({ createProjectModalOpen: false }),
    }),
    {
      name: STORAGE_KEYS.uiSidebarCollapsed,
      // Only the sidebarCollapsed field is worth persisting across sessions;
      // modals and drawers should always start closed.
      partialize: (s) => ({
        sidebarCollapsed: s.sidebarCollapsed,
        summaryPanelCollapsed: s.summaryPanelCollapsed,
      }),
    },
  ),
);
