
const ACCESS_TOKEN_KEY = "saftaja_auth_access_token";
const REFRESH_TOKEN_KEY = "saftaja_auth_refresh_token";

export const TokenVault = {
  getAccess: (): string | null => {
    return localStorage.getItem(ACCESS_TOKEN_KEY);
  },
  getRefresh: (): string | null => {
    return localStorage.getItem(REFRESH_TOKEN_KEY);
  },

  setTokens: (access: string, refresh: string): void => {
    localStorage.setItem(ACCESS_TOKEN_KEY, access);
    localStorage.setItem(REFRESH_TOKEN_KEY, refresh);
  },

  clearTokens: (): void => {
    localStorage.removeItem(ACCESS_TOKEN_KEY);
    localStorage.removeItem(REFRESH_TOKEN_KEY);
  },
};
