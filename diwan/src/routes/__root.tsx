import { createRootRoute, Outlet } from '@tanstack/react-router'
import { LanguageProvider } from "@/components/LanguageProvider"
import { TooltipProvider } from "@/components/ui/tooltip"

export const Route = createRootRoute({
  component: () => (
    <LanguageProvider>
      <TooltipProvider>
        <div className="min-h-screen font-sans antialiased">
          <Outlet />
        </div>
      </TooltipProvider>
    </LanguageProvider>
  ),
})
