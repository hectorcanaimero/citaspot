'use client';

import { useState, useEffect, useCallback } from 'react';
import {
  Building2, User, CreditCard, ExternalLink,
  Download, AlertTriangle, CheckCircle2, Clock, XCircle, Globe,
} from 'lucide-react';
import { Card, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Spinner } from '@/components/ui/spinner';
import {
  auth, billing, settingsApi,
  UserDTO, TenantDTO, TenantSettings, StripeSubscription, StripeInvoice,
} from '@/lib/api';
import { useTranslations, useLanguage } from '@/lib/i18n';

// ── Componentes auxiliares ──────────────────────────────────────────────────

function InfoRow({ label, value }: { label: string; value: string | undefined }) {
  return (
    <div className="flex items-center justify-between py-2 text-sm">
      <span className="text-neutral-500">{label}</span>
      <span className="font-medium text-neutral-900">{value ?? '—'}</span>
    </div>
  );
}

type Tab = 'negocio' | 'cuenta' | 'suscripcion';

function TabButton({
  active, onClick, children,
}: { active: boolean; onClick: () => void; children: React.ReactNode }) {
  return (
    <button
      onClick={onClick}
      className={`flex-1 px-4 py-2 text-sm font-medium text-center rounded-lg transition-colors ${
        active
          ? 'bg-white text-neutral-900 shadow-sm'
          : 'text-neutral-500 hover:text-neutral-700'
      }`}
    >
      {children}
    </button>
  );
}

function PlanStatusBadge({ plan, planStatus }: { plan: string; planStatus: string }) {
  const t = useTranslations();
  const isActive    = planStatus === 'active';
  const isTrial     = planStatus === 'trial' || planStatus === 'trialing' || plan === 'trial';
  const isPastDue   = planStatus === 'past_due';
  const isCancelled = planStatus === 'cancelled';

  if (isActive)    return <Badge variant="success">{t.settings.planStatus.active}</Badge>;
  if (isTrial)     return <Badge variant="warning">{t.settings.planStatus.trial}</Badge>;
  if (isPastDue)   return <Badge variant="error">{t.settings.planStatus.pastDue}</Badge>;
  if (isCancelled) return <Badge variant="default">{t.settings.planStatus.cancelled}</Badge>;
  return <Badge variant="default">{planStatus}</Badge>;
}

function InvoiceStatusBadge({ status }: { status: string }) {
  const t = useTranslations();
  if (status === 'paid') return <Badge variant="success">{t.settings.invoiceStatus.paid}</Badge>;
  if (status === 'open') return <Badge variant="warning">{t.settings.invoiceStatus.open}</Badge>;
  if (status === 'void') return <Badge variant="default">{t.settings.invoiceStatus.void}</Badge>;
  return <Badge variant="default">{status}</Badge>;
}

// ── Tab Negocio ─────────────────────────────────────────────────────────────

