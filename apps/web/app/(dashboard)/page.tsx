'use client';

// Dashboard principal — vista general de operaciones del negocio.

import { useState, useEffect } from 'react';
import { format } from 'date-fns';
import Link from 'next/link';
import {
  CalendarDays, AlertCircle, ChevronLeft, ChevronRight,
  Wifi, WifiOff, BookOpen, Users, Plus, ArrowUpRight,
  MessageCircle, XCircle, User, Tag, Sparkles,
} from 'lucide-react';
import {
  appointments, Appointment, whatsapp, knowledge,
  professionals, Professional, KnowledgeDocument,
  auth, TenantDTO, APIError,
} from '@/lib/api';
import { useTranslations, useDateLocale } from '@/lib/i18n';

// ── Helpers ───────────────────────────────────────────────────────────────────

function toDateStr(d: Date): string {
  return d.toISOString().split('T')[0];
}

function addDays(d: Date, n: number): Date {
  const r = new Date(d);
  r.setDate(r.getDate() + n);
  return r;
}

// ── Sub-componentes ───────────────────────────────────────────────────────────

function KPISkeleton() {
  return <div className="h-9 w-16 animate-pulse rounded-lg bg-neutral-100" />;
}

function RowSkeleton() {
  return <div className="h-[72px] animate-pulse rounded-xl border border-neutral-200 bg-white" />;
}

function WAIndicator({ status }: { status?: 'CONNECTED' | 'DISCONNECTED' | 'CONNECTING' }) {
  if (status === 'CONNECTED') {
    return (
      <span className="relative flex h-2 w-2 flex-shrink-0">
        <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-emerald-400 opacity-75" />
        <span className="relative inline-flex h-2 w-2 rounded-full bg-emerald-500" />
      </span>
    );
  }
  if (status === 'CONNECTING') {
    return (
      <span className="relative flex h-2 w-2 flex-shrink-0">
        <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-amber-400 opacity-75" />
        <span className="relative inline-flex h-2 w-2 rounded-full bg-amber-500" />
      </span>
    );
  }
  return <span className="h-2 w-2 rounded-full bg-neutral-300" />;
}

// ── Página principal ──────────────────────────────────────────────────────────

