'use client';

import { useState, useEffect, useCallback } from 'react';
import {
  Check, X, UserX, ChevronLeft, ChevronRight,
  ChevronDown, ChevronUp, ClipboardList, Search,
} from 'lucide-react';
import { format, startOfWeek, endOfWeek, parseISO } from 'date-fns';
import { useTranslations, useDateLocale } from '@/lib/i18n';
import {
  appointments, professionals, services,
  Appointment, PaginatedAppointments, Professional, Service,
} from '@/lib/api';

// ── Helpers ───────────────────────────────────────────────────────────────────

function toDateInput(d: Date): string {
  return format(d, 'yyyy-MM-dd');
}

function capitalize(s: string): string {
  return s.charAt(0).toUpperCase() + s.slice(1);
}

const STATUS_BADGE: Record<Appointment['status'], string> = {
  pending:   'bg-amber-100 text-amber-700',
  confirmed: 'bg-primary-100 text-primary-700',
  completed: 'bg-emerald-100 text-emerald-700',
  cancelled: 'bg-red-100 text-red-700',
  no_show:   'bg-neutral-100 text-neutral-600',
};

type SortField = 'starts_at' | 'customer_name';
type SortDir = 'asc' | 'desc';

// ── Componente principal ──────────────────────────────────────────────────────

