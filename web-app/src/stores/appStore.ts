import { create } from 'zustand';
import type { AppState, Server, Session } from '../types';

export const useAppStore = create<AppState>((set, get) => ({
  servers: [],
  sessions: [],
  activeSessions: 0,
  loading: false,
  error: null,

  setServers: (servers: Server[]) => set({ servers }),

  updateServerStatus: (serverId: number, status: string) => {
    const currentServers = get().servers;
    const updatedServers = currentServers.map(server =>
      server.id === serverId
        ? { ...server, status: status as 'available' | 'unavailable' | 'checking' }
        : server
    );
    set({ servers: updatedServers });
  },

  setSessions: (sessions: Session[]) => set({ sessions }),

  setActiveSessions: (activeSessions: number) => set({ activeSessions }),

  setLoading: (loading: boolean) => set({ loading }),

  setError: (error: string | null) => set({ error }),
}));



