'use client';

// Página de Agenda — visualización de citas en vistas Día, Semana y Mes.
// Soporta: navegación de fecha, solapamientos en columnas, franjas de disponibilidad.

import { useState, useEffect, useRef, useCallback, useMemo } from 'react';
import {
  format, addDays, addWeeks, addMonths, subDays, subWeeks, subMonths,
  startOfMonth, endOfMonth, startOfWeek, endOfWeek, eachDayOfInterval,
  isSameDay, isToday as isTodayFn, isSameMonth,
} from 'date-fns';
import { formatInTimeZone } from 'date-fns-tz';
import {
  ChevronLeft, ChevronRight, Plus, AlignLeft, CalendarDays, LayoutGrid, List,
} from 'lucide-react';
import AppointmentsList from '@/components/dashboard/appointments-list';
import { toast } from 'sonner';
import {
  appointments, Appointment, APIError,
  professionals, Professional,
  services, Service,
  TimeSlot,
} from '@/lib/api';
import { useTenantTimezone } from '@/store/tenant';
import {
  START_HOUR, END_HOUR, HOUR_PX, TOTAL_HOURS, GRID_PX,
  toDateStr, topPx, heightPx, assignColumns, nowPx,
  gridYToIsoUTC, addMinutesIso, diffMinutesIso, snapMinutes,
} from '@/lib/calendar-utils';
import { useTranslations, useDateLocale } from '@/lib/i18n';
import NewAppointmentModal from '@/components/dashboard/NewAppointmentModal';
import AppointmentDetailModal from '@/components/dashboard/AppointmentDetailModal';
import { ToggleSwitch } from '@/components/ui/toggle-switch';

// ── Tipos ──────────────────────────────────────────────────────────────────────

type CalView = 'day' | 'week' | 'month' | 'lista';
type DayState = Appointment[] | 'loading' | 'error';

interface ApptWithCol extends Appointment {
  col: number;
  span: number;
}

// etiquetas de hora: de 06:00 a 23:00 inclusive
const HOUR_LABELS = Array.from({ length: TOTAL_HOURS + 1 }, (_, i) => i + START_HOUR);

// ── Status visual ──────────────────────────────────────────────────────────────

const STATUS_CFG: Record<
  Appointment['status'],
  { text: string; bg: string; border: string }
> = {
  pending:   { text: 'text-amber-800',   bg: 'bg-amber-50',    border: 'border-amber-300'   },
  confirmed: { text: 'text-primary-800', bg: 'bg-primary-50',  border: 'border-primary-300' },
  completed: { text: 'text-emerald-800', bg: 'bg-emerald-50',  border: 'border-emerald-300' },
  cancelled: { text: 'text-red-700',     bg: 'bg-red-50',      border: 'border-red-200'     },
  no_show:   { text: 'text-neutral-600', bg: 'bg-neutral-100', border: 'border-neutral-300' },
};

// ── MIME type del drag (HTML5) ────────────────────────────────────────────────

// Usamos un tipo custom para distinguir nuestros drags de cualquier otro.
const DRAG_MIME = 'application/x-citaspot-appt';

interface DragPayload {
  appointmentId:        string;
  // Offset (en minutos) entre el punto donde el usuario "agarró" el bloque y el
  // inicio real del bloque. Necesario para no "saltar" la cita al top del cursor.
  grabOffsetMinutes:    number;
  // Duración original — para preservarla al soltar.
  durationMinutes:      number;
  // Date string del día de origen (YYYY-MM-DD) — debug / drop entre días.
  sourceDateStr:        string;
}

// ── Bloque de cita (en el grid de tiempo) ─────────────────────────────────────

