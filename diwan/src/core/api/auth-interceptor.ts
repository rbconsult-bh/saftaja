import type { Interceptor } from "@connectrpc/connect";

export const authInterceptor: Interceptor = (next) => async (req) => {
  // TODO: use token manager to get token and inject it into header :D
  return await next(req);
}
