import { createConnectTransport } from "@connectrpc/connect-web"
import { QueryClient } from "@tanstack/react-query";
import { authInterceptor } from "./auth-interceptor";

export const transport = createConnectTransport({
  baseUrl: "http://localhost:8080",
  interceptors: [authInterceptor],
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