function ApptBlock({
  appt,
  profColor,
  onClick,
  onResizeRequest,
  interactive,
  tz,
}: {
  appt: ApptWithCol;
  profColor?: string;
  onClick?: () => void;
  // Solicitud de resize (cambio de ends_at). El parent decide qué hacer.
  onResizeRequest?: (appt: Appointment, newEndIso: string) => void;
  interactive: boolean;        // habilita drag & resize (solo en day/week)
  tz: string;
}) {
  const top    = topPx(appt.starts_at, tz);
  const height = heightPx(appt.service_duration_min);
  const pct    = 100 / appt.span;
  const color  = profColor ?? '#6b7280';

  // Mientras se hace resize manual marcamos el bloque para feedback visual.
  const [resizing,  setResizing]  = useState(false);
  const [ghostEnds, setGhostEnds] = useState<string | null>(null);

  // ── Drag & drop nativo HTML5 ────────────────────────────────────────────────
  function handleDragStart(e: React.DragEvent<HTMLDivElement>) {
    if (!interactive) return;
    const rect       = e.currentTarget.getBoundingClientRect();
    const grabY      = e.clientY - rect.top;                 // px desde top del block
    const grabMins   = (grabY / HOUR_PX) * 60;
    const durMins    = diffMinutesIso(appt.starts_at, appt.ends_at) || appt.service_duration_min;
    const payload: DragPayload = {
      appointmentId:     appt.id,
      grabOffsetMinutes: grabMins,
      durationMinutes:   durMins,
      sourceDateStr:     toDateStr(new Date(appt.starts_at)),
    };
    e.dataTransfer.effectAllowed = 'move';
    e.dataTransfer.setData(DRAG_MIME, JSON.stringify(payload));
    // Fallback para navegadores que requieren un tipo "text/plain".
    e.dataTransfer.setData('text/plain', appt.id);
  }

  // ── Resize handle (mousedown en el borde inferior, NO drag HTML5) ──────────
  // Usamos mouse events crudos para evitar conflicto con el drag del bloque
  // y para no disparar el click al soltar.
  function handleResizeStart(e: React.MouseEvent<HTMLDivElement>) {
    if (!interactive || !onResizeRequest) return;
    e.preventDefault();
    e.stopPropagation();

    const startY     = e.clientY;
    const durStart   = diffMinutesIso(appt.starts_at, appt.ends_at) || appt.service_duration_min;
    let   finalEnds  = appt.ends_at;
    setResizing(true);

    function onMove(ev: MouseEvent) {
      const deltaY     = ev.clientY - startY;
      const deltaMins  = (deltaY / HOUR_PX) * 60;
      // Nueva duración con snap a 15min y clamp a >= 15.
      let   newDur     = snapMinutes(durStart + deltaMins, 15);
      if (newDur < 15) newDur = 15;
      finalEnds = addMinutesIso(appt.starts_at, newDur);
      setGhostEnds(finalEnds);
    }
    function onUp() {
      window.removeEventListener('mousemove', onMove);
      window.removeEventListener('mouseup',   onUp);
      setResizing(false);
      setGhostEnds(null);
      if (finalEnds !== appt.ends_at) {
        onResizeRequest!(appt, finalEnds);
      }
    }
    window.addEventListener('mousemove', onMove);
    window.addEventListener('mouseup',   onUp);
  }

  // Altura "fantasma" mientras se hace resize, para feedback en vivo.
  const liveHeight = ghostEnds
    ? Math.max(diffMinutesIso(appt.starts_at, ghostEnds) * (HOUR_PX / 60), 22)
    : height;

  return (
    <div
      className={`absolute z-10 overflow-hidden rounded border border-neutral-200 px-1.5 py-0.5 text-xs transition-shadow hover:z-20 hover:shadow-md ${interactive ? 'cursor-grab active:cursor-grabbing' : 'cursor-pointer'} ${resizing ? 'ring-2 ring-primary-300' : ''}`}
      style={{
        top,
        height:          liveHeight,
        width:           `calc(${pct}% - 4px)`,
        left:            `calc(${(appt.col / appt.span) * 100}% + ${appt.col > 0 ? 2 : 0}px)`,
        minWidth:        0,
        borderLeftWidth: '4px',
        borderLeftColor: color,
        backgroundColor: `${color}14`,
      }}
      title={`${appt.customer_name} · ${appt.service_name} · ${appt.professional_name}`}
      draggable={interactive}
      onDragStart={handleDragStart}
      onClick={onClick}
    >
      <p className="font-semibold leading-tight truncate text-neutral-800 pointer-events-none">
        {formatInTimeZone(appt.starts_at, tz, 'HH:mm')} {appt.customer_name}
      </p>
      {liveHeight >= 38 && (
        <p className="truncate leading-tight text-neutral-500 pointer-events-none">{appt.service_name}</p>
      )}
      {liveHeight >= 54 && (
        <p className="truncate leading-tight font-medium pointer-events-none" style={{ color }}>
          {appt.professional_name}
        </p>
      )}

      {/* Resize handle inferior — solo si es interactivo */}
      {interactive && onResizeRequest && (
        <div
          onMouseDown={handleResizeStart}
          onClick={e => e.stopPropagation()}
          className="absolute inset-x-0 bottom-0 h-1.5 cursor-ns-resize bg-transparent hover:bg-neutral-400/50 transition-colors"
          aria-label="Resize"
        />
      )}
    </div>
  );
}

// ── Franja de disponibilidad ──────────────────────────────────────────────────

function SlotBlock({ slot, tz }: { slot: TimeSlot; tz: string }) {
  const top    = topPx(slot.starts_at, tz);
  const dur    = (new Date(slot.ends_at).getTime() - new Date(slot.starts_at).getTime()) / 60000;
  const height = Math.max(dur * (HOUR_PX / 60), 8);
  return (
    <div
      className="absolute inset-x-0 z-5 mx-1 rounded border border-emerald-300 bg-emerald-50/70 pointer-events-none"
      style={{ top, height }}
    />
  );
}

// ── Columna de etiquetas de hora ──────────────────────────────────────────────

function TimeLabels() {
  return (
    <div className="w-14 flex-shrink-0 select-none border-r border-neutral-100">
      {HOUR_LABELS.map(h => (
        <div
          key={h}
          className="relative flex items-start justify-end pr-2 text-right"
          style={{ height: HOUR_PX }}
        >
          <span className="text-[10px] leading-none text-neutral-400 font-mono -translate-y-1.5">
            {String(h).padStart(2, '0')}:00
          </span>
        </div>
      ))}
    </div>
  );
}

