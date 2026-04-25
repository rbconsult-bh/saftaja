import { createConnectTransport, type ConnectTransportOptions } from "@connectrpc/connect-web"
import { QueryClient } from "@tanstack/react-query";
import { authInterceptor } from "./auth-interceptor";
import { createClient } from "@connectrpc/connect";
import { AuthService } from "../../gen/saftaja/dashboard/auth/v1/auth_pb";

const baseTransportOpts: ConnectTransportOptions = {
  baseUrl: "http://localhost:8080",
};

const nakedTransport = createConnectTransport(baseTransportOpts)
export const refreshClient = createClient(AuthService, nakedTransport);


export const transport = createConnectTransport({
  ...baseTransportOpts,
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