export default function AppointmentsList() {
  const t = useTranslations();
  const dateLocale = useDateLocale();

  // Filtros
  const now = new Date();
  const [dateFrom, setDateFrom] = useState(() => toDateInput(startOfWeek(now, { weekStartsOn: 1 })));
  const [dateTo, setDateTo]     = useState(() => toDateInput(endOfWeek(now, { weekStartsOn: 1 })));
  const [profFilter, setProfFilter]     = useState('');
  const [svcFilter, setSvcFilter]       = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const [searchText, setSearchText]     = useState('');
  const [debouncedSearch, setDebouncedSearch] = useState('');

  // Ordenamiento
  const [sortBy, setSortBy]   = useState<SortField>('starts_at');
  const [sortDir, setSortDir] = useState<SortDir>('asc');

  // Paginación
  const [page, setPage] = useState(1);
  const perPage = 20;

  // Datos
  const [result, setResult] = useState<PaginatedAppointments | null>(null);
  const [loading, setLoading] = useState(true);
  const [profList, setProfList] = useState<Professional[]>([]);
  const [svcList, setSvcList]   = useState<Service[]>([]);

  // Cargar profesionales y servicios para filtros
  useEffect(() => {
    professionals.list().then(r => setProfList(r.data ?? [])).catch(() => {});
    services.list().then(r => setSvcList((r.data ?? []).filter(s => s.is_active))).catch(() => {});
  }, []);

  // Debounce de búsqueda
  useEffect(() => {
    const timer = setTimeout(() => setDebouncedSearch(searchText), 300);
    return () => clearTimeout(timer);
  }, [searchText]);

  // Reset página cuando filtros cambian
  useEffect(() => {
    setPage(1);
  }, [dateFrom, dateTo, profFilter, svcFilter, statusFilter, debouncedSearch]);

  // Fetch de citas
  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await appointments.listFiltered({
        date_from: dateFrom,
        date_to: dateTo,
        professional_id: profFilter || undefined,
        service_id: svcFilter || undefined,
        status: statusFilter || undefined,
        search: debouncedSearch || undefined,
        sort_by: sortBy,
        sort_dir: sortDir,
        page,
        per_page: perPage,
      });
      setResult(res);
    } catch {
      setResult(null);
    } finally {
      setLoading(false);
    }
  }, [dateFrom, dateTo, profFilter, svcFilter, statusFilter, debouncedSearch, sortBy, sortDir, page]);

  useEffect(() => { fetchData(); }, [fetchData]);

  // Acciones
  async function handleAction(id: string, status: string) {
    try {
      await appointments.updateStatus(id, { status });
      fetchData();
    } catch {
      // Silenciar — se podría agregar toast en el futuro
    }
  }

  // Toggle sort
  function toggleSort(field: SortField) {
    if (sortBy === field) {
      setSortDir(d => d === 'asc' ? 'desc' : 'asc');
    } else {
      setSortBy(field);
      setSortDir('asc');
    }
  }

  function SortIcon({ field }: { field: SortField }) {
    if (sortBy !== field) return <ChevronDown className="h-3 w-3 opacity-30" />;
    return sortDir === 'asc'
      ? <ChevronUp className="h-3 w-3" />
      : <ChevronDown className="h-3 w-3" />;
  }

  const data       = result?.data ?? [];
  const pagination = result?.pagination;
  const totalPages = pagination?.total_pages ?? 1;
  const total      = pagination?.total ?? 0;
  const from       = total === 0 ? 0 : (page - 1) * perPage + 1;
  const to         = Math.min(page * perPage, total);

  return (
    <div className="flex h-full flex-col overflow-hidden p-4">

      {/* ── Filtros ──────────────────────────────────────────────────────────── */}
      <div className="mb-4 flex flex-wrap items-center gap-2 rounded-xl border border-neutral-200 bg-neutral-50 p-3">
        <input
          type="date"
          value={dateFrom}
          onChange={e => setDateFrom(e.target.value)}
          className="rounded-lg border border-neutral-200 bg-white px-2 py-1.5 text-xs text-neutral-700 focus:border-primary-400 focus:outline-none focus:ring-1 focus:ring-primary-100"
        />
        <input
          type="date"
          value={dateTo}
          onChange={e => setDateTo(e.target.value)}
          className="rounded-lg border border-neutral-200 bg-white px-2 py-1.5 text-xs text-neutral-700 focus:border-primary-400 focus:outline-none focus:ring-1 focus:ring-primary-100"
        />

        <select
          value={profFilter}
          onChange={e => setProfFilter(e.target.value)}
          className="rounded-lg border border-neutral-200 bg-white px-2 py-1.5 text-xs text-neutral-700 focus:border-primary-400 focus:outline-none focus:ring-1 focus:ring-primary-100"
        >
          <option value="">{t.agenda.filterAll} &mdash; {t.agenda.filterProfessional}</option>
          {profList.filter(p => p.is_active).map(p => (
            <option key={p.id} value={p.id}>{p.name}</option>
          ))}
        </select>

        <select
          value={svcFilter}
          onChange={e => setSvcFilter(e.target.value)}
          className="rounded-lg border border-neutral-200 bg-white px-2 py-1.5 text-xs text-neutral-700 focus:border-primary-400 focus:outline-none focus:ring-1 focus:ring-primary-100"
        >
          <option value="">{t.agenda.filterAll} &mdash; {t.agenda.filterService}</option>
          {svcList.map(s => (
            <option key={s.id} value={s.id}>{s.name}</option>
          ))}
        </select>

        <select
          value={statusFilter}
          onChange={e => setStatusFilter(e.target.value)}
          className="rounded-lg border border-neutral-200 bg-white px-2 py-1.5 text-xs text-neutral-700 focus:border-primary-400 focus:outline-none focus:ring-1 focus:ring-primary-100"
        >
          <option value="">{t.agenda.filterAll} &mdash; {t.agenda.filterStatus}</option>
          <option value="pending">{t.status.pending}</option>
          <option value="confirmed">{t.status.confirmed}</option>
          <option value="completed">{t.status.completed}</option>
          <option value="cancelled">{t.status.cancelled}</option>
          <option value="no_show">{t.status.noShow}</option>
        </select>

        <div className="relative">
          <Search className="absolute left-2 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-neutral-400" />
          <input
            type="text"
            value={searchText}
            onChange={e => setSearchText(e.target.value)}
            placeholder={t.agenda.filterSearch}
            className="rounded-lg border border-neutral-200 bg-white py-1.5 pl-7 pr-2 text-xs text-neutral-700 placeholder:text-neutral-400 focus:border-primary-400 focus:outline-none focus:ring-1 focus:ring-primary-100"
          />
        </div>
      </div>

      {/* ── Tabla ────────────────────────────────────────────────────────────── */}
      {loading ? (
        <div className="overflow-hidden rounded-xl border border-neutral-200 bg-white">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-neutral-100 bg-neutral-50 text-left text-xs font-medium uppercase tracking-wide text-neutral-500">
                <th className="px-4 py-3">{t.agenda.colDate}</th>
                <th className="px-4 py-3">{t.agenda.colTime}</th>
                <th className="px-4 py-3">{t.agenda.colClient}</th>
                <th className="px-4 py-3">{t.agenda.colService}</th>
                <th className="px-4 py-3">{t.agenda.colProfessional}</th>
                <th className="px-4 py-3">{t.agenda.colStatus}</th>
                <th className="px-4 py-3">{t.agenda.colActions}</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-neutral-100">
              {Array.from({ length: 5 }).map((_, i) => (
                <tr key={i}>
                  <td className="px-4 py-3"><div className="h-4 w-20 animate-pulse rounded bg-neutral-200" /></td>
                  <td className="px-4 py-3"><div className="h-4 w-16 animate-pulse rounded bg-neutral-200" /></td>
                  <td className="px-4 py-3"><div className="h-4 w-28 animate-pulse rounded bg-neutral-200" /></td>
                  <td className="px-4 py-3"><div className="h-4 w-24 animate-pulse rounded bg-neutral-200" /></td>
                  <td className="px-4 py-3"><div className="h-4 w-24 animate-pulse rounded bg-neutral-200" /></td>
                  <td className="px-4 py-3"><div className="h-4 w-16 animate-pulse rounded bg-neutral-200" /></td>
                  <td className="px-4 py-3"><div className="h-4 w-12 animate-pulse rounded bg-neutral-200" /></td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : data.length === 0 ? (
        <div className="flex flex-1 flex-col items-center justify-center py-16 text-center">
          <ClipboardList className="mb-3 h-12 w-12 text-neutral-200" />
          <p className="text-sm font-medium text-neutral-500">{t.agenda.emptyTitle}</p>
          <p className="mt-1 text-xs text-neutral-400">{t.agenda.emptySubtitle}</p>
        </div>
      ) : (
        <div className="flex flex-1 flex-col overflow-hidden rounded-xl border border-neutral-200 bg-white">
          <div className="flex-1 overflow-auto">
            <table className="w-full text-sm">
              <thead className="sticky top-0 z-10">
                <tr className="border-b border-neutral-100 bg-neutral-50 text-left text-xs font-medium uppercase tracking-wide text-neutral-500">
                  <th
                    className="cursor-pointer px-4 py-3 select-none"
                    onClick={() => toggleSort('starts_at')}
                  >
                    <span className="inline-flex items-center gap-1">
                      {t.agenda.colDate} <SortIcon field="starts_at" />
                    </span>
                  </th>
                  <th
                    className="cursor-pointer px-4 py-3 select-none"
                    onClick={() => toggleSort('starts_at')}
                  >
                    <span className="inline-flex items-center gap-1">
                      {t.agenda.colTime} <SortIcon field="starts_at" />
                    </span>
                  </th>
                  <th
                    className="cursor-pointer px-4 py-3 select-none"
                    onClick={() => toggleSort('customer_name')}
                  >
                    <span className="inline-flex items-center gap-1">
                      {t.agenda.colClient} <SortIcon field="customer_name" />
                    </span>
                  </th>
                  <th className="px-4 py-3">{t.agenda.colService}</th>
                  <th className="px-4 py-3">{t.agenda.colProfessional}</th>
                  <th className="px-4 py-3">{t.agenda.colStatus}</th>
                  <th className="px-4 py-3">{t.agenda.colActions}</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-neutral-100">
                {data.map(appt => {
                  const dt = parseISO(appt.starts_at);
                  return (
                    <tr key={appt.id} className="hover:bg-neutral-50 transition-colors">
                      <td className="px-4 py-3 text-neutral-700">
                        {capitalize(format(dt, 'EEE d MMM', { locale: dateLocale }))}
                      </td>
                      <td className="px-4 py-3 text-neutral-700">
                        {format(dt, 'hh:mm a', { locale: dateLocale })}
                      </td>
                      <td className="px-4 py-3 font-medium text-neutral-900">
                        {appt.customer_name}
                      </td>
                      <td className="px-4 py-3 text-neutral-600">
                        {appt.service_name}
                      </td>
                      <td className="px-4 py-3 text-neutral-600">
                        {appt.professional_name}
                      </td>
                      <td className="px-4 py-3">
                        <span className={`inline-block rounded-full px-2 py-0.5 text-xs font-medium ${STATUS_BADGE[appt.status]}`}>
                          {t.status[appt.status === 'no_show' ? 'noShow' : appt.status]}
                        </span>
                      </td>
                      <td className="px-4 py-3">
                        <ActionButtons appt={appt} onAction={handleAction} t={t} />
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>

          {/* ── Paginación ────────────────────────────────────────────────────── */}
          <div className="flex items-center justify-between border-t border-neutral-200 bg-neutral-50 px-4 py-2.5">
            <span className="text-xs text-neutral-500">
              {t.agenda.showingOf
                .replace('{from}', String(from))
                .replace('{to}', String(to))
                .replace('{total}', String(total))}
            </span>
            <div className="flex items-center gap-2">
              <button
                onClick={() => setPage(p => Math.max(1, p - 1))}
                disabled={page <= 1}
                className="flex items-center gap-1 rounded-lg px-2 py-1 text-xs font-medium text-neutral-600 transition-colors hover:bg-neutral-200 disabled:opacity-40 disabled:cursor-not-allowed"
              >
                <ChevronLeft className="h-3.5 w-3.5" />
                {t.agenda.prevPage}
              </button>
              <span className="text-xs text-neutral-500">
                {t.agenda.pageOf
                  .replace('{page}', String(page))
                  .replace('{total}', String(totalPages))}
              </span>
              <button
                onClick={() => setPage(p => Math.min(totalPages, p + 1))}
                disabled={page >= totalPages}
                className="flex items-center gap-1 rounded-lg px-2 py-1 text-xs font-medium text-neutral-600 transition-colors hover:bg-neutral-200 disabled:opacity-40 disabled:cursor-not-allowed"
              >
                {t.agenda.nextPage}
                <ChevronRight className="h-3.5 w-3.5" />
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

// ── Botones de acción por estado ──────────────────────────────────────────────

function ActionButtons({
  appt,
  onAction,
  t,
}: {
  appt: Appointment;
  onAction: (id: string, status: string) => void;
  t: ReturnType<typeof useTranslations>;
}) {
  if (appt.status === 'pending') {
    return (
      <div className="flex items-center gap-1">
        <button
          onClick={() => onAction(appt.id, 'confirmed')}
          title={t.agenda.actionConfirm}
          className="rounded p-1 text-emerald-600 transition-colors hover:bg-emerald-50"
        >
          <Check className="h-4 w-4" />
        </button>
        <button
          onClick={() => onAction(appt.id, 'cancelled')}
          title={t.agenda.actionCancel}
          className="rounded p-1 text-red-500 transition-colors hover:bg-red-50"
        >
          <X className="h-4 w-4" />
        </button>
      </div>
    );
  }

  if (appt.status === 'confirmed') {
    return (
      <div className="flex items-center gap-1">
        <button
          onClick={() => onAction(appt.id, 'completed')}
          title={t.agenda.actionComplete}
          className="rounded p-1 text-emerald-600 transition-colors hover:bg-emerald-50"
        >
          <Check className="h-4 w-4" />
        </button>
        <button
          onClick={() => onAction(appt.id, 'no_show')}
          title={t.agenda.actionNoShow}
          className="rounded p-1 text-neutral-500 transition-colors hover:bg-neutral-100"
        >
          <UserX className="h-4 w-4" />
        </button>
        <button
          onClick={() => onAction(appt.id, 'cancelled')}
          title={t.agenda.actionCancel}
          className="rounded p-1 text-red-500 transition-colors hover:bg-red-50"
        >
          <X className="h-4 w-4" />
        </button>
      </div>
    );
  }

  return <span className="text-neutral-300">&mdash;</span>;
}