// ── Columna de un día en el grid de tiempo ────────────────────────────────────

function DayColumn({
  date,
  appts,
  slots,
  isLoadingDay,
  profColorMap = {},
  onApptClick,
  onDropAppt,
  onResizeRequest,
  interactive,
  tz,
}: {
  date: Date;
  appts: Appointment[];
  slots: TimeSlot[];
  isLoadingDay: boolean;
  profColorMap?: Record<string, string>;
  onApptClick?: (appt: Appointment) => void;
  // Solicitud de movimiento (drop): el parent recibe el id, el nuevo starts_at y la duración a preservar.
  onDropAppt?: (appointmentId: string, newStartIso: string, durationMinutes: number) => void;
  onResizeRequest?: (appt: Appointment, newEndIso: string) => void;
  interactive: boolean;
  tz: string;
}) {
  const positioned = assignColumns(appts);
  const today      = isTodayFn(date);
  const nowTop     = today ? nowPx(new Date(), tz) : -1;
  const [dragOver, setDragOver] = useState(false);

  // ── Drop handlers HTML5 ────────────────────────────────────────────────────
  function handleDragOver(e: React.DragEvent<HTMLDivElement>) {
    if (!interactive || !onDropAppt) return;
    // preventDefault necesario para permitir el drop.
    if (e.dataTransfer.types.includes(DRAG_MIME) || e.dataTransfer.types.includes('text/plain')) {
      e.preventDefault();
      e.dataTransfer.dropEffect = 'move';
      if (!dragOver) setDragOver(true);
    }
  }

  function handleDragLeave() {
    if (dragOver) setDragOver(false);
  }

  function handleDrop(e: React.DragEvent<HTMLDivElement>) {
    if (!interactive || !onDropAppt) return;
    e.preventDefault();
    setDragOver(false);
    const raw = e.dataTransfer.getData(DRAG_MIME);
    if (!raw) return;
    let payload: DragPayload;
    try { payload = JSON.parse(raw); } catch { return; }

    // Y dentro del column relativo al borde superior del grid.
    const rect = e.currentTarget.getBoundingClientRect();
    const y    = e.clientY - rect.top;
    // Restamos el offset de agarre para que la cita quede donde el usuario "agarró".
    const grabPx = (payload.grabOffsetMinutes / 60) * HOUR_PX;
    const newY   = y - grabPx;

    const newStartIso = gridYToIsoUTC(date, newY, tz, 15);
    onDropAppt(payload.appointmentId, newStartIso, payload.durationMinutes);
  }

  return (
    <div
      className={`relative flex-1 min-w-0 border-l border-neutral-100 ${dragOver ? 'bg-primary-50/40' : ''}`}
      style={{ height: GRID_PX }}
      onDragOver={handleDragOver}
      onDragLeave={handleDragLeave}
      onDrop={handleDrop}
    >
      {HOUR_LABELS.map(h => (
        <div
          key={h}
          className="absolute left-0 right-0 border-t border-neutral-100"
          style={{ top: (h - START_HOUR) * HOUR_PX }}
        />
      ))}
      {HOUR_LABELS.slice(0, -1).map(h => (
        <div
          key={`m${h}`}
          className="absolute left-0 right-0 border-t border-neutral-50"
          style={{ top: (h - START_HOUR) * HOUR_PX + HOUR_PX / 2 }}
        />
      ))}

      {isLoadingDay && (
        <div className="absolute inset-0 flex items-start justify-center pt-20 bg-white/60">
          <div className="h-4 w-4 animate-spin rounded-full border-2 border-primary-300 border-t-primary-600" />
        </div>
      )}

      {slots.map((slot, i) => <SlotBlock key={i} slot={slot} tz={tz} />)}
      {positioned.map(appt => (
        <ApptBlock
          key={appt.id}
          appt={appt}
          profColor={profColorMap[appt.professional_id]}
          onClick={onApptClick ? () => onApptClick(appt) : undefined}
          onResizeRequest={onResizeRequest}
          interactive={interactive}
          tz={tz}
        />
      ))}

      {today && nowTop >= 0 && nowTop <= GRID_PX && (
        <div
          className="absolute left-0 right-0 z-30 flex items-center pointer-events-none"
          style={{ top: nowTop }}
        >
          <div className="h-2.5 w-2.5 flex-shrink-0 rounded-full bg-red-500 -ml-1.5 shadow-sm" />
          <div className="flex-1 h-0.5 bg-red-400" />
        </div>
      )}
    </div>
  );
}

// ── Vista: Día ────────────────────────────────────────────────────────────────

