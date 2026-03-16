'use client';

// Paso 1 del onboarding: crear al menos un servicio.
import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { Plus, Trash2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input }  from '@/components/ui/input';
import { Card }   from '@/components/ui/card';
import { services as servicesApi, APIError } from '@/lib/api';
import { useTranslations } from '@/lib/i18n';

interface ServiceForm {
  name: string;
  duration_min: string;
  price: string;
}

const EMPTY: ServiceForm = { name: '', duration_min: '60', price: '' };

export default function OnboardingServicesPage() {
  const t      = useTranslations();
  const router = useRouter();
  const [forms, setForms]   = useState<ServiceForm[]>([{ ...EMPTY }]);
  const [saving, setSaving] = useState(false);
  const [error, setError]   = useState('');

  function addService() {
    setForms((f) => [...f, { ...EMPTY }]);
  }

  function removeService(idx: number) {
    setForms((f) => f.filter((_, i) => i !== idx));
  }

  function update(idx: number, field: keyof ServiceForm, value: string) {
    setForms((f) => f.map((s, i) => (i === idx ? { ...s, [field]: value } : s)));
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    const valid = forms.filter((s) => s.name.trim() && s.duration_min);
    if (valid.length === 0) {
      setError(t.onboarding.addAtLeastOne);
      return;
    }
    setSaving(true);
    setError('');
    try {
      await Promise.all(
        valid.map((s) =>
          servicesApi.create({
            name:         s.name.trim(),
            duration_min: parseInt(s.duration_min, 10),
            price:        s.price ? parseFloat(s.price) : undefined,
            currency:     'USD',
            is_active:    true,
          })
        )
      );
      router.push('/onboarding/schedule');
    } catch (err) {
      setError(err instanceof APIError ? err.message : t.onboarding.saveError);
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="mx-auto max-w-lg px-4 py-10">
      {/* Progreso */}
      <div className="mb-2 flex gap-2">
        {[1, 2, 3, 4].map((s) => (
          <div
            key={s}
            className={`h-1.5 flex-1 rounded-full transition-colors duration-300 ${
              s === 1 ? 'bg-primary-600' : 'bg-neutral-200'
            }`}
          />
        ))}
      </div>
      <p className="mb-6 text-xs text-neutral-400">
        {t.onboarding.stepOf
          .replace('{step}', '1')
          .replace('{total}', '4')
          .replace('{label}', t.onboarding.servicesLabel)}
      </p>

      <h1 className="mb-1 text-xl font-bold text-neutral-900">{t.onboarding.servicesTitle}</h1>
      <p className="mb-6 text-sm text-neutral-500">{t.onboarding.servicesDesc}</p>

      <form onSubmit={handleSubmit} className="flex flex-col gap-4">
        {forms.map((s, idx) => (
          <Card key={idx} className="relative">
            <div className="flex flex-col gap-3">
              <div className="flex items-center justify-between">
                <p className="text-sm font-medium text-neutral-700">
                  {t.onboarding.serviceNumber.replace('{n}', String(idx + 1))}
                </p>
                {forms.length > 1 && (
                  <button
                    type="button"
                    onClick={() => removeService(idx)}
                    className="text-neutral-400 hover:text-red-500 transition-colors"
                  >
                    <Trash2 className="h-4 w-4" />
                  </button>
                )}
              </div>
              <Input
                label={t.onboarding.nameLabel}
                placeholder={t.onboarding.namePlaceholder}
                value={s.name}
                onChange={(e) => update(idx, 'name', e.target.value)}
                required
              />
              <div className="grid grid-cols-2 gap-3">
                <Input
                  label={t.onboarding.durationLabel}
                  type="number"
                  min="5"
                  max="480"
                  value={s.duration_min}
                  onChange={(e) => update(idx, 'duration_min', e.target.value)}
                  required
                />
                <Input
                  label={t.onboarding.priceLabel}
                  type="number"
                  min="0"
                  step="0.01"
                  placeholder="25.00"
                  value={s.price}
                  onChange={(e) => update(idx, 'price', e.target.value)}
                />
              </div>
            </div>
          </Card>
        ))}

        <button
          type="button"
          onClick={addService}
          className="flex items-center gap-2 rounded-lg border border-dashed border-neutral-300 px-4 py-3 text-sm text-neutral-500 hover:border-primary-400 hover:text-primary-600 transition-colors"
        >
          <Plus className="h-4 w-4" />
          {t.onboarding.addAnotherService}
        </button>

        {error && (
          <p className="rounded-md bg-red-50 px-3 py-2 text-sm text-red-700">{error}</p>
        )}

        <Button type="submit" size="lg" loading={saving}>
          {t.onboarding.continueButton}
        </Button>
      </form>
    </div>
  );
}
