import { create } from "zustand";
import { TokenVault } from "./token-vault";
import { queryClient } from "../api/api-client";

interface SessionState {
  isAuthenticated: boolean;
  actions: {
    login: (accessToken: string, refreshToken: string) => void;
    logout: () => void;
  };
}

export const useSessionStore = create<SessionState>((set) => ({
  isAuthenticated: !!TokenVault.getRefresh(),

  actions: {
    login(accessToken, refreshToken) {
      TokenVault.setTokens(accessToken, refreshToken);
      set({ isAuthenticated: true });
    },
    logout() {
      TokenVault.clearTokens();
      set({ isAuthenticated: false });

      queryClient.clear();
    },
  },
}));
