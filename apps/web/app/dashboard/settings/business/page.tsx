'use client';

import { useState, useEffect } from 'react';
import { Building2, ExternalLink, Pencil, X, Check } from 'lucide-react';
import { Card, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Spinner } from '@/components/ui/spinner';
import { auth, TenantDTO } from '@/lib/api';
import { useTranslations } from '@/lib/i18n';

function InfoRow({ label, value }: { label: string; value: string | undefined }) {
  return (
    <div className="flex items-center justify-between py-2 text-sm">
      <span className="text-neutral-500">{label}</span>
      <span className="font-medium text-neutral-900">{value ?? '—'}</span>
    </div>
  );
}

export default function BusinessPage() {
  const t = useTranslations();
  const [tenant, setTenant] = useState<TenantDTO | null>(null);
  const [loading, setLoading] = useState(true);
  const [editing, setEditing] = useState(false);
  const [nameValue, setNameValue] = useState('');
  const [countryValue, setCountryValue] = useState('DO');
  const [cityValue, setCityValue] = useState('');
  const [saving, setSaving] = useState(false);
  const [feedback, setFeedback] = useState<{ ok: boolean; msg: string } | null>(null);
  const businessTypes = t.settings.businessTypes as Record<string, string>;
  const countries = t.auth.countries as Record<string, string>;
  const COUNTRY_OPTS = Object.entries(countries).map(([value, label]) => ({ value, label }));

  useEffect(() => {
    auth.me()
      .then(({ tenant: tn }) => {
        setTenant(tn);
        setNameValue(tn.name ?? '');
        setCountryValue(tn.country ?? 'DO');
        setCityValue(tn.city ?? '');
      })
      .catch(() => {})
      .finally(() => setLoading(false));
  }, []);

  function startEdit() {
    setNameValue(tenant?.name ?? '');
    setCountryValue(tenant?.country ?? 'DO');
    setCityValue(tenant?.city ?? '');
    setFeedback(null);
    setEditing(true);
  }

  function cancelEdit() {
    setEditing(false);
    setFeedback(null);
  }

  async function handleSave() {
    const trimmedName = nameValue.trim();
    const trimmedCity = cityValue.trim();
    if (!trimmedName || trimmedName.length < 2) return;
    setSaving(true);
    setFeedback(null);
    try {
      await auth.updateBusinessProfile({
        name: trimmedName,
        city: trimmedCity,
        country: countryValue,
      });
      setTenant(prev => prev ? { ...prev, name: trimmedName, city: trimmedCity, country: countryValue } : prev);
      setEditing(false);
      setFeedback({ ok: true, msg: t.settings.business.saveSuccess });
    } catch {
      setFeedback({ ok: false, msg: t.settings.business.saveError });
    } finally {
      setSaving(false);
    }
  }

  if (loading) return <div className="flex justify-center py-16"><Spinner size="lg" /></div>;

  return (
    <Card>
      <CardHeader>
        <Building2 className="h-4 w-4 text-neutral-400" />
        <CardTitle>{t.settings.business.title}</CardTitle>
      </CardHeader>

      {editing ? (
        <div className="flex flex-col gap-3 py-2">
          <Input
            label={t.settings.business.nameLabel}
            value={nameValue}
            onChange={e => setNameValue(e.target.value)}
            placeholder={t.settings.business.namePlaceholder}
            autoFocus
          />
          <div className="flex flex-col gap-1.5">
            <label className="text-sm font-medium text-neutral-700">{t.auth.countryLabel}</label>
            <select
              value={countryValue}
              onChange={e => setCountryValue(e.target.value)}
              className="h-9 w-full rounded-md border border-neutral-200 bg-white px-3 text-sm text-neutral-900 focus:outline-none focus:ring-2 focus:ring-primary-500"
            >
              {COUNTRY_OPTS.map(c => (
                <option key={c.value} value={c.value}>{c.label}</option>
              ))}
            </select>
          </div>
          <Input
            label={t.auth.cityLabel}
            value={cityValue}
            onChange={e => setCityValue(e.target.value)}
            placeholder={t.auth.cityPlaceholder}
          />
          <div className="flex items-center gap-2 pt-1">
            <Button
              size="sm"
              onClick={handleSave}
              disabled={saving || nameValue.trim().length < 2}
            >
              {saving ? <Spinner size="sm" /> : <><Check className="mr-1.5 h-3.5 w-3.5" />{t.common.save}</>}
            </Button>
            <Button
              size="sm"
              variant="secondary"
              onClick={cancelEdit}
              disabled={saving}
            >
              <X className="mr-1.5 h-3.5 w-3.5" />
              {t.common.cancel}
            </Button>
          </div>
        </div>
      ) : (
        <div className="divide-y divide-neutral-100">
          <div className="flex items-center justify-between py-2 text-sm">
            <span className="text-neutral-500">{t.settings.business.nameLabel}</span>
            <div className="flex items-center gap-2">
              <span className="font-medium text-neutral-900">{tenant?.name ?? '—'}</span>
              <Button
                size="sm"
                variant="ghost"
                className="h-6 w-6 p-0"
                onClick={startEdit}
                aria-label={t.settings.business.editName}
              >
                <Pencil className="h-3 w-3 text-neutral-400" />
              </Button>
            </div>
          </div>
          <InfoRow label={t.settings.business.typeLabel} value={businessTypes[tenant?.business_type ?? ''] ?? tenant?.business_type} />
          <InfoRow label={t.auth.countryLabel} value={countries[tenant?.country ?? ''] ?? tenant?.country} />
          <InfoRow label={t.auth.cityLabel} value={tenant?.city || undefined} />
          <InfoRow label={t.settings.business.bookingUrlLabel} value={`citaspot.com/book/${tenant?.slug ?? ''}`} />
        </div>
      )}

      {feedback && (
        <p className={`mt-2 text-xs ${feedback.ok ? 'text-green-600' : 'text-red-500'}`}>
          {feedback.msg}
        </p>
      )}

      {tenant?.slug && !editing && (
        <div className="mt-3 pt-3 border-t border-neutral-100">
          <Button
            variant="ghost"
            size="sm"
            onClick={() => window.open(`/book/${tenant.slug}`, '_blank')}
          >
            <ExternalLink className="mr-1.5 h-3.5 w-3.5" />
            {t.settings.business.viewBookingPage}
          </Button>
        </div>
      )}
    </Card>
  );
}
