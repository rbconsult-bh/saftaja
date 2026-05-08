import { createContext, useContext } from 'react'
import type { GetWorkspaceResponse } from '@/gen/saftaja/dashboard/workspace/v1/workspace_pb'

type WorkspaceContextValue = {
  workspace: GetWorkspaceResponse
}

const WorkspaceContext = createContext<WorkspaceContextValue | null>(null)

export function WorkspaceProvider({ workspace, children }: { workspace: GetWorkspaceResponse; children: React.ReactNode }) {
  return (
    <WorkspaceContext.Provider value={{ workspace }}>
      {children}
    </WorkspaceContext.Provider>
  )
}

export function useWorkspace(): WorkspaceContextValue {
  const ctx = useContext(WorkspaceContext)
  if (!ctx) {
    throw new Error('useWorkspace must be used within a WorkspaceProvider')
  }
  return ctx
}
