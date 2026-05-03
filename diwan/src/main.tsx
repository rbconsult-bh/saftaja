import './index.css';
import { StrictMode } from 'react'
import ReactDOM from 'react-dom/client'
import { RouterProvider, createRouter } from '@tanstack/react-router'
import { TransportProvider } from '@connectrpc/connect-query';
import { QueryClientProvider } from '@tanstack/react-query';

import { routeTree } from './routeTree.gen'
import { queryClient, transport } from './core/api/api-client';
import { m } from './paraglide/messages'

const router = createRouter({
  routeTree, defaultPendingComponent: () => (
    <div className="flex h-screen items-center justify-center">
      <p>{m.common_loading_saftaja()}</p>
    </div>
  ),
})

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router
  }
}

const rootElement = document.getElementById('root')!
if (!rootElement.innerHTML) {
  const root = ReactDOM.createRoot(rootElement)
  root.render(
    <StrictMode>
      <TransportProvider transport={transport}>
        <QueryClientProvider client={queryClient}>
          <RouterProvider router={router} />
        </QueryClientProvider>
      </TransportProvider>
    </StrictMode>,
  )
}
