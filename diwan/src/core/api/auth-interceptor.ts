import { Code, ConnectError, type Interceptor } from "@connectrpc/connect";
import { TokenVault } from "../session/token-vault";
import { useSessionStore } from "../session/store";
import { refreshClient } from "./api-client";

let refreshPromise: Promise<string> | null = null;

export const authInterceptor: Interceptor = (next) => async (req) => {
  const accessToken = TokenVault.getAccess();
  if (accessToken) {
    req.header.set("authorization", `Bearer ${accessToken}`);
  }

  try {
    return await next(req);
  } catch (err) {
    const isUnauthenticated = err instanceof ConnectError && err.code === Code.Unauthenticated;
    if (!isUnauthenticated) {
      throw err;
    }

    const refreshToken = TokenVault.getRefresh();
    if (!refreshToken) {
      useSessionStore.getState().actions.logout();
      throw err;
    }

    if (!refreshPromise) {
      refreshPromise = refreshClient
        .refreshToken({ refreshToken })
        .then((resp) => {
          useSessionStore.getState().actions.login(resp.accessToken, resp.refreshToken);
          return resp.accessToken;
        })
        .catch((refreshErr) => {
          useSessionStore.getState().actions.logout();
          throw refreshErr
        })
        .finally(() => {
          refreshPromise = null;
        });
    }

    try {
      const newAccessToken = await refreshPromise;

      req.header.set("authorization", `Bearer ${newAccessToken}`);
      return next(req);
    } catch (queueErr) {
      throw err
    }
  }
}