function DayView({
  date,
  dayMap,
  slots,
  profColorMap = {},
  onApptClick,
  onDropAppt,
  onResizeRequest,
  tz,
}: {
  date: Date;
  dayMap: Record<string, DayState>;
  slots: TimeSlot[];
  profColorMap?: Record<string, string>;
  onApptClick?: (appt: Appointment) => void;
  onDropAppt?: (appointmentId: string, newStartIso: string, durationMinutes: number) => void;
  onResizeRequest?: (appt: Appointment, newEndIso: string) => void;
  tz: string;
}) {
  const str     = toDateStr(date);
  const state   = dayMap[str];
  const appts   = Array.isArray(state) ? state : [];
  const loading = state === 'loading';

  return (
    <div className="flex h-full overflow-y-auto">
      <TimeLabels />
      <DayColumn
        date={date}
        appts={appts}
        slots={slots}
        isLoadingDay={loading}
        profColorMap={profColorMap}
        onApptClick={onApptClick}
        onDropAppt={onDropAppt}
        onResizeRequest={onResizeRequest}
        interactive
        tz={tz}
      />
    </div>
  );
}

// ── Vista: Semana ─────────────────────────────────────────────────────────────

function WeekView({
  weekStart,
  dayMap,
  profColorMap = {},
  onApptClick,
  onDropAppt,
  onResizeRequest,
  tz,
}: {
  weekStart: Date;
  dayMap: Record<string, DayState>;
  profColorMap?: Record<string, string>;
  onApptClick?: (appt: Appointment) => void;
  onDropAppt?: (appointmentId: string, newStartIso: string, durationMinutes: number) => void;
  onResizeRequest?: (appt: Appointment, newEndIso: string) => void;
  tz: string;
}) {
  const dateLocale = useDateLocale();
  const days = Array.from({ length: 7 }, (_, i) => addDays(weekStart, i));

  return (
    <div className="flex h-full flex-col">
      {/* Encabezados de día */}
      <div className="flex flex-shrink-0 border-b border-neutral-200 bg-white">
        <div className="w-14 flex-shrink-0 border-r border-neutral-100" />
        {days.map(day => {
          const today = isTodayFn(day);
          return (
            <div
              key={toDateStr(day)}
              className="flex-1 border-l border-neutral-100 px-2 py-2 text-center"
            >
              <p className="text-[10px] uppercase tracking-wider text-neutral-400">
                {format(day, 'EEE', { locale: dateLocale })}
              </p>
              <p className={`mt-0.5 text-sm font-bold ${
                today
                  ? 'inline-flex h-7 w-7 items-center justify-center rounded-full bg-primary-600 text-white'
                  : 'text-neutral-700'
              }`}>
                {format(day, 'd')}
              </p>
            </div>
          );
        })}
      </div>

      {/* Grid de tiempo */}
      <div className="flex flex-1 overflow-y-auto">
        <TimeLabels />
        {days.map(day => {
          const str     = toDateStr(day);
          const state   = dayMap[str];
          const appts   = Array.isArray(state) ? state : [];
          const loading = state === 'loading';
          return (
            <DayColumn
              key={str}
              date={day}
              appts={appts}
              slots={[]}
              isLoadingDay={loading}
              profColorMap={profColorMap}
              onApptClick={onApptClick}
              onDropAppt={onDropAppt}
              onResizeRequest={onResizeRequest}
              interactive
              tz={tz}
            />
          );
        })}
      </div>
    </div>
  );
}

// ── Vista: Mes ────────────────────────────────────────────────────────────────

