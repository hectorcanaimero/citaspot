'use client';

// Paso 4 del onboarding: confirmación y CTA al dashboard.
import { useRouter } from 'next/navigation';
import { CheckCircle2, Calendar, Scissors, Clock, MessageCircle } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card }   from '@/components/ui/card';
import { useTranslations } from '@/lib/i18n';

export default function OnboardingDonePage() {
  const t      = useTranslations();
  const router = useRouter();

  const STEPS_DONE = [
    { icon: Scissors,      label: t.onboarding.servicesCreated },
    { icon: Clock,         label: t.onboarding.schedulesConfigured },
    { icon: MessageCircle, label: t.onboarding.whatsappConfigured },
  ];

  return (
    <div className="flex min-h-[calc(100vh-56px)] items-center justify-center px-4 py-10">
      <div className="w-full max-w-md animate-slide-up text-center">
        <CheckCircle2 className="mx-auto mb-4 h-16 w-16 text-emerald-500" />
        <h1 className="mb-2 text-2xl font-bold text-neutral-900">{t.onboarding.doneTitle}</h1>
        <p className="mb-8 text-sm text-neutral-500">{t.onboarding.doneDesc}</p>

        <Card className="mb-6 text-left">
          <p className="mb-3 text-sm font-medium text-neutral-700">{t.onboarding.whatYouConfigured}</p>
          <div className="flex flex-col gap-2">
            {STEPS_DONE.map(({ icon: Icon, label }) => (
              <div key={label} className="flex items-center gap-3 text-sm text-neutral-600">
                <div className="flex h-7 w-7 items-center justify-center rounded-full bg-emerald-50">
                  <Icon className="h-4 w-4 text-emerald-600" />
                </div>
                {label}
              </div>
            ))}
          </div>
        </Card>

        <Button size="lg" className="w-full" onClick={() => router.push('/dashboard')}>
          <Calendar className="mr-2 h-4 w-4" />
          {t.onboarding.goToDashboard}
        </Button>

        <p className="mt-4 text-xs text-neutral-400">
          {t.onboarding.publicUrlText}{' '}
          <span className="font-medium text-primary-600">citaspot.com/book/tu-negocio</span>
        </p>
      </div>
    </div>
  );
}
