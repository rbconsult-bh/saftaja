import { createConnectTransport } from "@connectrpc/connect-web"
import { QueryClient } from "@tanstack/react-query";

export const transport = createConnectTransport({
  baseUrl: "https://khazina.safaja.com",
  interceptors: [], // TODO: add auth interceptor once implemented
});

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      refetchOnWindowFocus: false,
      retry: 1,
      staleTime: 1000 * 5 * 60
    },
  },
});
