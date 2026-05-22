'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { Toaster } from 'sonner';
import { Sidebar } from '@/components/dashboard/sidebar';
import { auth, APIError } from '@/lib/api';
import { useTranslations, useLanguage } from '@/lib/i18n';
import { useTenantStore } from '@/store/tenant';

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const t = useTranslations();
  const { setLanguageFromCountry } = useLanguage();
  const setTenant = useTenantStore((s) => s.setTenant);
  const [ready, setReady] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    auth.me().then(({ tenant, enabled_modules }) => {
      if (cancelled) return;
      setTenant({ ...tenant, enabled_modules: enabled_modules ?? [] });
      setLanguageFromCountry(tenant.country);
      if (!tenant.onboarding_done) {
        router.replace('/onboarding');
      } else {
        setReady(true);
      }
    }).catch((err: unknown) => {
      if (cancelled) return;
      // 401/403: sesion invalida o expirada → login. Nunca renderizar dashboard sin tenant.
      if (err instanceof APIError && (err.status === 401 || err.status === 403)) {
        router.replace('/login');
        return;
      }
      // Red / 5xx → UI de error con reintento, NO render del dashboard.
      setError(t.common.errorLoadingSession);
    });
    return () => { cancelled = true; };
  }, [router, t, setTenant, setLanguageFromCountry]);

  if (error) {
    return (
      <div className="flex h-screen flex-col items-center justify-center gap-4 bg-neutral-50 px-6 text-center">
        <p className="text-neutral-700">{error}</p>
        <button
          onClick={() => { setError(null); router.refresh(); }}
          className="rounded-md bg-primary-500 px-4 py-2 text-white hover:bg-primary-600"
        >
          {t.common.tryAgain}
        </button>
      </div>
    );
  }

  if (!ready) {
    return (
      <div className="flex h-screen items-center justify-center bg-neutral-50">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-primary-500 border-t-transparent" />
      </div>
    );
  }

  return (
    <div className="flex h-screen overflow-hidden bg-white">
      <Sidebar />
      <main className="flex-1 overflow-y-auto bg-neutral-50">
        <div className="md:hidden h-12" />
        {children}
      </main>
      <Toaster position="bottom-right" richColors />
    </div>
  );
}
