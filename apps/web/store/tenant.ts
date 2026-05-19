'use client';

import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import type { TenantDTO } from '@/lib/api';

interface TenantState {
  tenant: TenantDTO | null;
  setTenant: (t: TenantDTO | null) => void;
}

export const useTenantStore = create<TenantState>()(
  persist(
    (set) => ({ tenant: null, setTenant: (tenant) => set({ tenant }) }),
    { name: 'citaspot_tenant', partialize: (s) => ({ tenant: s.tenant }) },
  ),
);

export function useTenantTimezone(): string {
  return useTenantStore((s) => s.tenant?.timezone || 'UTC');
}
