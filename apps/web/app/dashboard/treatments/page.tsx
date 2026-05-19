'use client';

import { useState, useEffect, useCallback } from 'react';
import Link from 'next/link';
import {
  Stethoscope, Search,
} from 'lucide-react';
import { Spinner } from '@/components/ui/spinner';
import {
  treatments as treatmentsApi, Treatment,
  professionals, Professional,
} from '@/lib/api';
import { useTranslations } from '@/lib/i18n';

type StatusFilter = '' | 'proposed' | 'accepted' | 'in_progress' | 'completed' | 'abandoned';

const STATUS_CFG: Record<string, { bg: string; text: string }> = {
  proposed:    { bg: 'bg-neutral-100',  text: 'text-neutral-600'  },
  accepted:    { bg: 'bg-blue-100',     text: 'text-blue-700'     },
  in_progress: { bg: 'bg-violet-100',   text: 'text-violet-700'   },
  completed:   { bg: 'bg-emerald-100',  text: 'text-emerald-700'  },
  abandoned:   { bg: 'bg-red-100',      text: 'text-red-600'      },
};

export default function TreatmentsPage() {
  const t = useTranslations();

  const [list, setList]             = useState<Treatment[]>([]);
  const [profList, setProfList]     = useState<Professional[]>([]);
  const [loading, setLoading]       = useState(true);
  const [search, setSearch]         = useState('');
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('');
  const [profFilter, setProfFilter] = useState('');

  // Mapa de status -> label
  const statusLabel: Record<string, string> = {
    proposed:    t.crm.treatments.statusProposed,
    accepted:    t.crm.treatments.statusAccepted,
    in_progress: t.crm.treatments.statusInProgress,
    completed:   t.crm.treatments.statusCompleted,
    abandoned:   t.crm.treatments.statusAbandoned,
  };

  const load = useCallback(async (q: string) => {
    setLoading(true);
    try {
      const params: Record<string, string> = {};
      if (statusFilter) params.status = statusFilter;
      if (profFilter) params.professional_id = profFilter;
      if (q) params.search = q;
      const res = await treatmentsApi.list(params);
      setList(res.data ?? []);
    } catch {
      setList([]);
    } finally {
      setLoading(false);
    }
  }, [statusFilter, profFilter]);

  // Busqueda con debounce
  useEffect(() => {
    const timer = setTimeout(() => load(search), 300);
    return () => clearTimeout(timer);
  }, [search, load]);

  useEffect(() => {
    professionals.list().then(r => setProfList(r.data ?? [])).catch(() => {});
  }, []);

  return (
    <div className="p-6">
      {/* Header */}
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold text-neutral-900">{t.crm.treatments.title}</h1>
          <p className="mt-0.5 text-sm text-neutral-500">{t.crm.treatments.description}</p>
        </div>
        <div className="flex items-center gap-2 text-sm text-neutral-500">
          <Stethoscope className="h-4 w-4" />
          <span>{t.crm.treatments.treatmentsCount.replace('{n}', String(list.length))}</span>
        </div>
      </div>

      {/* Buscador + filtros */}
      <div className="mb-4 flex flex-wrap items-center gap-2">
        <div className="relative max-w-sm flex-1">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-neutral-400" />
          <input
            type="text"
            placeholder={t.crm.treatments.searchPlaceholder}
            value={search}
            onChange={e => setSearch(e.target.value)}
            className="w-full rounded-lg border border-neutral-200 bg-white py-2 pl-9 pr-3 text-sm text-neutral-900 placeholder:text-neutral-400 focus:border-primary-400 focus:outline-none focus:ring-2 focus:ring-primary-100"
          />
        </div>

        <select
          value={statusFilter}
          onChange={e => setStatusFilter(e.target.value as StatusFilter)}
          className="rounded-lg border border-neutral-200 bg-white px-2 py-1.5 text-xs text-neutral-700 focus:outline-none focus:ring-1 focus:ring-primary-400"
        >
          <option value="">{t.crm.treatments.filterStatus}: {t.crm.treatments.filterAll}</option>
          <option value="proposed">{t.crm.treatments.statusProposed}</option>
          <option value="accepted">{t.crm.treatments.statusAccepted}</option>
          <option value="in_progress">{t.crm.treatments.statusInProgress}</option>
          <option value="completed">{t.crm.treatments.statusCompleted}</option>
          <option value="abandoned">{t.crm.treatments.statusAbandoned}</option>
        </select>

        <select
          value={profFilter}
          onChange={e => setProfFilter(e.target.value)}
          className="rounded-lg border border-neutral-200 bg-white px-2 py-1.5 text-xs text-neutral-700 focus:outline-none focus:ring-1 focus:ring-primary-400"
        >
          <option value="">{t.crm.treatments.filterProfessional}: {t.crm.treatments.filterAll}</option>
          {profList.filter(p => p.is_active).map(p => (
            <option key={p.id} value={p.id}>{p.name}</option>
          ))}
        </select>
      </div>

      {/* Contenido */}
      {loading ? (
        <div className="flex justify-center py-16"><Spinner size="lg" /></div>
      ) : list.length === 0 ? (
        <div className="flex flex-col items-center px-4 py-12 text-center">
          <Stethoscope className="mb-3 h-12 w-12 text-neutral-200" />
          <p className="text-base font-semibold text-neutral-700">{t.crm.treatments.noTreatments}</p>
          <p className="mt-1 max-w-md text-sm text-neutral-500">{t.crm.treatments.noTreatmentsDesc}</p>

          {/* Tarjeta-ejemplo read-only para que el usuario entienda como se ve un tratamiento */}
          <div className="mt-8 w-full max-w-xl">
            <div className="mb-2 flex items-center justify-between">
              <span className="inline-flex items-center rounded-full bg-amber-100 px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-amber-700">
                {t.crm.treatments.emptyExampleLabel}
              </span>
              <span className="text-xs text-neutral-400">{t.crm.treatments.emptyExampleHint}</span>
            </div>

            <div className="rounded-xl border border-dashed border-neutral-300 bg-white p-5 text-left shadow-sm">
              <div className="mb-3 flex items-start justify-between gap-3">
                <div>
                  <p className="font-semibold text-neutral-900">{t.crm.treatments.emptyExampleName}</p>
                  <p className="text-xs text-neutral-400">{t.crm.treatments.emptyExampleType}</p>
                </div>
                <span className={`inline-flex shrink-0 items-center rounded-full px-2 py-0.5 text-[10px] font-medium ${STATUS_CFG.in_progress.bg} ${STATUS_CFG.in_progress.text}`}>
                  {t.crm.treatments.statusInProgress}
                </span>
              </div>

              <div className="mb-3 grid grid-cols-2 gap-3 text-xs">
                <div>
                  <p className="text-neutral-400">{t.crm.treatments.customer}</p>
                  <p className="font-medium text-neutral-700">{t.crm.treatments.emptyExampleCustomer}</p>
                </div>
                <div>
                  <p className="text-neutral-400">{t.crm.treatments.professional}</p>
                  <p className="font-medium text-neutral-700">{t.crm.treatments.emptyExampleProfessional}</p>
                </div>
              </div>

              <div className="mb-3">
                <div className="mb-1 flex items-center justify-between text-xs">
                  <span className="text-neutral-500">{t.crm.treatments.sessions}</span>
                  <span className="text-neutral-600">{t.crm.treatments.emptyExampleSessions}</span>
                </div>
                <div className="h-1.5 w-full rounded-full bg-neutral-100">
                  <div className="h-full rounded-full bg-primary-500" style={{ width: '16.6%' }} />
                </div>
              </div>

              <div className="grid grid-cols-2 gap-3 border-t border-neutral-100 pt-3 text-xs">
                <div>
                  <p className="text-neutral-400">{t.crm.treatments.cost}</p>
                  <p className="font-medium text-neutral-700">{t.crm.treatments.emptyExampleCost}</p>
                </div>
                <div>
                  <p className="text-neutral-400">{t.crm.treatments.paid}</p>
                  <p className="font-medium text-emerald-600">{t.crm.treatments.emptyExamplePaid}</p>
                </div>
              </div>
            </div>
          </div>
        </div>
      ) : (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {list.map(tr => {
            const sc = STATUS_CFG[tr.status] ?? STATUS_CFG.proposed;
            const progressPct = tr.total_sessions
              ? Math.min(100, Math.round((tr.completed_sessions / tr.total_sessions) * 100))
              : 0;

            return (
              <Link
                key={tr.id}
                href={`/dashboard/treatments/${tr.id}`}
                className="block rounded-xl border border-neutral-200 bg-white p-5 shadow-sm transition-shadow hover:shadow-md focus:outline-none focus:ring-2 focus:ring-primary-300"
              >
                {/* Nombre + status */}
                <div className="mb-3 flex items-start justify-between gap-3">
                  <div>
                    <p className="font-semibold text-neutral-900">{tr.name}</p>
                    <p className="text-xs text-neutral-400">{tr.treatment_type}</p>
                  </div>
                  <span className={`inline-flex shrink-0 items-center rounded-full px-2 py-0.5 text-[10px] font-medium ${sc.bg} ${sc.text}`}>
                    {statusLabel[tr.status] ?? tr.status}
                  </span>
                </div>

                {/* Cliente + profesional */}
                <div className="mb-3 grid grid-cols-2 gap-3 text-xs">
                  <div>
                    <p className="text-neutral-400">{t.crm.treatments.customer}</p>
                    <p className="font-medium text-neutral-700">{tr.customer_name ?? '—'}</p>
                  </div>
                  <div>
                    <p className="text-neutral-400">{t.crm.treatments.professional}</p>
                    <p className="font-medium text-neutral-700">{tr.professional_name ?? '—'}</p>
                  </div>
                </div>

                {/* Progreso de sesiones */}
                {tr.total_sessions ? (
                  <div className="mb-3">
                    <div className="mb-1 flex items-center justify-between text-xs">
                      <span className="text-neutral-500">{t.crm.treatments.sessions}</span>
                      <span className="text-neutral-600">
                        {t.crm.treatments.sessionsProgress
                          .replace('{completed}', String(tr.completed_sessions))
                          .replace('{total}', String(tr.total_sessions))}
                      </span>
                    </div>
                    <div className="h-1.5 w-full rounded-full bg-neutral-100">
                      <div
                        className="h-full rounded-full bg-primary-500 transition-all"
                        style={{ width: `${progressPct}%` }}
                      />
                    </div>
                  </div>
                ) : null}

                {/* Costo + pagado */}
                <div className="grid grid-cols-2 gap-3 border-t border-neutral-100 pt-3 text-xs">
                  <div>
                    <p className="text-neutral-400">{t.crm.treatments.cost}</p>
                    <p className="font-medium text-neutral-700">
                      {tr.estimated_cost != null ? `$${tr.estimated_cost.toFixed(2)} ${tr.currency}` : '—'}
                    </p>
                  </div>
                  <div>
                    <p className="text-neutral-400">{t.crm.treatments.paid}</p>
                    <p className={tr.paid_amount > 0 ? 'font-medium text-emerald-600' : 'font-medium text-neutral-300'}>
                      {tr.paid_amount > 0 ? `$${tr.paid_amount.toFixed(2)} ${tr.currency}` : '—'}
                    </p>
                  </div>
                </div>
              </Link>
            );
          })}
        </div>
      )}
    </div>
  );
}
