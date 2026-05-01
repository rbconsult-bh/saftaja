import { createRootRoute, Outlet } from '@tanstack/react-router'
import { LanguageProvider } from "@/components/LanguageProvider"

export const Route = createRootRoute({
  component: () => (
    <LanguageProvider>
      <div className="min-h-screen font-sans antialiased">
        <Outlet />
      </div>
    </LanguageProvider>
  ),
})