function NegocioTab({ tenant }: { tenant: TenantDTO | null }) {
  const t = useTranslations();
  const businessTypes = t.settings.businessTypes as Record<string, string>;

  const [settings, setSettings] = useState<TenantSettings | null>(null);
  const [settingsLoading, setSettingsLoading] = useState(true);
  const [settingsSaving, setSettingsSaving] = useState(false);

  useEffect(() => {
    settingsApi.get()
      .then(setSettings)
      .catch(console.error)
      .finally(() => setSettingsLoading(false));
  }, []);

  const handleSaveSettings = async () => {
    if (!settings) return;
    setSettingsSaving(true);
    try {
      await settingsApi.update(settings);
    } catch (err) {
      console.error(err);
    } finally {
      setSettingsSaving(false);
    }
  };

  return (
    <div className="flex flex-col gap-6">
      <Card>
        <CardHeader>
          <Building2 className="h-4 w-4 text-neutral-400" />
          <CardTitle>{t.settings.business.title}</CardTitle>
        </CardHeader>
        <div className="divide-y divide-neutral-100">
          <InfoRow label={t.settings.business.nameLabel}       value={tenant?.name} />
          <InfoRow label={t.settings.business.typeLabel}       value={businessTypes[tenant?.business_type ?? ''] ?? tenant?.business_type} />
          <InfoRow label={t.settings.business.bookingUrlLabel} value={`citaspot.com/book/${tenant?.slug ?? ''}`} />
        </div>
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

      {/* Configuración personalizable */}
      {!settingsLoading && settings && (
        <Card>
          <CardHeader>
            <CardTitle>Personalización</CardTitle>
          </CardHeader>

          <div className="space-y-6">
            <div>
              <label className="block text-sm font-medium text-neutral-700 mb-1">Nombre del asistente (bot)</label>
              <input
                type="text"
                className="w-full rounded-lg border border-neutral-300 px-3 py-2 text-sm"
                placeholder="Ej: Luna, Asistente Virtual"
                value={settings.bot_name}
                onChange={e => setSettings({ ...settings, bot_name: e.target.value })}
              />
              <p className="text-xs text-neutral-500 mt-1">Nombre con el que se presentará el bot por WhatsApp</p>
            </div>

            <div>
              <label className="block text-sm font-medium text-neutral-700 mb-1">Saludo inicial del bot</label>
              <textarea
                className="w-full rounded-lg border border-neutral-300 px-3 py-2 text-sm"
                rows={2}
                placeholder="Mensaje de bienvenida personalizado"
                value={settings.bot_greeting}
                onChange={e => setSettings({ ...settings, bot_greeting: e.target.value })}
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-neutral-700 mb-1">Texto antes del formulario de reserva</label>
              <textarea
                className="w-full rounded-lg border border-neutral-300 px-3 py-2 text-sm"
                rows={3}
                placeholder="Texto que verán los clientes al abrir la página de reservas"
                value={settings.booking_intro_text}
                onChange={e => setSettings({ ...settings, booking_intro_text: e.target.value })}
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-neutral-700 mb-1">Texto después de confirmar reserva</label>
              <textarea
                className="w-full rounded-lg border border-neutral-300 px-3 py-2 text-sm"
                rows={3}
                placeholder="Texto que verán los clientes después de reservar"
                value={settings.booking_success_text}
                onChange={e => setSettings({ ...settings, booking_success_text: e.target.value })}
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-neutral-700 mb-1">Recordatorios (minutos antes de la cita)</label>
              <div className="flex flex-wrap gap-2">
                {[
                  { label: '24 horas', value: 1440 },
                  { label: '12 horas', value: 720 },
                  { label: '2 horas', value: 120 },
                  { label: '1 hora', value: 60 },
                  { label: '30 min', value: 30 },
                ].map(opt => (
                  <button
                    key={opt.value}
                    type="button"
                    className={`px-3 py-1.5 rounded-full text-sm border transition-colors ${
                      settings.reminder_minutes.includes(opt.value)
                        ? 'bg-primary-100 border-primary-500 text-primary-700'
                        : 'bg-white border-neutral-300 text-neutral-600 hover:border-neutral-400'
                    }`}
                    onClick={() => {
                      const mins = settings.reminder_minutes.includes(opt.value)
                        ? settings.reminder_minutes.filter(m => m !== opt.value)
                        : [...settings.reminder_minutes, opt.value];
                      setSettings({ ...settings, reminder_minutes: mins });
                    }}
                  >
                    {opt.label}
                  </button>
                ))}
              </div>
              <p className="text-xs text-neutral-500 mt-1">Selecciona cuándo enviar recordatorios por WhatsApp</p>
            </div>

            <Button
              onClick={handleSaveSettings}
              loading={settingsSaving}
            >
              {settingsSaving ? 'Guardando...' : 'Guardar cambios'}
            </Button>
          </div>
        </Card>
      )}
    </div>
  );
}

// ── Tab Cuenta ──────────────────────────────────────────────────────────────

