'use client';

// Dashboard principal — vista general de operaciones del negocio.

import { useState, useEffect, useRef } from 'react';
import { format } from 'date-fns';
import { formatInTimeZone } from 'date-fns-tz';
import Link from 'next/link';
import { toast } from 'sonner';
import {
  CalendarDays, AlertCircle, ChevronLeft, ChevronRight,
  Wifi, WifiOff, BookOpen, Users, Plus, ArrowUpRight,
  MessageCircle, XCircle, User, Tag, Sparkles,
  Activity, Zap,
} from 'lucide-react';
import {
  appointments, Appointment, whatsapp, knowledge,
  professionals, Professional, KnowledgeDocument,
  APIError, crm, CRMMetrics,
} from '@/lib/api';
import { useTranslations, useDateLocale } from '@/lib/i18n';
import { useTenantStore, useTenantTimezone } from '@/store/tenant';
import { useNotifications } from '@/store/notifications';
import { useAppointmentEvents } from '@/hooks/useAppointmentEvents';
import NewAppointmentModal from '@/components/dashboard/NewAppointmentModal';
import UpcomingAppointmentsCard from '@/components/dashboard/UpcomingAppointmentsCard';
import SoundToggle from '@/components/dashboard/SoundToggle';

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

  const tenant = useTenantStore((s) => s.tenant);
  const tz = useTenantTimezone();
  const [greeting, setGreeting]     = useState('');

  const [crmMetrics, setCrmMetrics] = useState<CRMMetrics | null>(null);
  const [crmLoading, setCrmLoading] = useState(true);

  const [showNewAppt, setShowNewAppt] = useState(false);

  // Real-time: tick fuerza refetch del widget de próximas citas, audioRef toca el sonido.
  const [tick, setTick] = useState(0);
  const audioRef = useRef<HTMLAudioElement>(null);
  const soundEnabled = useNotifications((s) => s.soundEnabled);

  const dateStr = toDateStr(date);
  const isToday = dateStr === toDateStr(new Date());

  // Warm-up del elemento <audio> en la primera interacción para evitar el
  // bloqueo de autoplay de los navegadores: cargamos, reproducimos en volumen 0
  // y volvemos a estado inicial. Después podemos reproducir programáticamente.
  useEffect(() => {
    if (typeof window === 'undefined') return;
    let warmed = false;
    const warm = () => {
      if (warmed || !audioRef.current) return;
      warmed = true;
      audioRef.current.load();
      audioRef.current.volume = 0;
      audioRef.current.play().catch(() => {}).finally(() => {
        if (audioRef.current) {
          audioRef.current.pause();
          audioRef.current.currentTime = 0;
          audioRef.current.volume = 1;
        }
      });
      document.removeEventListener('click', warm);
      document.removeEventListener('keydown', warm);
      document.removeEventListener('touchstart', warm);
    };
    document.addEventListener('click', warm, { once: true });
    document.addEventListener('keydown', warm, { once: true });
    document.addEventListener('touchstart', warm, { once: true, passive: true });
    return () => {
      document.removeEventListener('click', warm);
      document.removeEventListener('keydown', warm);
      document.removeEventListener('touchstart', warm);
    };
  }, []);

  // Suscripción al stream SSE de eventos de citas.
  useAppointmentEvents((envelope) => {
    // Cualquier evento dispara refetch del widget de próximas citas.
    setTick((n) => n + 1);

    // Toast + sonido sólo para creaciones nuevas.
    if (envelope.event === 'appointment.created') {
      const clientName = envelope.data.customer_name ?? 'Cliente';
      toast.success(
        t.dashboard.newAppointmentToast.replace('{clientName}', clientName),
        {
          description: envelope.data.service_name ?? '',
        },
      );
      if (soundEnabled && audioRef.current) {
        audioRef.current.currentTime = 0;
        audioRef.current.play().catch(() => {
          // Autoplay puede bloquear antes del warm-up; fallar silenciosamente.
        });
      }
    }
  });

  // Saludo según la hora del día
  useEffect(() => {
    const h = new Date().getHours();
    if (h < 12) setGreeting(t.greeting.morning);
    else if (h < 19) setGreeting(t.greeting.afternoon);
    else setGreeting(t.greeting.evening);
  }, [t]);

  useEffect(() => {
    let cancelled = false;
    setApptLoading(true);
    setApptError('');
    appointments.list(dateStr, tz)
      .then((res) => { if (!cancelled) setAppts(res.data ?? []); })
      .catch((err) => {
        if (!cancelled) setApptError(
          err instanceof APIError ? err.message : t.dashboard.errorLoadingAppts
        );
      })
      .finally(() => { if (!cancelled) setApptLoading(false); });
    return () => { cancelled = true; };
  }, [dateStr, t, tz]);

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

  useEffect(() => {
    crm.getMetrics()
      .then(setCrmMetrics)
      .catch(() => {})
      .finally(() => setCrmLoading(false));
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
      await appointments.updateStatus(appt.id, { status: next });
      setAppts((prev) =>
        prev.map((a) => (a.id === appt.id ? { ...a, status: next } : a))
      );
    } catch { /* ignorar */ }
    finally { setUpdating(null); }
  }

  const pending = appts.filter((a) => a.status === 'pending').length;

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
            <SoundToggle />
            <Link
              href="/dashboard/whatsapp"
              className="flex items-center gap-1.5 rounded-lg bg-white/10 px-3 py-2 text-xs font-semibold text-white backdrop-blur-sm transition-all hover:bg-white/20"
            >
              <MessageCircle className="h-3.5 w-3.5" />
              WhatsApp
            </Link>
            <button
              onClick={() => setShowNewAppt(true)}
              className="flex items-center gap-1.5 rounded-lg bg-white px-3 py-2 text-xs font-semibold text-primary-700 transition-colors hover:bg-primary-50"
            >
              <Plus className="h-3.5 w-3.5" />
              {t.dashboard.newAppointment}
            </button>
          </div>
        </div>
      </div>

      <div className="p-6">

        {/* ── KPIs ──────────────────────────────────────────────────────────── */}
        <div className="mb-6 grid grid-cols-3 gap-3">

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
              waStatus !== 'CONNECTED' ? (
                <div className="relative flex flex-col items-center overflow-hidden rounded-xl border border-neutral-200 bg-gradient-to-br from-white via-primary-50/40 to-sky-50/40 px-6 py-16 text-center shadow-sm">
                  <div className="pointer-events-none absolute -right-12 -top-12 h-40 w-40 rounded-full bg-primary-200/30 blur-3xl" />
                  <div className="pointer-events-none absolute -bottom-10 -left-10 h-36 w-36 rounded-full bg-emerald-200/30 blur-3xl" />

                  <div className="relative mb-4 flex h-16 w-16 items-center justify-center rounded-2xl bg-gradient-to-br from-primary-500 to-emerald-500 shadow-lg shadow-primary-500/30">
                    <MessageCircle className="h-8 w-8 text-white" />
                  </div>
                  <h3 className="relative max-w-sm text-base font-bold text-neutral-900">
                    {t.dashboard.emptyConnectWATitle}
                  </h3>
                  <p className="relative mt-2 max-w-sm text-sm text-neutral-500">
                    {t.dashboard.emptyConnectWADesc}
                  </p>
                  <Link
                    href="/dashboard/whatsapp"
                    className="relative mt-5 inline-flex items-center gap-2 rounded-lg bg-primary-600 px-4 py-2.5 text-sm font-semibold text-white shadow-sm shadow-primary-500/20 transition-all hover:bg-primary-500 hover:shadow-md"
                  >
                    <MessageCircle className="h-4 w-4" />
                    {t.dashboard.emptyConnectWACta}
                    <ArrowUpRight className="h-4 w-4" />
                  </Link>
                  {tenant?.slug && (
                    <Link
                      href={`/book/${tenant.slug}`}
                      target="_blank"
                      rel="noreferrer"
                      className="relative mt-3 text-xs font-medium text-neutral-500 transition-colors hover:text-primary-600"
                    >
                      {t.dashboard.emptyShareBookingLink}
                    </Link>
                  )}
                </div>
              ) : (
                <div className="flex flex-col items-center rounded-xl border border-neutral-200 bg-gradient-to-br from-white to-primary-50/30 px-6 py-16 text-center shadow-sm">
                  <div className="mb-3 flex h-12 w-12 items-center justify-center rounded-xl bg-primary-50">
                    <CalendarDays className="h-6 w-6 text-primary-500" />
                  </div>
                  <p className="text-sm font-semibold text-neutral-700">{t.dashboard.noAppointmentsDay}</p>
                  <p className="mt-1 max-w-xs text-xs text-neutral-500">{t.dashboard.noAppointmentsDesc}</p>
                </div>
              )
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
                          {formatInTimeZone(appt.starts_at, tz, 'HH:mm')}
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

            {/* Próximas citas — refetcha en vivo vía SSE */}
            <UpcomingAppointmentsCard refreshKey={tick} />

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

            {/* Widget CRM */}
            <div className="rounded-xl border border-neutral-200 bg-white p-4 shadow-sm">
              <div className="mb-3 flex items-center justify-between">
                <h3 className="text-sm font-semibold text-neutral-900">{t.dashboard.crmTitle}</h3>
                <Activity className="h-4 w-4 text-neutral-400" />
              </div>

              {crmLoading ? (
                <div className="space-y-2">
                  <div className="h-8 w-16 animate-pulse rounded-lg bg-neutral-100" />
                  <div className="h-4 w-28 animate-pulse rounded bg-neutral-100" />
                </div>
              ) : !crmMetrics ? (
                <p className="text-xs text-neutral-400">{t.dashboard.noMetricsYet}</p>
              ) : (
                <>
                  {/* KPIs en grid 2x2 */}
                  <div className="grid grid-cols-2 gap-2">
                    <div className="rounded-lg bg-primary-50 p-2.5">
                      <p className="text-xs text-primary-600">{t.dashboard.activeTreatments}</p>
                      <p className="text-lg font-black text-primary-700">{crmMetrics.active_treatments}</p>
                    </div>
                    <div className="rounded-lg bg-amber-50 p-2.5">
                      <p className="text-xs text-amber-600">{t.dashboard.pendingTasks}</p>
                      <p className="text-lg font-black text-amber-700">{crmMetrics.pending_tasks}</p>
                    </div>
                    <div className="rounded-lg bg-sky-50 p-2.5">
                      <p className="text-xs text-sky-600">{t.dashboard.rulesFired}</p>
                      <p className="text-lg font-black text-sky-700">{crmMetrics.rules_fired_30d}</p>
                    </div>
                    <div className="rounded-lg bg-emerald-50 p-2.5">
                      <p className="text-xs text-emerald-600">{t.dashboard.successRate}</p>
                      <p className="text-lg font-black text-emerald-700">
                        {Math.round(crmMetrics.rules_success_rate * 100)}%
                      </p>
                    </div>
                  </div>

                  {/* Pipeline distribution - simple horizontal bar */}
                  {crmMetrics.customers_per_stage.length > 0 && (
                    <div className="mt-3">
                      <p className="mb-2 text-xs font-medium text-neutral-500">{t.dashboard.pipelineDistribution}</p>
                      <div className="space-y-1.5">
                        {crmMetrics.customers_per_stage.map((stage) => {
                          const maxCount = Math.max(...crmMetrics.customers_per_stage.map(s => s.count), 1);
                          const pct = (stage.count / maxCount) * 100;
                          return (
                            <div key={stage.stage_id} className="flex items-center gap-2">
                              <span className="w-24 truncate text-xs text-neutral-600">{stage.stage_name}</span>
                              <div className="flex-1">
                                <div className="h-2 overflow-hidden rounded-full bg-neutral-100">
                                  <div
                                    className="h-full rounded-full transition-all"
                                    style={{ width: `${pct}%`, backgroundColor: stage.color }}
                                  />
                                </div>
                              </div>
                              <span className="w-6 text-right text-xs font-semibold text-neutral-700">{stage.count}</span>
                            </div>
                          );
                        })}
                      </div>
                    </div>
                  )}

                  {/* Top rules */}
                  {crmMetrics.top_rules.length > 0 && (
                    <div className="mt-3 border-t border-neutral-100 pt-3">
                      <div className="space-y-1.5">
                        {crmMetrics.top_rules.slice(0, 3).map((rule) => (
                          <div key={rule.rule_id} className="flex items-center justify-between">
                            <span className="flex items-center gap-1.5 text-xs text-neutral-600">
                              <Zap className="h-3 w-3 text-amber-500" />
                              {rule.rule_name}
                            </span>
                            <span className="text-xs font-semibold text-neutral-500">{rule.executions}x</span>
                          </div>
                        ))}
                      </div>
                    </div>
                  )}
                </>
              )}

              <Link
                href="/dashboard/pipeline"
                className="mt-3 flex items-center justify-between rounded-lg border border-neutral-200 px-3 py-2 text-xs font-medium text-neutral-600 transition-colors hover:border-primary-200 hover:bg-primary-50 hover:text-primary-700"
              >
                {t.dashboard.viewCRM}
                <ArrowUpRight className="h-3.5 w-3.5" />
              </Link>
            </div>
          </div>
        </div>
      </div>

      <NewAppointmentModal
        open={showNewAppt}
        onClose={() => setShowNewAppt(false)}
        onCreated={() => {
          // Recargar citas del dia actual
          appointments.list(dateStr, tz)
            .then((res) => setAppts(res.data ?? []))
            .catch(() => {});
        }}
        defaultDate={dateStr}
      />

      {/* Elemento de audio oculto para notificaciones sonoras de nuevas citas. */}
      <audio
        ref={audioRef}
        preload="auto"
        src="/sounds/notification.mp3"
        style={{ display: 'none' }}
      />
    </div>
  );
}
