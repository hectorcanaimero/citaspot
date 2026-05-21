'use client';

import { create } from 'zustand';
import { persist } from 'zustand/middleware';

interface NotificationState {
  soundEnabled: boolean;
  setSoundEnabled: (value: boolean) => void;
  toggleSound: () => void;
}

export const useNotifications = create<NotificationState>()(
  persist(
    (set) => ({
      soundEnabled: true,
      setSoundEnabled: (value) => set({ soundEnabled: value }),
      toggleSound: () => set((s) => ({ soundEnabled: !s.soundEnabled })),
    }),
    {
      name: 'citaspot:sound_enabled',
      partialize: (s) => ({ soundEnabled: s.soundEnabled }),
    },
  ),
);
