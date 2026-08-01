import { createRootRoute, Outlet } from '@tanstack/react-router'
import { LanguageProvider } from "@/components/LanguageProvider"
import { TooltipProvider } from "@/components/ui/tooltip"
import { ReactQueryDevtools } from '@tanstack/react-query-devtools'
import { TanStackRouterDevtools } from '@tanstack/react-router-devtools'


export const Route = createRootRoute({
  component: () => (
    <LanguageProvider>
      <TooltipProvider>
        <div className="min-h-screen font-sans antialiased">
          <Outlet />

          <TanStackRouterDevtools />
          <ReactQueryDevtools />
        </div>
      </TooltipProvider>
    </LanguageProvider>
  ),
})
