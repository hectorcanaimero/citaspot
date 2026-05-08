'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { Sidebar } from '@/components/dashboard/sidebar';
import { auth } from '@/lib/api';

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const [ready, setReady] = useState(false);

  useEffect(() => {
    auth.me().then(({ tenant }) => {
      if (!tenant.onboarding_done) {
        router.replace('/onboarding');
      } else {
        setReady(true);
      }
    }).catch(() => {
      setReady(true);
    });
  }, [router]);

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
    </div>
  );
}
