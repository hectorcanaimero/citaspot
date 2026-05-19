'use client';

import { useState, useEffect } from 'react';
import { format, subDays } from 'date-fns';
import { TrendingUp, TrendingDown, Calendar, Users, CheckCircle, DollarSign } from 'lucide-react';
import { Card, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge }   from '@/components/ui/badge';
import { Spinner } from '@/components/ui/spinner';
import { appointments, Appointment } from '@/lib/api';
import { cn } from '@/lib/utils';
import { useTranslations } from '@/lib/i18n';
import { useTenantTimezone } from '@/store/tenant';

interface KPI {
  label: string;
  value: string | number;
  icon: React.ElementType;
  trend?: number;
  variant: 'primary' | 'success' | 'warning' | 'default';
}

function KPICard({ kpi }: { kpi: KPI }) {
  const t = useTranslations();
  const Icon = kpi.icon;
  const iconColors: Record<string, string> = {
    primary: 'bg-primary-100 text-primary-700',
    success: 'bg-emerald-100 text-emerald-700',
    warning: 'bg-amber-100 text-amber-700',
    default: 'bg-neutral-100 text-neutral-700',
  };

  return (
    <Card className="flex items-center gap-4">
      <div className={cn('flex h-11 w-11 flex-shrink-0 items-center justify-center rounded-xl', iconColors[kpi.variant])}>
        <Icon className="h-5 w-5" />
      </div>
      <div className="min-w-0">
        <p className="text-xs text-neutral-500">{kpi.label}</p>
        <p className="text-xl font-bold text-neutral-900">{kpi.value}</p>
        {kpi.trend !== undefined && (
          <div className="flex items-center gap-1 text-xs">
            {kpi.trend >= 0
              ? <TrendingUp className="h-3 w-3 text-emerald-600" />
              : <TrendingDown className="h-3 w-3 text-red-500" />
            }
            <span className={kpi.trend >= 0 ? 'text-emerald-600' : 'text-red-500'}>
              {Math.abs(kpi.trend)}{t.analytics.vsPrevMonth}
            </span>
          </div>
        )}
      </div>
    </Card>
  );
}

function StatusBar({ label, count, total, color }: { label: string; count: number; total: number; color: string }) {
  const pct = total > 0 ? Math.round((count / total) * 100) : 0;
  return (
    <div className="flex items-center gap-3 text-sm">
      <div className="w-28 flex-shrink-0 text-neutral-600">{label}</div>
      <div className="flex-1 rounded-full bg-neutral-100 h-2 overflow-hidden">
        <div className={`h-full rounded-full transition-all duration-500 ${color}`} style={{ width: `${pct}%` }} />
      </div>
      <div className="w-14 flex-shrink-0 text-right text-neutral-700">
        <span className="font-medium">{count}</span>
        <span className="text-neutral-400"> ({pct}%)</span>
      </div>
    </div>
  );
}