function MonthView({
  month,
  dayMap,
  onDayClick,
  profColorMap = {},
  tz,
}: {
  month: Date;
  dayMap: Record<string, DayState>;
  onDayClick: (d: Date) => void;
  profColorMap?: Record<string, string>;
  tz: string;
}) {
  const t     = useTranslations();
  const start = startOfWeek(startOfMonth(month), { weekStartsOn: 1 });
  const end   = endOfWeek(endOfMonth(month), { weekStartsOn: 1 });
  const days  = eachDayOfInterval({ start, end });

  return (
    <div className="flex h-full flex-col overflow-auto">
      {/* Encabezados de columna */}
      <div className="grid grid-cols-7 flex-shrink-0 border-b border-neutral-200 bg-white">
        {t.agenda.dowLabels.map((d: string) => (
          <div
            key={d}
            className="py-2 text-center text-[10px] font-semibold uppercase tracking-wider text-neutral-400"
          >
            {d}
          </div>
        ))}
      </div>

      {/* Celdas de días */}
      <div className="grid flex-1 grid-cols-7">
        {days.map(day => {
          const str      = toDateStr(day);
          const state    = dayMap[str];
          const appts    = Array.isArray(state) ? state : [];
          const loading  = state === 'loading';
          const inMonth  = isSameMonth(day, month);
          const today    = isTodayFn(day);

          const pending   = appts.filter(a => a.status === 'pending').length;
          const confirmed = appts.filter(a => a.status === 'confirmed').length;
          const completed = appts.filter(a => a.status === 'completed').length;

          return (
            <div
              key={str}
              onClick={() => onDayClick(day)}
              className={`min-h-[96px] border-b border-r border-neutral-100 p-2 cursor-pointer transition-colors
                ${inMonth ? 'bg-white hover:bg-neutral-50' : 'bg-neutral-50'}
                ${today ? 'ring-1 ring-inset ring-primary-400' : ''}
              `}
            >
              <div className="flex items-center justify-between">
                <span className={`text-sm font-bold leading-none
                  ${!inMonth ? 'text-neutral-300' : ''}
                  ${today
                    ? 'flex h-6 w-6 items-center justify-center rounded-full bg-primary-600 text-white text-xs'
                    : 'text-neutral-700'
                  }
                `}>
                  {format(day, 'd')}
                </span>
                {loading && (
                  <div className="h-3 w-3 animate-spin rounded-full border border-neutral-200 border-t-primary-500" />
                )}
              </div>

              {inMonth && (pending + confirmed + completed) > 0 && (
                <div className="mt-1.5 flex flex-wrap gap-1">
                  {pending > 0 && (
                    <span className="flex items-center gap-0.5 rounded-full bg-amber-100 px-1.5 py-0.5 text-[10px] font-medium text-amber-700">
                      <span className="h-1.5 w-1.5 rounded-full bg-amber-400" />
                      {pending}
                    </span>
                  )}
                  {confirmed > 0 && (
                    <span className="flex items-center gap-0.5 rounded-full bg-primary-100 px-1.5 py-0.5 text-[10px] font-medium text-primary-700">
                      <span className="h-1.5 w-1.5 rounded-full bg-primary-500" />
                      {confirmed}
                    </span>
                  )}
                  {completed > 0 && (
                    <span className="flex items-center gap-0.5 rounded-full bg-emerald-100 px-1.5 py-0.5 text-[10px] font-medium text-emerald-700">
                      <span className="h-1.5 w-1.5 rounded-full bg-emerald-500" />
                      {completed}
                    </span>
                  )}
                </div>
              )}

              {inMonth && appts.length > 0 && (
                <div className="mt-1 space-y-0.5">
                  {appts.slice(0, 2).map(a => {
                    const color = profColorMap[a.professional_id];
                    return (
                      <p
                        key={a.id}
                        className="truncate rounded px-1 py-0.5 text-[10px] text-neutral-700"
                        style={{
                          backgroundColor: color ? `${color}14` : '#f3f4f6',
                          borderLeft:      color ? `2px solid ${color}` : '2px solid #d1d5db',
                        }}
                      >
                        {formatInTimeZone(a.starts_at, tz, 'HH:mm')} {a.customer_name}
                      </p>
                    );
                  })}
                  {appts.length > 2 && (
                    <p className="text-[10px] text-neutral-400">
                      {t.common.more.replace('{n}', String(appts.length - 2))}
                    </p>
                  )}
                </div>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
}

// ── Página principal ──────────────────────────────────────────────────────────

export default function AgendaPage() {
  const t          = useTranslations();
  const dateLocale = useDateLocale();

  const [view, setView]               = useState<CalView>('week');
  const [currentDate, setCurrentDate] = useState(() => new Date());
  const currentDateStr                = useMemo(() => toDateStr(currentDate), [currentDate]);

  const [dayMap, setDayMap]           = useState<Record<string, DayState>>({});
  const dayMapRef                     = useRef<Record<string, DayState>>({});
  useEffect(() => { dayMapRef.current = dayMap; }, [dayMap]);

  const [profList, setProfList]       = useState<Professional[]>([]);
  const [svcList, setSvcList]         = useState<Service[]>([]);
  const [showAvail, setShowAvail]     = useState(false);
  const [selProf, setSelProf]         = useState('');
  const [selSvc, setSelSvc]           = useState('');
  const [slots, setSlots]             = useState<TimeSlot[]>([]);
  const [loadingAvail, setLoadingAvail] = useState(false);
  const [showNewAppt, setShowNewAppt] = useState(false);
  const [detailAppt, setDetailAppt] = useState<Appointment | null>(null);
  const [filterProfIds, setFilterProfIds] = useState<Set<string>>(new Set());
  // Timezone del tenant proviene del store global (hidratado por DashboardLayout).
  const tenantTz = useTenantTimezone();

  // Mapa de color por profesional para el borde izquierdo de los bloques
  const profColorMap = useMemo(() => {
    const map: Record<string, string> = {};
    profList.forEach(p => { map[p.id] = p.color; });
    return map;
  }, [profList]);

  // dayMap filtrado por profesionales seleccionados (vacío = todos)
  const filteredDayMap = useMemo(() => {
    if (filterProfIds.size === 0) return dayMap;
    const filtered: Record<string, DayState> = {};
    for (const [key, val] of Object.entries(dayMap)) {
      if (Array.isArray(val)) {
        filtered[key] = val.filter(a => filterProfIds.has(a.professional_id));
      } else {
        filtered[key] = val;
      }
    }
    return filtered;
  }, [dayMap, filterProfIds]);

  useEffect(() => {
    professionals.list().then(r => setProfList(r.data ?? [])).catch(() => {});
    services.list().then(r => setSvcList((r.data ?? []).filter(s => s.is_active))).catch(() => {});
  }, []);

  const ensureLoaded = useCallback((dates: Date[]) => {
    const toLoad = dates.filter(d => !dayMapRef.current[toDateStr(d)]);
    if (!toLoad.length) return;

    setDayMap(prev => {
      const patch: Record<string, DayState> = {};
      toLoad.forEach(d => { patch[toDateStr(d)] = 'loading'; });
      return { ...prev, ...patch };
    });

    toLoad.forEach(d => {
      const str = toDateStr(d);
      appointments.list(str, tenantTz)
        .then(res => {
          dayMapRef.current[str] = res.data ?? [];
          setDayMap(prev => ({ ...prev, [str]: res.data ?? [] }));
        })
        .catch(() => {
          dayMapRef.current[str] = 'error';
          setDayMap(prev => ({ ...prev, [str]: 'error' }));
        });
    });
  }, [tenantTz]);

  useEffect(() => {
    if (view === 'lista') return;
    if (view === 'day') {
      ensureLoaded([currentDate]);
    } else if (view === 'week') {
      const ws   = startOfWeek(currentDate, { weekStartsOn: 1 });
      const days = Array.from({ length: 7 }, (_, i) => addDays(ws, i));
      ensureLoaded(days);
    } else {
      const monthDays = eachDayOfInterval({
        start: startOfMonth(currentDate),
        end:   endOfMonth(currentDate),
      });
      ensureLoaded(monthDays);
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [view, currentDateStr, ensureLoaded]);

  useEffect(() => {
    if (!showAvail || !selProf || !selSvc || view !== 'day') {
      setSlots([]);
      return;
    }
    setLoadingAvail(true);
    appointments.availability(selProf, selSvc, currentDateStr, tenantTz)
      .then(res => setSlots(res.data ?? []))
      .catch(() => setSlots([]))
      .finally(() => setLoadingAvail(false));
  }, [showAvail, selProf, selSvc, currentDateStr, view]);

  const refreshDays = useCallback((dateStrs: string[]) => {
    dateStrs.forEach(str => {
      dayMapRef.current[str] = undefined as unknown as DayState;
    });
    setDayMap(prev => {
      const next = { ...prev };
      dateStrs.forEach(s => delete next[s]);
      return next;
    });
    dateStrs.forEach(str => ensureLoaded([new Date(str + 'T00:00:00')]));
  }, [ensureLoaded]);

  // ── Reschedule con optimistic update + revert + toast ─────────────────────
  // Aplica el cambio en memoria primero, llama al backend, y revierte si falla.
  // Maneja conflicto (409 / código "slot_unavailable" / "conflict") con toast amable.
  const applyReschedule = useCallback(async (
    appointmentId: string,
    newStartIso:   string,
    newEndIso:     string,
  ) => {
    // Localizar la cita en dayMapRef (los días filtrados son derivados de dayMap)
    let original: Appointment | null = null;
    let oldDateStr = '';
    for (const [str, val] of Object.entries(dayMapRef.current)) {
      if (Array.isArray(val)) {
        const found = val.find(a => a.id === appointmentId);
        if (found) { original = found; oldDateStr = str; break; }
      }
    }
    if (!original) return;

    // No-op si no cambia nada
    if (original.starts_at === newStartIso && original.ends_at === newEndIso) return;

    const newDateStr = toDateStr(new Date(newStartIso));
    const newDurMin  = diffMinutesIso(newStartIso, newEndIso);

    const updated: Appointment = {
      ...original,
      starts_at:            newStartIso,
      ends_at:              newEndIso,
      service_duration_min: newDurMin > 0 ? newDurMin : original.service_duration_min,
    };

    // Snapshot para revertir
    const snapshot: Record<string, DayState> = {};
    const touchedDates = oldDateStr === newDateStr ? [oldDateStr] : [oldDateStr, newDateStr];
    touchedDates.forEach(d => {
      snapshot[d] = dayMapRef.current[d];
    });

    // Optimistic: quitar del día viejo, añadir al nuevo
    setDayMap(prev => {
      const next = { ...prev };
      const oldList = Array.isArray(next[oldDateStr]) ? next[oldDateStr] as Appointment[] : [];
      next[oldDateStr] = oldList.filter(a => a.id !== appointmentId);
      if (oldDateStr === newDateStr) {
        next[newDateStr] = [...(next[oldDateStr] as Appointment[]), updated];
      } else {
        const newList = Array.isArray(next[newDateStr]) ? next[newDateStr] as Appointment[] : [];
        next[newDateStr] = [...newList, updated];
      }
      // Sincronizar el ref para que el próximo cálculo lo encuentre actualizado
      Object.assign(dayMapRef.current, next);
      return next;
    });

    try {
      await appointments.reschedule(appointmentId, {
        starts_at: newStartIso,
        ends_at:   newEndIso,
      });
    } catch (err) {
      // Revertir
      setDayMap(prev => {
        const next = { ...prev };
        touchedDates.forEach(d => { next[d] = snapshot[d]; });
        Object.assign(dayMapRef.current, next);
        return next;
      });
      // Mostrar toast según código de error
      const isConflict = err instanceof APIError && (
        err.status === 409 ||
        err.code   === 'slot_unavailable' ||
        err.code   === 'conflict' ||
        /unavailable|conflict/i.test(err.message)
      );
      if (isConflict) {
        // Toast con acción para abrir el modal de detalle en modo reschedule.
        // Si esto requiriera refactor grande del modal, dejamos solo el toast simple.
        toast.error(t.agenda.rescheduleConflict, {
          action: original ? {
            label: t.agenda.viewAvailableSlots,
            onClick: () => setDetailAppt(original!),
          } : undefined,
        });
      } else {
        toast.error(t.agenda.rescheduleError);
      }
    }
  }, [t]);

  // Handler para drop: preserva la duración del payload (que viene del bloque arrastrado).
  const handleDropAppt = useCallback((
    appointmentId:   string,
    newStartIso:     string,
    durationMinutes: number,
  ) => {
    const newEndIso = addMinutesIso(newStartIso, durationMinutes);
    applyReschedule(appointmentId, newStartIso, newEndIso);
  }, [applyReschedule]);

  // Handler para resize: mantiene starts_at, cambia ends_at.
  const handleResizeRequest = useCallback((appt: Appointment, newEndIso: string) => {
    applyReschedule(appt.id, appt.starts_at, newEndIso);
  }, [applyReschedule]);

  function navigate(dir: 1 | -1) {
    setCurrentDate(prev => {
      if (view === 'day')   return dir > 0 ? addDays(prev, 1)   : subDays(prev, 1);
      if (view === 'week')  return dir > 0 ? addWeeks(prev, 1)  : subWeeks(prev, 1);
      return                       dir > 0 ? addMonths(prev, 1) : subMonths(prev, 1);
    });
  }

  function periodLabel(): string {
    if (view === 'day') {
      return format(currentDate, "EEEE d 'de' MMMM, yyyy", { locale: dateLocale });
    }
    if (view === 'week') {
      const ws = startOfWeek(currentDate, { weekStartsOn: 1 });
      const we = endOfWeek(currentDate, { weekStartsOn: 1 });
      if (ws.getMonth() === we.getMonth()) {
        return `${format(ws, 'd')} – ${format(we, "d 'de' MMMM yyyy", { locale: dateLocale })}`;
      }
      return `${format(ws, 'd MMM', { locale: dateLocale })} – ${format(we, 'd MMM yyyy', { locale: dateLocale })}`;
    }
    return format(currentDate, 'MMMM yyyy', { locale: dateLocale });
  }

  const isThisToday = isTodayFn(currentDate);
  const weekStart   = startOfWeek(currentDate, { weekStartsOn: 1 });

  return (
    <div className="flex h-full flex-col bg-white">

      {/* ── Header principal ───────────────────────────────────────────────── */}
      <div className="flex flex-shrink-0 items-center gap-3 border-b border-neutral-200 bg-white px-5 py-3">

        {/* Navegación de fecha */}
        {view !== 'lista' && (
          <div className="flex items-center gap-1">
            <button
              onClick={() => navigate(-1)}
              className="rounded-lg p-1.5 text-neutral-400 transition-colors hover:bg-neutral-100 hover:text-neutral-700"
            >
              <ChevronLeft className="h-4 w-4" />
            </button>
            {!isThisToday && (
              <button
                onClick={() => setCurrentDate(new Date())}
                className="rounded-lg px-2 py-1 text-xs font-medium text-primary-600 transition-colors hover:bg-primary-50"
              >
                {t.agenda.today}
              </button>
            )}
            <button
              onClick={() => navigate(1)}
              className="rounded-lg p-1.5 text-neutral-400 transition-colors hover:bg-neutral-100 hover:text-neutral-700"
            >
              <ChevronRight className="h-4 w-4" />
            </button>
          </div>
        )}

        {view !== 'lista' && (
          <h2 className="flex-1 text-sm font-semibold capitalize text-neutral-800">
            {periodLabel()}
          </h2>
        )}
        {view === 'lista' && <div className="flex-1" />}

        {/* Switcher de vista */}
        <div className="flex items-center rounded-lg border border-neutral-200 bg-neutral-100 p-0.5">
          {([
            { key: 'day'   as const, label: t.agenda.day,   icon: AlignLeft    },
            { key: 'week'  as const, label: t.agenda.week,  icon: CalendarDays },
            { key: 'month' as const, label: t.agenda.month, icon: LayoutGrid   },
            { key: 'lista' as const, label: t.agenda.list,  icon: List        },
          ]).map(({ key, label, icon: Icon }) => (
            <button
              key={key}
              onClick={() => setView(key)}
              className={`flex items-center gap-1.5 rounded-md px-3 py-1.5 text-xs font-medium transition-all ${
                view === key
                  ? 'bg-white text-neutral-900 shadow-sm'
                  : 'text-neutral-500 hover:text-neutral-700'
              }`}
            >
              <Icon className="h-3.5 w-3.5" />
              {label}
            </button>
          ))}
        </div>

        {/* Nueva cita */}
        <button
          onClick={() => setShowNewAppt(true)}
          className="flex items-center gap-1.5 rounded-lg bg-primary-600 px-3 py-1.5 text-xs font-semibold text-white transition-colors hover:bg-primary-500"
        >
          <Plus className="h-3.5 w-3.5" />
          {t.agenda.newAppointment}
        </button>
      </div>

      {/* ── Filtro por profesional (multi-selección) ─────────────────────── */}
      {profList.length > 1 && (
        <div className="flex flex-shrink-0 items-center gap-2 border-b border-neutral-100 bg-white px-5 py-2 overflow-x-auto">
          <button
            onClick={() => setFilterProfIds(new Set())}
            className={`flex-shrink-0 rounded-full px-3 py-1 text-xs font-medium transition-all ${
              filterProfIds.size === 0
                ? 'bg-neutral-900 text-white'
                : 'bg-neutral-100 text-neutral-600 hover:bg-neutral-200'
            }`}
          >
            {t.agenda.allProfessionals}
          </button>
          {profList.filter(p => p.is_active && !p.is_archived).map(p => {
            const isSelected = filterProfIds.has(p.id);
            return (
              <button
                key={p.id}
                onClick={() => {
                  setFilterProfIds(prev => {
                    const next = new Set(prev);
                    if (next.has(p.id)) next.delete(p.id);
                    else next.add(p.id);
                    return next;
                  });
                }}
                className="flex-shrink-0 rounded-full px-3 py-1 text-xs font-medium transition-all hover:opacity-80"
                style={{
                  backgroundColor: isSelected ? p.color : `${p.color}20`,
                  color:           isSelected ? 'white'  : p.color,
                  border:          isSelected ? 'none'   : `1.5px solid ${p.color}`,
                }}
              >
                {p.name}
              </button>
            );
          })}
        </div>
      )}

      {/* ── Barra de disponibilidad (solo vista Día) ──────────────────────── */}
      {view === 'day' && (
        <div className="flex flex-shrink-0 items-center gap-3 border-b border-neutral-100 bg-neutral-50 px-5 py-2">

          <label className="flex cursor-pointer items-center gap-2 text-xs font-medium text-neutral-600">
            <ToggleSwitch checked={showAvail} onCheckedChange={() => setShowAvail(v => !v)} activeColor="bg-emerald-500" size="sm" />
            {t.agenda.viewAvailability}
          </label>

          {showAvail && (
            <>
              <select
                value={selProf}
                onChange={e => setSelProf(e.target.value)}
                className="rounded-lg border border-neutral-200 bg-white px-2 py-1 text-xs text-neutral-700 focus:outline-none focus:ring-1 focus:ring-primary-400"
              >
                <option value="">{t.agenda.selectProfessional}</option>
                {profList.filter(p => p.is_active).map(p => (
                  <option key={p.id} value={p.id}>{p.name}</option>
                ))}
              </select>

              <select
                value={selSvc}
                onChange={e => setSelSvc(e.target.value)}
                className="rounded-lg border border-neutral-200 bg-white px-2 py-1 text-xs text-neutral-700 focus:outline-none focus:ring-1 focus:ring-primary-400"
              >
                <option value="">{t.agenda.selectService}</option>
                {svcList.map(s => (
                  <option key={s.id} value={s.id}>{s.name} ({s.duration_min}{t.common.min})</option>
                ))}
              </select>

              {loadingAvail ? (
                <div className="h-3.5 w-3.5 animate-spin rounded-full border-2 border-neutral-200 border-t-emerald-500" />
              ) : slots.length > 0 ? (
                <span className="text-xs font-medium text-emerald-600">
                  {t.agenda.availableSlots.replace('{n}', String(slots.length))}
                </span>
              ) : selProf && selSvc ? (
                <span className="text-xs text-neutral-400">{t.agenda.noAvailability}</span>
              ) : null}
            </>
          )}
        </div>
      )}

      {/* ── Contenido del calendario ───────────────────────────────────────── */}
      <div className="min-h-0 flex-1 overflow-hidden">
        {view === 'day' && (
          <DayView
            date={currentDate}
            dayMap={filteredDayMap}
            slots={slots}
            profColorMap={profColorMap}
            onApptClick={setDetailAppt}
            onDropAppt={handleDropAppt}
            onResizeRequest={handleResizeRequest}
            tz={tenantTz}
          />
        )}
        {view === 'week' && (
          <WeekView
            weekStart={weekStart}
            dayMap={filteredDayMap}
            profColorMap={profColorMap}
            onApptClick={setDetailAppt}
            onDropAppt={handleDropAppt}
            onResizeRequest={handleResizeRequest}
            tz={tenantTz}
          />
        )}
        {view === 'month' && (
          <MonthView
            month={currentDate}
            dayMap={filteredDayMap}
            onDayClick={d => { setCurrentDate(d); setView('day'); }}
            profColorMap={profColorMap}
            tz={tenantTz}
          />
        )}
        {view === 'lista' && <AppointmentsList timezone={tenantTz} />}
      </div>

      <NewAppointmentModal
        open={showNewAppt}
        onClose={() => setShowNewAppt(false)}
        onCreated={() => refreshDays([toDateStr(currentDate)])}
        defaultDate={toDateStr(currentDate)}
      />

      <AppointmentDetailModal
        appointment={detailAppt}
        open={detailAppt !== null}
        onClose={() => setDetailAppt(null)}
        onUpdated={refreshDays}
        timezone={tenantTz}
      />
    </div>
  );
}
