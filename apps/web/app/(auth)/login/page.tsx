'use client';

import { Suspense, useState } from 'react';
import Link from 'next/link';
import { useRouter, useSearchParams } from 'next/navigation';
import { Button }  from '@/components/ui/button';
import { Input }   from '@/components/ui/input';
import { Card }    from '@/components/ui/card';
import { createClient } from '@/lib/supabase/browser';
import { useTranslations } from '@/lib/i18n';

function LoginForm() {
  const t      = useTranslations();
  const router = useRouter();
  const params = useSearchParams();
  const from   = params.get('from') ?? '/dashboard';

  const [email,    setEmail]    = useState('');
  const [password, setPassword] = useState('');
  const [error,    setError]    = useState('');
  const [loading,  setLoading]  = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError('');
    setLoading(true);
    try {
      const supabase = createClient();
      const { error: authError } = await supabase.auth.signInWithPassword({ email, password });
      if (authError) {
        setError(t.auth.invalidCredentials);
        return;
      }
      router.push(from);
      router.refresh();
    } catch {
      setError(t.auth.loginError);
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-neutral-50 px-4">
      <div className="w-full max-w-sm animate-slide-up">
        {/* Logo */}
        <div className="mb-8 text-center">
          <div className="mx-auto mb-3 flex h-12 w-12 items-center justify-center rounded-xl bg-primary-600 text-white text-xl font-bold shadow-md">
            A
          </div>
          <h1 className="text-2xl font-bold text-neutral-900">CitaSpot</h1>
          <p className="mt-1 text-sm text-neutral-500">{t.auth.loginSubtitle}</p>
        </div>

        <Card>
          <form onSubmit={handleSubmit} className="flex flex-col gap-4">
            <Input
              label={t.auth.emailLabel}
              type="email"
              autoComplete="email"
              placeholder={t.auth.emailPlaceholder}
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              required
            />
            <Input
              label={t.auth.passwordLabel}
              type="password"
              autoComplete="current-password"
              placeholder="••••••••"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
            />

            {error && (
              <p className="rounded-md bg-red-50 px-3 py-2 text-sm text-red-700">{error}</p>
            )}

            <Button type="submit" size="lg" loading={loading} className="w-full mt-1">
              {t.auth.loginButton}
            </Button>
          </form>

          <p className="mt-5 text-center text-sm text-neutral-500">
            {t.auth.noAccount}{' '}
            <Link href="/waitlist" className="font-medium text-primary-600 hover:text-primary-700">
              {t.waitlist.navCta}
            </Link>
          </p>
        </Card>
      </div>
    </div>
  );
}

export default function LoginPage() {
  return (
    <Suspense>
      <LoginForm />
    </Suspense>
  );
}
