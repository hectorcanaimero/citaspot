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
      <span className="font-medium text-neutral-900">{value ?? '\u2014'}</span>
    </div>
  );
}

export default function BusinessPage() {
  const t = useTranslations();
  const [tenant, setTenant] = useState<TenantDTO | null>(null);
  const [loading, setLoading] = useState(true);
  const [editing, setEditing] = useState(false);
  const [nameValue, setNameValue] = useState('');
  const [saving, setSaving] = useState(false);
  const [feedback, setFeedback] = useState<{ ok: boolean; msg: string } | null>(null);
  const businessTypes = t.settings.businessTypes as Record<string, string>;

  useEffect(() => {
    auth.me()
      .then(({ tenant: tn }) => {
        setTenant(tn);
        setNameValue(tn.name ?? '');
      })
      .catch(() => {})
      .finally(() => setLoading(false));
  }, []);

  function startEdit() {
    setNameValue(tenant?.name ?? '');
    setFeedback(null);
    setEditing(true);
  }

  function cancelEdit() {
    setEditing(false);
    setFeedback(null);
  }

  async function handleSave() {
    if (!nameValue.trim() || nameValue.trim().length < 2) return;
    setSaving(true);
    setFeedback(null);
    try {
      await auth.updateBusinessProfile({ name: nameValue.trim() });
      setTenant(prev => prev ? { ...prev, name: nameValue.trim() } : prev);
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
      <div className="divide-y divide-neutral-100">
        {/* Nombre del negocio — editable */}
        <div className="flex items-center justify-between py-2 text-sm">
          <span className="text-neutral-500">{t.settings.business.nameLabel}</span>
          {editing ? (
            <div className="flex items-center gap-2">
              <Input
                value={nameValue}
                onChange={e => setNameValue(e.target.value)}
                placeholder={t.settings.business.namePlaceholder}
                className="h-7 text-sm w-48"
                autoFocus
                onKeyDown={e => {
                  if (e.key === 'Enter') handleSave();
                  if (e.key === 'Escape') cancelEdit();
                }}
              />
              <Button
                size="sm"
                variant="ghost"
                className="h-7 w-7 p-0"
                onClick={handleSave}
                disabled={saving || nameValue.trim().length < 2}
                aria-label="Guardar"
              >
                {saving ? <Spinner size="sm" /> : <Check className="h-3.5 w-3.5 text-green-600" />}
              </Button>
              <Button
                size="sm"
                variant="ghost"
                className="h-7 w-7 p-0"
                onClick={cancelEdit}
                disabled={saving}
                aria-label="Cancelar"
              >
                <X className="h-3.5 w-3.5 text-neutral-400" />
              </Button>
            </div>
          ) : (
            <div className="flex items-center gap-2">
              <span className="font-medium text-neutral-900">{tenant?.name ?? '\u2014'}</span>
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
          )}
        </div>
        <InfoRow label={t.settings.business.typeLabel} value={businessTypes[tenant?.business_type ?? ''] ?? tenant?.business_type} />
        <InfoRow label={t.settings.business.bookingUrlLabel} value={`citaspot.com/book/${tenant?.slug ?? ''}`} />
      </div>

      {feedback && (
        <p className={`mt-2 text-xs ${feedback.ok ? 'text-green-600' : 'text-red-500'}`}>
          {feedback.msg}
        </p>
      )}

      {tenant?.slug && (
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
