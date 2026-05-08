'use client';

import { useState, useEffect, useCallback } from 'react';
import {
  CreditCard, Download, AlertTriangle, CheckCircle2, Clock, XCircle,
} from 'lucide-react';
import { Card, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Spinner } from '@/components/ui/spinner';
import {
  auth, billing,
  TenantDTO, StripeSubscription, StripeInvoice,
} from '@/lib/api';
import { useTranslations } from '@/lib/i18n';

// -- Helpers ------------------------------------------------------------------

function InfoRow({ label, value }: { label: string; value: string | undefined }) {
  return (
    <div className="flex items-center justify-between py-2 text-sm">
      <span className="text-neutral-500">{label}</span>
      <span className="font-medium text-neutral-900">{value ?? '\u2014'}</span>
    </div>
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

// -- Page ---------------------------------------------------------------------

export default function SubscriptionPage() {
  const t = useTranslations();

  const [tenant, setTenant]     = useState<TenantDTO | null>(null);
  const [sub, setSub]           = useState<StripeSubscription | null>(null);
  const [invoices, setInvoices] = useState<StripeInvoice[]>([]);
  const [loading, setLoading]       = useState(true);
  const [loadingSub, setLoadingSub] = useState(true);
  const [loadingInv, setLoadingInv] = useState(true);
  const [checkingOut, setCheckingOut] = useState(false);
  const [cancelling, setCancelling]   = useState(false);
  const [confirmCancel, setConfirmCancel] = useState(false);
  const [cancelDone, setCancelDone]   = useState(false);

  const planLabels = t.settings.planLabels as Record<string, string>;
  const planLabel  = planLabels[tenant?.plan ?? ''] ?? tenant?.plan ?? '\u2014';
  const hasActiveSub = tenant?.plan_status === 'active' && !!sub && !sub.cancel_at_period_end;
  const isTrial     = tenant?.plan_status === 'trial' || tenant?.plan_status === 'trialing' || tenant?.plan === 'trial';

  useEffect(() => {
    auth.me()
      .then(({ tenant: tn }) => setTenant(tn))
      .catch(() => {})
      .finally(() => setLoading(false));
  }, []);

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

  async function handleUpgrade(plan: 'starter' | 'professional') {
    setCheckingOut(true);
    try {
      const data = await billing.checkout(
        plan,
        `${window.location.origin}/dashboard/settings/subscription?success=1`,
        `${window.location.origin}/dashboard/settings/subscription`,
      );
      if (data.url) window.location.href = data.url;
    } catch { /* ignorar */ }
    finally { setCheckingOut(false); }
  }

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

  if (loading) return <div className="flex justify-center py-16"><Spinner size="lg" /></div>;

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
                <Button variant="secondary" onClick={() => handleUpgrade('starter')} loading={checkingOut}>
                  {t.settings.billing.starterPrice}
                </Button>
              )}
              <Button onClick={() => handleUpgrade('professional')} loading={checkingOut}>
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
                    <td className="py-3 px-4 font-mono text-xs text-neutral-500">{inv.number || '\u2014'}</td>
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
