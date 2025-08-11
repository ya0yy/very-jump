import { create } from 'zustand';
import type { AppState, Server, Session } from '../types';

export const useAppStore = create<AppState>((set) => ({
  servers: [],
  sessions: [],
  activeSessions: 0,
  loading: false,
  error: null,

  setServers: (servers: Server[]) => set({ servers }),
  
  setSessions: (sessions: Session[]) => set({ sessions }),
  
  setActiveSessions: (activeSessions: number) => set({ activeSessions }),
  
  setLoading: (loading: boolean) => set({ loading }),
  
  setError: (error: string | null) => set({ error }),
}));



