import { create } from 'zustand'
import { persist } from 'zustand/middleware'

interface WorkspaceState {
  lastActiveProjectId: string | null
  actions: {
    setLastActiveProject: (id: string) => void
    clear: () => void
  }
}

export const useWorkspaceStore = create<WorkspaceState>()(
  persist(
    (set) => ({
      lastActiveProjectId: null,
      actions: {
        setLastActiveProject: (id) => set({ lastActiveProjectId: id }),
        clear: () => set({ lastActiveProjectId: null }),
      },
    }),
    {
      name: 'saftaja-workspace-context',
      partialize: (state) => ({ lastActiveProjectId: state.lastActiveProjectId }),
      merge: (persisted, current) => {
        const state = persisted as Partial<WorkspaceState> | undefined
        return {
          ...current,
          lastActiveProjectId: state?.lastActiveProjectId ?? current.lastActiveProjectId,
        }
      },
    }
  )
)
