'use client';

import { useState } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { Button } from '@/components/ui/button';
import { Input }  from '@/components/ui/input';
import { Card }   from '@/components/ui/card';
import { auth, APIError } from '@/lib/api';
import { useTranslations } from '@/lib/i18n';

type Step = 1 | 2 | 3;

interface FormData {
  name: string;
  email: string;
  password: string;
  business_name: string;
  business_type: string;
  city: string;
  country: string;
  timezone: string;
}

export default function RegisterPage() {
  const t      = useTranslations();
  const router = useRouter();
  const [step, setStep]     = useState<Step>(1);
  const [error, setError]   = useState('');
  const [loading, setLoading] = useState(false);

  const businessTypes = t.auth.businessTypes as Record<string, string>;
  const countries     = t.auth.countries as Record<string, string>;

  const BUSINESS_TYPE_OPTS = Object.entries(businessTypes).map(([value, label]) => ({ value, label }));
  const COUNTRY_OPTS       = Object.entries(countries).map(([value, label]) => ({ value, label }));
  const STEP_LABELS: Record<Step, string> = { 1: t.auth.step1, 2: t.auth.step2, 3: t.auth.step3 };

  const [form, setForm] = useState<FormData>({
    name: '', email: '', password: '',
    business_name: '', business_type: 'beauty',
    city: '', country: 'DO', timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
  });

  function update(field: keyof FormData, value: string) {
    setForm((f) => ({ ...f, [field]: value }));
  }

  function nextStep() {
    setError('');
    if (step < 3) setStep((s) => (s + 1) as Step);
  }

  function prevStep() {
    setError('');
    if (step > 1) setStep((s) => (s - 1) as Step);
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (step < 3) { nextStep(); return; }

    setError('');
    setLoading(true);
    try {
      await auth.register(form);
      router.push('/onboarding');
      router.refresh();
    } catch (err) {
      setError(err instanceof APIError ? err.message : t.auth.createAccountError);
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-neutral-50 px-4 py-10">
      <div className="w-full max-w-md animate-slide-up">
        {/* Logo */}
        <div className="mb-6 text-center">
          <div className="mx-auto mb-2 flex h-12 w-12 items-center justify-center rounded-xl bg-primary-600 text-white text-xl font-bold shadow-md">
            A
          </div>
          <h1 className="text-2xl font-bold text-neutral-900">{t.auth.registerTitle}</h1>
          <p className="mt-1 text-sm text-neutral-500">{t.auth.registerSubtitle}</p>
        </div>

        {/* Progress */}
        <div className="mb-6 flex gap-2">
          {([1, 2, 3] as Step[]).map((s) => (
            <div key={s} className="flex flex-1 flex-col items-center gap-1">
              <div
                className={`h-1.5 w-full rounded-full transition-colors duration-300 ${
                  s <= step ? 'bg-primary-600' : 'bg-neutral-200'
                }`}
              />
              <span className={`text-xs ${s <= step ? 'text-primary-600 font-medium' : 'text-neutral-400'}`}>
                {STEP_LABELS[s]}
              </span>
            </div>
          ))}
        </div>

        <Card>
          <form onSubmit={handleSubmit} className="flex flex-col gap-4">
            {/* Paso 1 — Cuenta */}
            {step === 1 && (
              <>
                <Input label={t.auth.nameLabel} placeholder={t.auth.namePlaceholder} value={form.name}
                  onChange={(e) => update('name', e.target.value)} required />
                <Input label={t.auth.emailLabel} type="email" placeholder={t.auth.emailPlaceholder}
                  value={form.email} onChange={(e) => update('email', e.target.value)} required />
                <Input label={t.auth.passwordLabel} type="password" placeholder={t.auth.passwordHint}
                  value={form.password} onChange={(e) => update('password', e.target.value)}
                  hint={t.auth.passwordHint} required minLength={8} />
              </>
            )}

            {/* Paso 2 — Negocio */}
            {step === 2 && (
              <>
                <Input label={t.auth.businessNameLabel} placeholder={t.auth.businessNamePlaceholder} value={form.business_name}
                  onChange={(e) => update('business_name', e.target.value)} required />
                <div className="flex flex-col gap-1.5">
                  <label className="text-sm font-medium text-neutral-700">{t.auth.businessTypeLabel}</label>
                  <div className="grid grid-cols-2 gap-2">
                    {BUSINESS_TYPE_OPTS.map((bt) => (
                      <button
                        key={bt.value}
                        type="button"
                        onClick={() => update('business_type', bt.value)}
                        className={`rounded-lg border px-3 py-2.5 text-sm font-medium transition-colors ${
                          form.business_type === bt.value
                            ? 'border-primary-500 bg-primary-50 text-primary-700'
                            : 'border-neutral-200 bg-white text-neutral-600 hover:bg-neutral-50'
                        }`}
                      >
                        {bt.label}
                      </button>
                    ))}
                  </div>
                </div>
              </>
            )}

            {/* Paso 3 — Ubicación */}
            {step === 3 && (
              <>
                <div className="flex flex-col gap-1.5">
                  <label className="text-sm font-medium text-neutral-700">{t.auth.countryLabel}</label>
                  <select
                    value={form.country}
                    onChange={(e) => update('country', e.target.value)}
                    className="h-9 w-full rounded-md border border-neutral-200 bg-white px-3 text-sm text-neutral-900 focus:outline-none focus:ring-2 focus:ring-primary-500"
                  >
                    {COUNTRY_OPTS.map((c) => (
                      <option key={c.value} value={c.value}>{c.label}</option>
                    ))}
                  </select>
                </div>
                <Input label={t.auth.cityLabel} placeholder={t.auth.cityPlaceholder} value={form.city}
                  onChange={(e) => update('city', e.target.value)} />
              </>
            )}

            {error && (
              <p className="rounded-md bg-red-50 px-3 py-2 text-sm text-red-700">{error}</p>
            )}

            <div className="flex gap-3 mt-1">
              {step > 1 && (
                <Button type="button" variant="secondary" size="lg" onClick={prevStep} className="flex-1">
                  {t.auth.backButton}
                </Button>
              )}
              <Button type="submit" size="lg" loading={loading} className="flex-1">
                {step < 3 ? t.auth.continueButton : t.auth.createAccountButton}
              </Button>
            </div>
          </form>

          <p className="mt-5 text-center text-sm text-neutral-500">
            {t.auth.alreadyHaveAccount}{' '}
            <Link href="/login" className="font-medium text-primary-600 hover:text-primary-700">
              {t.auth.signInLink}
            </Link>
          </p>
        </Card>
      </div>
    </div>
  );
}