function CuentaTab({ user }: { user: UserDTO | null }) {
  const t = useTranslations();
  const { language, setLanguage } = useLanguage();
  return (
    <Card>
      <CardHeader>
        <User className="h-4 w-4 text-neutral-400" />
        <CardTitle>{t.settings.account.title}</CardTitle>
      </CardHeader>
      <div className="divide-y divide-neutral-100">
        <InfoRow label={t.settings.account.nameLabel}  value={user?.name} />
        <InfoRow label={t.settings.account.emailLabel} value={user?.email} />
        <InfoRow label={t.settings.account.roleLabel}  value={user?.role === 'owner' ? t.settings.account.ownerRole : user?.role} />
      </div>
      {/* Selector de idioma */}
      <div className="mt-4 pt-4 border-t border-neutral-100">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Globe className="h-4 w-4 text-neutral-400" />
            <span className="text-sm text-neutral-600">{t.settings.account.languageLabel}</span>
          </div>
          <div className="flex rounded-lg border border-neutral-200 overflow-hidden text-sm">
            {(['es', 'en', 'pt'] as const).map((lang) => (
              <button
                key={lang}
                onClick={() => setLanguage(lang)}
                className={`px-3 py-1.5 transition-colors ${
                  language === lang
                    ? 'bg-primary-600 text-white font-medium'
                    : 'text-neutral-600 hover:bg-neutral-50'
                }`}
              >
                {t.settings.language[lang]}
              </button>
            ))}
          </div>
        </div>
      </div>
    </Card>
  );
}

// ── Tab Suscripción ─────────────────────────────────────────────────────────