export default function DashboardPage() {
  const t = useTranslations();
  const dateLocale = useDateLocale();

  const [date, setDate]             = useState(() => new Date());
  const [appts, setAppts]           = useState<Appointment[]>([]);
  const [apptLoading, setApptLoading] = useState(true);
  const [apptError, setApptError]   = useState('');
  const [updating, setUpdating]     = useState<string | null>(null);

  const [waStatus, setWAStatus]     = useState<'CONNECTED' | 'DISCONNECTED' | 'CONNECTING'>();
  const [waLoading, setWALoading]   = useState(true);

  const [docs, setDocs]             = useState<KnowledgeDocument[]>([]);
  const [docsLoading, setDocsLoading] = useState(true);
  const [profList, setProfList]     = useState<Professional[]>([]);

  const [tenant, setTenant]         = useState<TenantDTO | null>(null);
  const [greeting, setGreeting]     = useState('');

  const dateStr = toDateStr(date);
  const isToday = dateStr === toDateStr(new Date());

  // Saludo según la hora del día
  useEffect(() => {
    const h = new Date().getHours();
    if (h < 12) setGreeting(t.greeting.morning);
    else if (h < 19) setGreeting(t.greeting.afternoon);
    else setGreeting(t.greeting.evening);
  }, [t]);

  useEffect(() => {
    auth.me().then(({ tenant }) => setTenant(tenant)).catch(() => {});
  }, []);

  useEffect(() => {
    let cancelled = false;
    setApptLoading(true);
    setApptError('');
    appointments.list(dateStr)
      .then((res) => { if (!cancelled) setAppts(res.data ?? []); })
      .catch((err) => {
        if (!cancelled) setApptError(
          err instanceof APIError ? err.message : t.dashboard.errorLoadingAppts
        );
      })
      .finally(() => { if (!cancelled) setApptLoading(false); });
    return () => { cancelled = true; };
  }, [dateStr, t]);

  useEffect(() => {
    setWALoading(true);
    whatsapp.getStatus()
      .then((res) => setWAStatus(res.status))
      .catch(() => setWAStatus('DISCONNECTED'))
      .finally(() => setWALoading(false));

    const interval = setInterval(() => {
      whatsapp.getStatus()
        .then((res) => setWAStatus(res.status))
        .catch(() => setWAStatus('DISCONNECTED'));
    }, 30_000);
    return () => clearInterval(interval);
  }, []);

  useEffect(() => {
    knowledge.list()
      .then((res) => setDocs(res.data ?? []))
      .catch(() => {})
      .finally(() => setDocsLoading(false));
    professionals.list()
      .then((res) => setProfList(res.data ?? []))
      .catch(() => {});
  }, []);

  // Avanzar estado de una cita
  const NEXT_STATUS: Partial<Record<Appointment['status'], Appointment['status']>> = {
    pending:   'confirmed',
    confirmed: 'completed',
  };

  const STATUS_CFG: Record<
    Appointment['status'],
    { label: string; text: string; bg: string; dot: string }
  > = {
    pending:   { label: t.status.pending,   text: 'text-amber-700',   bg: 'bg-amber-50 border-amber-200',      dot: 'bg-amber-400' },
    confirmed: { label: t.status.confirmed, text: 'text-primary-700', bg: 'bg-primary-50 border-primary-200',  dot: 'bg-primary-500' },
    completed: { label: t.status.completed, text: 'text-emerald-700', bg: 'bg-emerald-50 border-emerald-200',  dot: 'bg-emerald-500' },
    cancelled: { label: t.status.cancelled, text: 'text-red-700',     bg: 'bg-red-50 border-red-200',          dot: 'bg-red-400' },
    no_show:   { label: t.status.noShow,    text: 'text-neutral-600', bg: 'bg-neutral-100 border-neutral-200', dot: 'bg-neutral-400' },
  };

  async function advance(appt: Appointment) {
    const next = NEXT_STATUS[appt.status];
    if (!next) return;
    setUpdating(appt.id);
    try {
      const updated = await appointments.updateStatus(appt.id, { status: next });
      setAppts((prev) =>
        prev.map((a) => (a.id === appt.id ? { ...a, ...(updated as Partial<Appointment>) } : a))
      );
    } catch { /* ignorar */ }
    finally { setUpdating(null); }
  }

  const total     = appts.length;
  const pending   = appts.filter((a) => a.status === 'pending').length;
  const confirmed = appts.filter((a) => a.status === 'confirmed').length;
  const completed = appts.filter((a) => a.status === 'completed').length;

  const activeDocs  = docs.filter((d) => d.is_active).length;
  const categories  = Array.from(new Set(docs.map((d) => d.category)));
  const activeProfs = profList.filter((p) => p.is_active);

  const sorted = [...appts].sort(
    (a, b) => new Date(a.starts_at).getTime() - new Date(b.starts_at).getTime()
  );

  const pendingLabel = pending === 0
    ? t.dashboard.allUpToDate
    : pending === 1
    ? t.dashboard.pendingOne
    : t.dashboard.pendingMany.replace('{n}', String(pending));

  return (
    <div className="min-h-full bg-neutral-50">

      {/* ── Header ──────────────────────────────────────────────────────────── */}
      <div className="relative overflow-hidden bg-gradient-to-br from-primary-700 via-primary-600 to-primary-800 px-6 py-6">
        <div className="pointer-events-none absolute -right-16 -top-16 h-56 w-56 rounded-full bg-white/5 blur-3xl" />
        <div className="pointer-events-none absolute -bottom-10 right-40 h-40 w-40 rounded-full bg-accent-500/10 blur-2xl" />
        <div className="pointer-events-none absolute left-1/2 top-0 h-px w-full -translate-x-1/2 bg-gradient-to-r from-transparent via-white/10 to-transparent" />

        <div className="relative flex items-end justify-between">
          <div>
            <p className="flex items-center gap-1.5 text-xs font-medium uppercase tracking-widest text-primary-200">
              <Sparkles className="h-3 w-3" />
              {greeting}
            </p>
            <h1 className="mt-1 text-2xl font-black tracking-tight text-white">
              {tenant?.name ?? 'CitaSpot'}
            </h1>
            <p className="mt-0.5 text-sm capitalize text-primary-200">
              {format(new Date(), "EEEE d 'de' MMMM, yyyy", { locale: dateLocale })}
            </p>
          </div>

          <div className="flex items-center gap-2">
            <Link
              href="/dashboard/whatsapp"
              className="flex items-center gap-1.5 rounded-lg bg-white/10 px-3 py-2 text-xs font-semibold text-white backdrop-blur-sm transition-all hover:bg-white/20"
            >
              <MessageCircle className="h-3.5 w-3.5" />
              WhatsApp
            </Link>
            <button className="flex items-center gap-1.5 rounded-lg bg-white px-3 py-2 text-xs font-semibold text-primary-700 transition-colors hover:bg-primary-50">
              <Plus className="h-3.5 w-3.5" />
              {t.dashboard.newAppointment}
            </button>
          </div>
        </div>
      </div>

      <div className="p-6">

        {/* ── KPIs ──────────────────────────────────────────────────────────── */}
        <div className="mb-6 grid grid-cols-4 gap-3">

          {/* Citas del día */}
          <div className="rounded-xl border border-neutral-200 bg-white p-4 shadow-sm">
            <div className="mb-2 flex items-center justify-between">
              <span className="text-xs font-medium text-neutral-500">
                {t.dashboard.appointmentsCount} {isToday ? t.common.today.toLowerCase() : format(date, 'd MMM', { locale: dateLocale })}
              </span>
              <span className="flex h-7 w-7 items-center justify-center rounded-lg bg-primary-50">
                <CalendarDays className="h-4 w-4 text-primary-600" />
              </span>
            </div>
            {apptLoading ? <KPISkeleton /> : (
              <p className="text-4xl font-black text-primary-600">{total}</p>
            )}
            {!apptLoading && total > 0 && (
              <div className="mt-2 flex gap-3 text-xs text-neutral-500">
                <span className="flex items-center gap-1">
                  <span className="h-1.5 w-1.5 rounded-full bg-amber-400" />
                  {pending} pend.
                </span>
                <span className="flex items-center gap-1">
                  <span className="h-1.5 w-1.5 rounded-full bg-primary-500" />
                  {confirmed} conf.
                </span>
                <span className="flex items-center gap-1">
                  <span className="h-1.5 w-1.5 rounded-full bg-emerald-500" />
                  {completed} comp.
                </span>
              </div>
            )}
          </div>

          {/* Requieren acción */}
          <div className="rounded-xl border border-neutral-200 bg-white p-4 shadow-sm">
            <div className="mb-2 flex items-center justify-between">
              <span className="text-xs font-medium text-neutral-500">{t.dashboard.needsAction}</span>
              <span className="flex h-7 w-7 items-center justify-center rounded-lg bg-amber-50">
                <AlertCircle className="h-4 w-4 text-amber-600" />
              </span>
            </div>
            {apptLoading ? <KPISkeleton /> : (
              <p className={`text-4xl font-black ${pending > 0 ? 'text-amber-600' : 'text-neutral-200'}`}>
                {pending}
              </p>
            )}
            <p className="mt-2 text-xs text-neutral-500">{pendingLabel}</p>
          </div>

          {/* Estado de WhatsApp */}
          <div className="rounded-xl border border-neutral-200 bg-white p-4 shadow-sm">
            <div className="mb-2 flex items-center justify-between">
              <span className="text-xs font-medium text-neutral-500">WhatsApp</span>
              <span className={`flex h-7 w-7 items-center justify-center rounded-lg ${
                waStatus === 'CONNECTED' ? 'bg-emerald-50' : 'bg-neutral-100'
              }`}>
                {waStatus === 'CONNECTED'
                  ? <Wifi className="h-4 w-4 text-emerald-600" />
                  : <WifiOff className="h-4 w-4 text-neutral-400" />
                }
              </span>
            </div>
            {waLoading ? <KPISkeleton /> : (
              <div className="flex items-center gap-2">
                <WAIndicator status={waStatus} />
                <p className={`text-lg font-black leading-none ${
                  waStatus === 'CONNECTED'  ? 'text-emerald-600' :
                  waStatus === 'CONNECTING' ? 'text-amber-600'   : 'text-neutral-400'
                }`}>
                  {waStatus === 'CONNECTED'  ? t.dashboard.waActive
                   : waStatus === 'CONNECTING' ? t.dashboard.waConnecting
                   : t.dashboard.waInactive}
                </p>
              </div>
            )}
            <Link
              href="/dashboard/whatsapp"
              className="mt-2 block text-xs text-primary-600 transition-colors hover:text-primary-700"
            >
              {waStatus !== 'CONNECTED' ? t.dashboard.connectWALink : t.dashboard.viewSessionLink}
            </Link>
          </div>

          {/* Base de conocimiento */}
          <div className="rounded-xl border border-neutral-200 bg-white p-4 shadow-sm">
            <div className="mb-2 flex items-center justify-between">
              <span className="text-xs font-medium text-neutral-500">{t.dashboard.aiKnowledge}</span>
              <span className="flex h-7 w-7 items-center justify-center rounded-lg bg-sky-50">
                <BookOpen className="h-4 w-4 text-sky-600" />
              </span>
            </div>
            {docsLoading ? <KPISkeleton /> : (
              <p className="text-4xl font-black text-sky-600">{activeDocs}</p>
            )}
            <p className="mt-2 text-xs text-neutral-500">
              {docsLoading ? '—'
               : categories.length > 0
               ? t.dashboard.activeCategories.replace('{n}', String(categories.length))
               : t.dashboard.noDocumentsYet}
            </p>
          </div>
        </div>

        {/* ── Grid: agenda (izq) + widgets (der) ───────────────────────────── */}
        <div className="grid grid-cols-5 gap-4">

          {/* Agenda */}
          <div className="col-span-3 space-y-3">
            <div className="flex items-center justify-between">
              <h2 className="text-sm font-bold text-neutral-900">
                {isToday ? t.dashboard.agendaToday : t.dashboard.agendaDate.replace('{date}', format(date, "d 'de' MMMM", { locale: dateLocale }))}
              </h2>
              <div className="flex items-center gap-1">
                {!isToday && (
                  <button
                    onClick={() => setDate(new Date())}
                    className="rounded-lg px-2 py-1 text-xs font-medium text-primary-600 transition-colors hover:bg-primary-50"
                  >
                    {t.common.today}
                  </button>
                )}
                <button
                  onClick={() => setDate((d) => addDays(d, -1))}
                  className="rounded-lg p-1.5 text-neutral-400 transition-colors hover:bg-neutral-200 hover:text-neutral-700"
                >
                  <ChevronLeft className="h-4 w-4" />
                </button>
                <button
                  onClick={() => setDate((d) => addDays(d, 1))}
                  className="rounded-lg p-1.5 text-neutral-400 transition-colors hover:bg-neutral-200 hover:text-neutral-700"
                >
                  <ChevronRight className="h-4 w-4" />
                </button>
              </div>
            </div>

            {apptLoading ? (
              <div className="space-y-2">
                {[...Array(4)].map((_, i) => <RowSkeleton key={i} />)}
              </div>
            ) : apptError ? (
              <div className="flex flex-col items-center rounded-xl border border-neutral-200 bg-white py-12 text-center shadow-sm">
                <XCircle className="mb-2 h-8 w-8 text-red-400" />
                <p className="text-sm text-neutral-500">{apptError}</p>
              </div>
            ) : sorted.length === 0 ? (
              <div className="flex flex-col items-center rounded-xl border border-dashed border-neutral-200 bg-white py-16 text-center">
                <CalendarDays className="mb-3 h-10 w-10 text-neutral-200" />
                <p className="text-sm font-semibold text-neutral-600">{t.dashboard.noAppointmentsDay}</p>
                <p className="mt-1 text-xs text-neutral-400">{t.dashboard.noAppointmentsDesc}</p>
              </div>
            ) : (
              <div className="space-y-2">
                {sorted.map((appt) => {
                  const cfg   = STATUS_CFG[appt.status];
                  const next  = NEXT_STATUS[appt.status];
                  const isUpd = updating === appt.id;

                  return (
                    <div
                      key={appt.id}
                      className="group flex items-center gap-3 rounded-xl border border-neutral-200 bg-white px-4 py-3 shadow-sm transition-all hover:border-primary-200 hover:shadow-md"
                    >
                      <div className="w-14 flex-shrink-0 text-center">
                        <p className="text-sm font-bold text-neutral-900">
                          {format(new Date(appt.starts_at), 'HH:mm')}
                        </p>
                        <p className="text-xs text-neutral-400">{appt.service_duration_min}{t.common.min}</p>
                      </div>
                      <div className="h-10 w-1 flex-shrink-0 rounded-full bg-primary-400 opacity-60" />
                      <div className="min-w-0 flex-1">
                        <p className="truncate text-sm font-semibold text-neutral-900">{appt.customer_name}</p>
                        <div className="mt-0.5 flex items-center gap-2 text-xs text-neutral-500">
                          <span className="flex items-center gap-1">
                            <User className="h-3 w-3" />
                            {appt.professional_name}
                          </span>
                          <span>·</span>
                          <span>{appt.service_name}</span>
                        </div>
                      </div>
                      <span className={`flex-shrink-0 rounded-full border px-2.5 py-0.5 text-xs font-medium ${cfg.bg} ${cfg.text}`}>
                        {cfg.label}
                      </span>
                      {next && (
                        <button
                          onClick={() => advance(appt)}
                          disabled={isUpd}
                          className="flex-shrink-0 rounded-lg bg-primary-600 px-3 py-1.5 text-xs font-semibold text-white opacity-0 transition-all group-hover:opacity-100 hover:bg-primary-500 disabled:opacity-50"
                        >
                          {isUpd ? '…' : next === 'confirmed' ? t.dashboard.confirm : t.dashboard.complete}
                        </button>
                      )}
                    </div>
                  );
                })}
              </div>
            )}
          </div>

          {/* Widgets */}
          <div className="col-span-2 space-y-3">

            {/* Widget WhatsApp */}
            <div className="rounded-xl border border-neutral-200 bg-white p-4 shadow-sm">
              <div className="mb-3 flex items-center justify-between">
                <h3 className="text-sm font-semibold text-neutral-900">WhatsApp</h3>
                <MessageCircle className="h-4 w-4 text-neutral-400" />
              </div>
              <div className={`inline-flex items-center gap-2 rounded-full border px-3 py-1.5 ${
                waStatus === 'CONNECTED'  ? 'border-emerald-200 bg-emerald-50' :
                waStatus === 'CONNECTING' ? 'border-amber-200 bg-amber-50'   :
                'border-neutral-200 bg-neutral-50'
              }`}>
                <WAIndicator status={waStatus} />
                <span className={`text-xs font-semibold ${
                  waStatus === 'CONNECTED'  ? 'text-emerald-700' :
                  waStatus === 'CONNECTING' ? 'text-amber-700'   : 'text-neutral-500'
                }`}>
                  {waLoading ? t.dashboard.waVerifying
                   : waStatus === 'CONNECTED'  ? t.dashboard.waConnectedStatus
                   : waStatus === 'CONNECTING' ? t.dashboard.waConnecting
                   : t.dashboard.waDisconnectedStatus}
                </span>
              </div>
              <p className="mt-2 text-xs text-neutral-500">
                {waStatus === 'CONNECTED' ? t.dashboard.waAIActive : t.dashboard.waConnectForAI}
              </p>
              <Link
                href="/dashboard/whatsapp"
                className="mt-3 flex items-center justify-between rounded-lg border border-neutral-200 px-3 py-2 text-xs font-medium text-neutral-600 transition-colors hover:border-primary-200 hover:bg-primary-50 hover:text-primary-700"
              >
                {waStatus === 'CONNECTED' ? t.dashboard.manageConnection : t.dashboard.waConnectLink}
                <ArrowUpRight className="h-3.5 w-3.5" />
              </Link>
            </div>

            {/* Widget conocimiento */}
            <div className="rounded-xl border border-neutral-200 bg-white p-4 shadow-sm">
              <div className="mb-3 flex items-center justify-between">
                <h3 className="text-sm font-semibold text-neutral-900">{t.dashboard.knowledgeTitle}</h3>
                <BookOpen className="h-4 w-4 text-neutral-400" />
              </div>
              {docsLoading ? (
                <div className="space-y-2">
                  <div className="h-8 w-16 animate-pulse rounded-lg bg-neutral-100" />
                  <div className="h-4 w-28 animate-pulse rounded bg-neutral-100" />
                </div>
              ) : (
                <>
                  <div className="flex items-baseline gap-2">
                    <p className="text-3xl font-black text-sky-600">{activeDocs}</p>
                    <p className="text-xs text-neutral-500">{t.dashboard.activeDocs}</p>
                  </div>
                  {categories.length > 0 ? (
                    <div className="mt-3 flex flex-wrap gap-1.5">
                      {categories.slice(0, 5).map((cat) => (
                        <span key={cat} className="flex items-center gap-1 rounded-full border border-sky-100 bg-sky-50 px-2 py-0.5 text-xs font-medium text-sky-700">
                          <Tag className="h-2.5 w-2.5" />{cat}
                        </span>
                      ))}
                      {categories.length > 5 && (
                        <span className="rounded-full border border-neutral-200 px-2 py-0.5 text-xs text-neutral-400">
                          {t.common.more.replace('{n}', String(categories.length - 5))}
                        </span>
                      )}
                    </div>
                  ) : (
                    <p className="mt-2 text-xs text-neutral-400">{t.dashboard.addDocsForAI}</p>
                  )}
                </>
              )}
              <Link
                href="/dashboard/knowledge"
                className="mt-3 flex items-center justify-between rounded-lg border border-neutral-200 px-3 py-2 text-xs font-medium text-neutral-600 transition-colors hover:border-sky-200 hover:bg-sky-50 hover:text-sky-700"
              >
                {t.dashboard.manageDocuments}
                <ArrowUpRight className="h-3.5 w-3.5" />
              </Link>
            </div>

            {/* Widget equipo */}
            <div className="rounded-xl border border-neutral-200 bg-white p-4 shadow-sm">
              <div className="mb-3 flex items-center justify-between">
                <h3 className="text-sm font-semibold text-neutral-900">{t.dashboard.activeTeam}</h3>
                <span className="flex items-center gap-1 rounded-full border border-neutral-200 px-2 py-0.5 text-xs font-medium text-neutral-600">
                  <Users className="h-3 w-3" />{activeProfs.length}
                </span>
              </div>
              {activeProfs.length === 0 ? (
                <p className="text-xs text-neutral-400">{t.dashboard.noActiveProfs}</p>
              ) : (
                <div className="space-y-2.5">
                  {activeProfs.slice(0, 4).map((p) => (
                    <div key={p.id} className="flex items-center gap-2.5">
                      <span
                        className="flex h-7 w-7 flex-shrink-0 items-center justify-center rounded-full text-xs font-bold text-white shadow-sm"
                        style={{ backgroundColor: p.color || '#8b5cf6' }}
                      >
                        {p.name[0].toUpperCase()}
                      </span>
                      <div className="min-w-0">
                        <p className="truncate text-xs font-semibold text-neutral-800">{p.name}</p>
                        {p.specialty && <p className="truncate text-xs text-neutral-400">{p.specialty}</p>}
                      </div>
                    </div>
                  ))}
                  {activeProfs.length > 4 && (
                    <p className="text-xs text-neutral-400">
                      {t.common.more.replace('{n}', String(activeProfs.length - 4))}
                    </p>
                  )}
                </div>
              )}
              <Link
                href="/dashboard/team"
                className="mt-3 flex items-center justify-between rounded-lg border border-neutral-200 px-3 py-2 text-xs font-medium text-neutral-600 transition-colors hover:border-primary-200 hover:bg-primary-50 hover:text-primary-700"
              >
                {t.dashboard.viewTeam}
                <ArrowUpRight className="h-3.5 w-3.5" />
              </Link>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