export default function AnalyticsPage() {
  const t = useTranslations();
  const [loading, setLoading] = useState(true);
  const [appts, setAppts]     = useState<Appointment[]>([]);
  const tz = useTenantTimezone();

  useEffect(() => {
    const today = new Date();
    const load = async () => {
      const all: Appointment[] = [];
      for (let i = 0; i < 30; i++) {
        const date = format(subDays(today, i), 'yyyy-MM-dd');
        try {
          const res = await appointments.list(date, tz);
          all.push(...res.data);
        } catch { /* ignorar días sin datos */ }
      }
      setAppts(all);
      setLoading(false);
    };
    load();
  }, [tz]);

  const total       = appts.length;
  const completed   = appts.filter((a) => a.status === 'completed').length;
  const pending     = appts.filter((a) => a.status === 'pending').length;
  const confirmed   = appts.filter((a) => a.status === 'confirmed').length;
  const cancelled   = appts.filter((a) => a.status === 'cancelled').length;
  const noShow      = appts.filter((a) => a.status === 'no_show').length;
  const completionRate = total > 0 ? Math.round((completed / total) * 100) : 0;

  const revenue = appts
    .filter((a) => a.status === 'completed' && a.price)
    .reduce((sum, a) => sum + (a.price ?? 0), 0);

  const profCounts: Record<string, { name: string; count: number }> = {};
  for (const a of appts.filter((x) => x.status === 'completed')) {
    if (!profCounts[a.professional_id]) {
      profCounts[a.professional_id] = { name: a.professional_name, count: 0 };
    }
    profCounts[a.professional_id].count++;
  }
  const topProfs = Object.values(profCounts).sort((a, b) => b.count - a.count).slice(0, 5);

  const kpis: KPI[] = [
    { label: t.analytics.appointments30d,  value: total,                                                 icon: Calendar,     variant: 'primary' },
    { label: t.analytics.completed,        value: completed,                                             icon: CheckCircle,  variant: 'success' },
    { label: t.analytics.completionRate,   value: `${completionRate}%`,                                  icon: TrendingUp,   variant: 'success' },
    { label: t.analytics.estimatedRevenue, value: revenue > 0 ? `$${revenue.toFixed(0)}` : '—',         icon: DollarSign,   variant: 'default' },
    { label: t.analytics.uniqueClients,    value: new Set(appts.map((a) => a.customer_id)).size,         icon: Users,        variant: 'primary' },
    { label: t.analytics.pendingToday,     value: pending,                                               icon: Calendar,     variant: 'warning' },
  ];

  return (
    <div className="p-6">
      <div className="mb-6">
        <h1 className="text-xl font-semibold text-neutral-900">{t.analytics.title}</h1>
        <p className="mt-0.5 text-sm text-neutral-500">{t.analytics.description}</p>
      </div>

      {loading ? (
        <div className="flex justify-center py-16"><Spinner size="lg" /></div>
      ) : (
        <div className="flex flex-col gap-6">
          {/* KPIs */}
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {kpis.map((k) => <KPICard key={k.label} kpi={k} />)}
          </div>

          <div className="grid gap-6 lg:grid-cols-2">
            {/* Distribución de estados */}
            <Card>
              <CardHeader>
                <CardTitle>{t.analytics.appointmentStatus}</CardTitle>
              </CardHeader>
              <div className="flex flex-col gap-3">
                <StatusBar label={t.analytics.completed}  count={completed}  total={total} color="bg-emerald-500" />
                <StatusBar label={t.analytics.confirmed}  count={confirmed}  total={total} color="bg-primary-500" />
                <StatusBar label={t.analytics.pending}    count={pending}    total={total} color="bg-amber-400" />
                <StatusBar label={t.analytics.cancelled}  count={cancelled}  total={total} color="bg-neutral-300" />
                <StatusBar label={t.analytics.noShow}     count={noShow}     total={total} color="bg-red-300" />
              </div>
            </Card>

            {/* Top profesionales */}
            <Card>
              <CardHeader>
                <CardTitle>{t.analytics.topProfessionals}</CardTitle>
                <Badge variant="primary">{topProfs.length}</Badge>
              </CardHeader>
              {topProfs.length === 0 ? (
                <p className="text-sm text-neutral-400">{t.analytics.noData}</p>
              ) : (
                <div className="flex flex-col gap-2">
                  {topProfs.map((p, i) => (
                    <div key={p.name} className="flex items-center gap-3 text-sm">
                      <span className="flex h-6 w-6 flex-shrink-0 items-center justify-center rounded-full bg-primary-100 text-xs font-bold text-primary-700">
                        {i + 1}
                      </span>
                      <span className="flex-1 text-neutral-700">{p.name}</span>
                      <Badge variant="default">{p.count} {t.analytics.appointmentsUnit}</Badge>
                    </div>
                  ))}
                </div>
              )}
            </Card>
          </div>
        </div>
      )}
    </div>
  );
}