function SuscripcionTab({
  tenant, onUpgrade, checkingOut,
}: {
  tenant: TenantDTO | null;
  onUpgrade: (plan: 'starter' | 'professional') => void;
  checkingOut: boolean;
}) {
  const t = useTranslations();
  const [sub, setSub]           = useState<StripeSubscription | null>(null);
  const [invoices, setInvoices] = useState<StripeInvoice[]>([]);
  const [loadingSub, setLoadingSub]   = useState(true);
  const [loadingInv, setLoadingInv]   = useState(true);
  const [cancelling, setCancelling]   = useState(false);
  const [confirmCancel, setConfirmCancel] = useState(false);
  const [cancelDone, setCancelDone]   = useState(false);

  const planLabels = t.settings.planLabels as Record<string, string>;
  const planLabel  = planLabels[tenant?.plan ?? ''] ?? tenant?.plan ?? '—';
  const hasActiveSub = tenant?.plan_status === 'active' && !!sub && !sub.cancel_at_period_end;
  const isTrial     = tenant?.plan_status === 'trial' || tenant?.plan_status === 'trialing' || tenant?.plan === 'trial';

  const fetchSub = useCallback(async () => {
    setLoadingSub(true);
    try {
      const res = await billing.subscription();
      setSub(res.subscription);
    } catch { /* ignorar */ }
    finally { setLoadingSub(false); }
  }, []);

  const fetchInvoices = useCallback(async () => {
    setLoadingInv(true);
    try {
      const res = await billing.invoices();
      setInvoices(res.data);
    } catch { /* ignorar */ }
    finally { setLoadingInv(false); }
  }, []);

  useEffect(() => {
    fetchSub();
    fetchInvoices();
  }, [fetchSub, fetchInvoices]);

  async function handleCancel() {
    setCancelling(true);
    try {
      const res = await billing.cancel();
      setSub(prev => prev ? { ...prev, cancel_at_period_end: res.cancel_at_period_end } : prev);
      setCancelDone(true);
      setConfirmCancel(false);
    } catch { /* ignorar */ }
    finally { setCancelling(false); }
  }

  function formatDate(iso: string) {
    return new Date(iso).toLocaleDateString(undefined, { day: 'numeric', month: 'long', year: 'numeric' });
  }

  function formatAmount(cents: number, currency: string) {
    return new Intl.NumberFormat(undefined, {
      style: 'currency',
      currency: currency.toUpperCase(),
      minimumFractionDigits: 2,
    }).format(cents / 100);
  }

  return (
    <div className="flex flex-col gap-6">
      {/* Estado del plan */}
      <Card>
        <CardHeader>
          <CreditCard className="h-4 w-4 text-neutral-400" />
          <CardTitle>{t.settings.billing.currentPlan}</CardTitle>
          {tenant && (
            <div className="ml-auto">
              <PlanStatusBadge plan={tenant.plan} planStatus={tenant.plan_status} />
            </div>
          )}
        </CardHeader>

        <div className="divide-y divide-neutral-100">
          <InfoRow label={t.settings.billing.planLabel} value={planLabel} />

          {isTrial && tenant?.trial_ends_at && (
            <InfoRow
              label={t.settings.billing.trialUntil}
              value={formatDate(tenant.trial_ends_at)}
            />
          )}

          {!loadingSub && sub && (
            <>
              <InfoRow
                label={sub.cancel_at_period_end ? t.settings.billing.accessUntil : t.settings.billing.nextRenewal}
                value={formatDate(sub.current_period_end)}
              />
              {sub.cancel_at_period_end && (
                <div className="py-3">
                  <div className="flex items-center gap-2 rounded-lg bg-amber-50 px-3 py-2 text-sm text-amber-700">
                    <AlertTriangle className="h-4 w-4 flex-shrink-0" />
                    <span>
                      {t.settings.billing.subscriptionCancelled.replace('{date}', formatDate(sub.current_period_end))}
                    </span>
                  </div>
                </div>
              )}
            </>
          )}
        </div>

        {(isTrial || (tenant?.plan !== 'professional' && !sub?.cancel_at_period_end)) && (
          <div className="mt-4 border-t border-neutral-100 pt-4">
            <p className="mb-3 text-sm text-neutral-600">
              {isTrial ? t.settings.billing.activatePlan : t.settings.billing.updatePlan}
            </p>
            <div className="flex flex-wrap gap-3">
              {tenant?.plan !== 'starter' && (
                <Button variant="secondary" onClick={() => onUpgrade('starter')} loading={checkingOut}>
                  {t.settings.billing.starterPrice}
                </Button>
              )}
              <Button onClick={() => onUpgrade('professional')} loading={checkingOut}>
                {t.settings.billing.professionalPrice}
              </Button>
            </div>
          </div>
        )}

        {hasActiveSub && !cancelDone && (
          <div className="mt-4 border-t border-neutral-100 pt-4">
            {!confirmCancel ? (
              <Button
                variant="ghost"
                size="sm"
                className="text-red-600 hover:text-red-700 hover:bg-red-50"
                onClick={() => setConfirmCancel(true)}
              >
                <XCircle className="mr-1.5 h-3.5 w-3.5" />
                {t.settings.billing.cancelSubscription}
              </Button>
            ) : (
              <div className="rounded-lg border border-red-200 bg-red-50 p-4">
                <p className="text-sm text-red-700 mb-3">
                  {t.settings.billing.confirmCancelText.replace('{date}', sub ? formatDate(sub.current_period_end) : '...')}
                </p>
                <div className="flex gap-2">
                  <Button variant="danger" size="sm" loading={cancelling} onClick={handleCancel}>
                    {t.settings.billing.yesCancel}
                  </Button>
                  <Button variant="ghost" size="sm" onClick={() => setConfirmCancel(false)}>
                    {t.settings.billing.keepPlan}
                  </Button>
                </div>
              </div>
            )}
          </div>
        )}

        {cancelDone && (
          <div className="mt-4 border-t border-neutral-100 pt-4">
            <div className="flex items-center gap-2 text-sm text-emerald-700">
              <CheckCircle2 className="h-4 w-4" />
              {t.settings.billing.cancellationConfirmed}
            </div>
          </div>
        )}
      </Card>

      {/* Facturas */}
      <Card>
        <CardHeader>
          <Clock className="h-4 w-4 text-neutral-400" />
          <CardTitle>{t.settings.billing.invoiceHistory}</CardTitle>
        </CardHeader>

        {loadingInv ? (
          <div className="flex justify-center py-8">
            <Spinner size="sm" />
          </div>
        ) : invoices.length === 0 ? (
          <p className="py-6 text-center text-sm text-neutral-400">
            {t.settings.billing.noInvoices}
          </p>
        ) : (
          <div className="overflow-x-auto -mx-4 px-0">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-neutral-100">
                  <th className="pb-2 px-4 text-left font-medium text-neutral-500">{t.settings.billing.colDate}</th>
                  <th className="pb-2 px-4 text-left font-medium text-neutral-500">{t.settings.billing.colInvoice}</th>
                  <th className="pb-2 px-4 text-right font-medium text-neutral-500">{t.settings.billing.colAmount}</th>
                  <th className="pb-2 px-4 text-center font-medium text-neutral-500">{t.settings.billing.colStatus}</th>
                  <th className="pb-2 px-4"></th>
                </tr>
              </thead>
              <tbody className="divide-y divide-neutral-50">
                {invoices.map((inv) => (
                  <tr key={inv.id} className="hover:bg-neutral-50 transition-colors">
                    <td className="py-3 px-4 text-neutral-600">{formatDate(inv.created_at)}</td>
                    <td className="py-3 px-4 font-mono text-xs text-neutral-500">{inv.number || '—'}</td>
                    <td className="py-3 px-4 text-right font-medium text-neutral-900">
                      {formatAmount(inv.amount, inv.currency)}
                    </td>
                    <td className="py-3 px-4 text-center">
                      <InvoiceStatusBadge status={inv.status} />
                    </td>
                    <td className="py-3 px-4 text-right">
                      {inv.pdf_url && (
                        <button
                          onClick={() => window.open(inv.pdf_url, '_blank')}
                          className="inline-flex items-center gap-1 text-xs text-violet-600 hover:text-violet-700 font-medium"
                          title="PDF"
                        >
                          <Download className="h-3.5 w-3.5" />
                          PDF
                        </button>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </Card>
    </div>
  );
}

// ── Página principal ─────────────────────────────────────────────────────────

export default function SettingsPage() {
  const t = useTranslations();
  const [user,   setUser]   = useState<UserDTO | null>(null);
  const [tenant, setTenant] = useState<TenantDTO | null>(null);
  const [loading, setLoading] = useState(true);
  const [tab, setTab]   = useState<Tab>('negocio');
  const [checkingOut, setCheckingOut] = useState(false);

  useEffect(() => {
    auth.me()
      .then(({ user: u, tenant: tn }) => { setUser(u); setTenant(tn); })
      .catch(() => { /* ignorar */ })
      .finally(() => setLoading(false));
  }, []);

  async function handleUpgrade(plan: 'starter' | 'professional') {
    setCheckingOut(true);
    try {
      const data = await billing.checkout(
        plan,
        `${window.location.origin}/dashboard/settings?success=1`,
        `${window.location.origin}/dashboard/settings`,
      );
      if (data.url) window.location.href = data.url;
    } catch { /* ignorar */ }
    finally { setCheckingOut(false); }
  }

  if (loading) {
    return <div className="flex justify-center py-16"><Spinner size="lg" /></div>;
  }

  return (
    <div className="p-6">
      <div className="mb-6">
        <h1 className="text-xl font-semibold text-neutral-900">{t.settings.title}</h1>
        <p className="mt-0.5 text-sm text-neutral-500">{t.settings.description}</p>
      </div>

      {/* Tabs */}
      <div className="mb-6 flex rounded-xl bg-neutral-100 p-1 gap-1">
        <TabButton active={tab === 'negocio'}     onClick={() => setTab('negocio')}>{t.settings.tabs.business}</TabButton>
        <TabButton active={tab === 'cuenta'}      onClick={() => setTab('cuenta')}>{t.settings.tabs.account}</TabButton>
        <TabButton active={tab === 'suscripcion'} onClick={() => setTab('suscripcion')}>{t.settings.tabs.billing}</TabButton>
      </div>

      <div className="max-w-2xl">
        {tab === 'negocio'     && <NegocioTab tenant={tenant} />}
        {tab === 'cuenta'      && <CuentaTab user={user} />}
        {tab === 'suscripcion' && (
          <SuscripcionTab
            tenant={tenant}
            onUpgrade={handleUpgrade}
            checkingOut={checkingOut}
          />
        )}
      </div>
    </div>
  );
}
