'use client';

// Modal de detalle de cita — confirmar, cancelar, reagendar (inline).
// Sigue el mismo patrón visual que NewAppointmentModal.

import { useState, useEffect, useCallback } from 'react';
import { format } from 'date-fns';
import { formatInTimeZone } from 'date-fns-tz';
import { X, Check, Calendar, Clock, User, Scissors, Phone } from 'lucide-react';
import {
  appointments,
  Appointment,
  TimeSlot,
  APIError,
} from '@/lib/api';
import { useTranslations, useDateLocale } from '@/lib/i18n';
import { Spinner } from '@/components/ui/spinner';

interface AppointmentDetailModalProps {
  appointment: Appointment | null;
  open: boolean;
  onClose: () => void;
  onUpdated: (dateStrs: string[]) => void;
  timezone?: string;
}

type Mode = 'view' | 'cancel' | 'reschedule';

const STATUS_CFG: Record<
  Appointment['status'],
  { label: string; bg: string; text: string }
> = {
  pending:   { label: 'pending',   bg: 'bg-amber-100',    text: 'text-amber-700'   },
  confirmed: { label: 'confirmed', bg: 'bg-primary-100',  text: 'text-primary-700' },
  completed: { label: 'completed', bg: 'bg-emerald-100',  text: 'text-emerald-700' },
  cancelled: { label: 'cancelled', bg: 'bg-red-100',      text: 'text-red-700'     },
  no_show:   { label: 'noShow',    bg: 'bg-neutral-200',  text: 'text-neutral-600' },
};

const SOURCE_LABELS: Record<string, string> = {
  whatsapp:  'sourceWhatsapp',
  dashboard: 'sourceDashboard',
  web:       'sourceBooking',
  booking:   'sourceBooking',
  api:       'sourceApi',
};

function toDateStr(d: Date | string): string {
  const date = typeof d === 'string' ? new Date(d) : d;
  return format(date, 'yyyy-MM-dd');
}

export default function AppointmentDetailModal({
  appointment,
  open,
  onClose,
  onUpdated,
  timezone,
}: AppointmentDetailModalProps) {
  // Fallback al timezone del browser si el caller no provee tenant.timezone.
  // Evita renderizar UTC literal cuando el tenant aún no cargó.
  const tz = timezone
    || (typeof window !== 'undefined' ? Intl.DateTimeFormat().resolvedOptions().timeZone : 'UTC');
  const t = useTranslations();
  const dateLocale = useDateLocale();

  const [mode, setMode] = useState<Mode>('view');
  const [cancelReason, setCancelReason] = useState('');
  const [rescheduleDate, setRescheduleDate] = useState('');
  const [rescheduleSlots, setRescheduleSlots] = useState<TimeSlot[]>([]);
  const [selectedSlot, setSelectedSlot] = useState<TimeSlot | null>(null);
  const [loadingSlots, setLoadingSlots] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState(false);

  // Reset al cerrar
  useEffect(() => {
    if (!open) {
      setMode('view');
      setCancelReason('');
      setRescheduleDate('');
      setRescheduleSlots([]);
      setSelectedSlot(null);
      setLoadingSlots(false);
      setSubmitting(false);
      setError('');
      setSuccess(false);
    }
  }, [open]);

  // Cargar slots al cambiar fecha de reagendamiento
  const loadSlots = useCallback(async () => {
    if (!appointment || !rescheduleDate) {
      setRescheduleSlots([]);
      return;
    }
    setLoadingSlots(true);
    setSelectedSlot(null);
    try {
      const res = await appointments.availability(
        appointment.professional_id,
        appointment.service_id,
        rescheduleDate,
      );
      setRescheduleSlots(res.data ?? []);
    } catch {
      setRescheduleSlots([]);
    } finally {
      setLoadingSlots(false);
    }
  }, [appointment, rescheduleDate]);

  useEffect(() => {
    if (mode === 'reschedule' && rescheduleDate) {
      loadSlots();
    }
  }, [mode, rescheduleDate, loadSlots]);

  // Pre-llenar fecha al entrar en modo reschedule
  useEffect(() => {
    if (mode === 'reschedule' && appointment && !rescheduleDate) {
      setRescheduleDate(toDateStr(appointment.starts_at));
    }
  }, [mode, appointment, rescheduleDate]);

  async function handleConfirm() {
    if (!appointment) return;
    setSubmitting(true);
    setError('');
    try {
      await appointments.updateStatus(appointment.id, { status: 'confirmed' });
      setSuccess(true);
      setTimeout(() => {
        onUpdated([toDateStr(appointment.starts_at)]);
        onClose();
      }, 1000);
    } catch (err) {
      setError(err instanceof APIError ? err.message : t.appointmentDetail.error);
    } finally {
      setSubmitting(false);
    }
  }

  async function handleComplete() {
    if (!appointment) return;
    setSubmitting(true);
    setError('');
    try {
      await appointments.updateStatus(appointment.id, { status: 'completed' });
      setSuccess(true);
      setTimeout(() => {
        onUpdated([toDateStr(appointment.starts_at)]);
        onClose();
      }, 1000);
    } catch (err) {
      setError(err instanceof APIError ? err.message : t.appointmentDetail.error);
    } finally {
      setSubmitting(false);
    }
  }

  async function handleNoShow() {
    if (!appointment) return;
    setSubmitting(true);
    setError('');
    try {
      await appointments.updateStatus(appointment.id, { status: 'no_show' });
      setSuccess(true);
      setTimeout(() => {
        onUpdated([toDateStr(appointment.starts_at)]);
        onClose();
      }, 1000);
    } catch (err) {
      setError(err instanceof APIError ? err.message : t.appointmentDetail.error);
    } finally {
      setSubmitting(false);
    }
  }

  async function handleCancel() {
    if (!appointment) return;
    setSubmitting(true);
    setError('');
    try {
      await appointments.cancel(appointment.id, cancelReason || undefined);
      setSuccess(true);
      setTimeout(() => {
        onUpdated([toDateStr(appointment.starts_at)]);
        onClose();
      }, 1000);
    } catch (err) {
      setError(err instanceof APIError ? err.message : t.appointmentDetail.error);
    } finally {
      setSubmitting(false);
    }
  }

  async function handleReschedule() {
    if (!appointment || !selectedSlot) return;
    setSubmitting(true);
    setError('');
    try {
      await appointments.reschedule(appointment.id, {
        starts_at: selectedSlot.starts_at,
        ends_at: selectedSlot.ends_at,
      });
      setSuccess(true);
      const oldDate = toDateStr(appointment.starts_at);
      const newDate = toDateStr(selectedSlot.starts_at);
      const datesToRefresh = oldDate === newDate ? [oldDate] : [oldDate, newDate];
      setTimeout(() => {
        onUpdated(datesToRefresh);
        onClose();
      }, 1000);
    } catch (err) {
      setError(err instanceof APIError ? err.message : t.appointmentDetail.error);
    } finally {
      setSubmitting(false);
    }
  }

  if (!open || !appointment) return null;

  const statusCfg = STATUS_CFG[appointment.status];
  const statusLabel = t.status[statusCfg.label as keyof typeof t.status];
  const sourceKey = SOURCE_LABELS[appointment.source] ?? 'sourceApi';
  const sourceLabel = t.appointmentDetail[sourceKey as keyof typeof t.appointmentDetail] as string;
  const canAct = appointment.status === 'pending' || appointment.status === 'confirmed';

  return (
    <div
      className="fixed inset-0 z-50 flex items-end justify-center bg-black/50 backdrop-blur-sm sm:items-center"
      onClick={(e) => { if (e.target === e.currentTarget) onClose(); }}
    >
      <div className="relative w-full max-w-md rounded-t-2xl bg-white shadow-2xl sm:rounded-2xl">
        {/* Header */}
        <div className="flex items-center justify-between border-b border-neutral-100 px-5 py-4">
          <h2 className="text-base font-bold text-neutral-900">
            {t.appointmentDetail.title}
          </h2>
          <button
            onClick={onClose}
            className="rounded-lg p-1.5 text-neutral-400 transition-colors hover:bg-neutral-100 hover:text-neutral-600"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        {/* Contenido */}
        <div className="max-h-[60vh] overflow-y-auto px-5 py-4">
          {success ? (
            <div className="flex flex-col items-center py-10">
              <div className="mb-3 flex h-14 w-14 items-center justify-center rounded-full bg-emerald-100">
                <Check className="h-7 w-7 text-emerald-600" />
              </div>
              <p className="text-sm font-semibold text-emerald-700">
                {t.appointmentDetail.success}
              </p>
            </div>
          ) : (
            <>
              {/* Status badge */}
              <div className="mb-4">
                <span className={`inline-flex items-center rounded-full px-2.5 py-1 text-xs font-semibold ${statusCfg.bg} ${statusCfg.text}`}>
                  {statusLabel}
                </span>
              </div>

              {/* Info de la cita */}
              <div className="space-y-3 mb-4">
                <div className="flex items-center gap-3">
                  <User className="h-4 w-4 flex-shrink-0 text-neutral-400" />
                  <div className="min-w-0 flex-1">
                    <p className="text-xs text-neutral-500">{t.appointmentDetail.customer}</p>
                    <p className="text-sm font-medium text-neutral-900 truncate">{appointment.customer_name}</p>
                  </div>
                </div>

                {appointment.customer_phone && (
                  <div className="flex items-center gap-3">
                    <Phone className="h-4 w-4 flex-shrink-0 text-neutral-400" />
                    <div className="min-w-0 flex-1">
                      <p className="text-xs text-neutral-500">{t.appointmentDetail.phone}</p>
                      <p className="text-sm font-medium text-neutral-900">{appointment.customer_phone}</p>
                    </div>
                  </div>
                )}

                <div className="flex items-center gap-3">
                  <Scissors className="h-4 w-4 flex-shrink-0 text-neutral-400" />
                  <div className="min-w-0 flex-1">
                    <p className="text-xs text-neutral-500">{t.appointmentDetail.service}</p>
                    <p className="text-sm font-medium text-neutral-900 truncate">
                      {appointment.service_name} · {appointment.service_duration_min} {t.common.min}
                    </p>
                  </div>
                </div>

                <div className="flex items-center gap-3">
                  <User className="h-4 w-4 flex-shrink-0 text-neutral-400" />
                  <div className="min-w-0 flex-1">
                    <p className="text-xs text-neutral-500">{t.appointmentDetail.professional}</p>
                    <p className="text-sm font-medium text-neutral-900 truncate">{appointment.professional_name}</p>
                  </div>
                </div>

                <div className="flex items-center gap-3">
                  <Calendar className="h-4 w-4 flex-shrink-0 text-neutral-400" />
                  <div className="min-w-0 flex-1">
                    <p className="text-xs text-neutral-500">{t.appointmentDetail.dateTime}</p>
                    <p className="text-sm font-medium text-neutral-900">
                      {formatInTimeZone(appointment.starts_at, tz, "EEEE d 'de' MMMM, h:mm a", { locale: dateLocale })}
                    </p>
                  </div>
                </div>

                {appointment.price != null && (
                  <div className="flex items-center gap-3">
                    <Clock className="h-4 w-4 flex-shrink-0 text-neutral-400" />
                    <div className="min-w-0 flex-1">
                      <p className="text-xs text-neutral-500">{t.appointmentDetail.price}</p>
                      <p className="text-sm font-medium text-neutral-900">${appointment.price.toFixed(2)} USD</p>
                    </div>
                  </div>
                )}

                {appointment.notes && (
                  <div className="rounded-lg bg-neutral-50 px-3 py-2">
                    <p className="text-xs text-neutral-500 mb-0.5">{t.appointmentDetail.notes}</p>
                    <p className="text-sm text-neutral-700">{appointment.notes}</p>
                  </div>
                )}

                <div className="flex items-center gap-2">
                  <span className="text-xs text-neutral-400">{t.appointmentDetail.source}:</span>
                  <span className="rounded-full bg-neutral-100 px-2 py-0.5 text-[10px] font-medium text-neutral-600">
                    {sourceLabel}
                  </span>
                </div>
              </div>

              {/* Sección de cancelar */}
              {mode === 'cancel' && (
                <div className="rounded-xl border border-red-200 bg-red-50/50 p-4 space-y-3">
                  <p className="text-sm font-semibold text-red-800">{t.appointmentDetail.cancelTitle}</p>
                  <textarea
                    value={cancelReason}
                    onChange={(e) => setCancelReason(e.target.value)}
                    placeholder={t.appointmentDetail.cancelReasonPlaceholder}
                    rows={2}
                    className="w-full resize-none rounded-lg border border-red-200 bg-white px-3 py-2.5 text-sm text-neutral-900 placeholder-neutral-400 focus:border-red-400 focus:outline-none focus:ring-2 focus:ring-red-100"
                  />
                  {error && (
                    <p className="rounded-lg bg-red-100 px-3 py-2 text-xs font-medium text-red-700">{error}</p>
                  )}
                  <div className="flex gap-2">
                    <button
                      onClick={() => { setMode('view'); setError(''); }}
                      className="rounded-lg border border-neutral-200 px-3 py-2 text-xs font-medium text-neutral-600 transition-colors hover:bg-neutral-50"
                    >
                      {t.appointmentDetail.back}
                    </button>
                    <button
                      onClick={handleCancel}
                      disabled={submitting}
                      className="flex items-center gap-1.5 rounded-lg bg-red-600 px-4 py-2 text-xs font-semibold text-white transition-colors hover:bg-red-500 disabled:opacity-50"
                    >
                      {submitting && <Spinner size="sm" className="text-white" />}
                      {submitting ? t.appointmentDetail.cancelling : t.appointmentDetail.confirmCancel}
                    </button>
                  </div>
                </div>
              )}

              {/* Sección de reagendar */}
              {mode === 'reschedule' && (
                <div className="rounded-xl border border-primary-200 bg-primary-50/50 p-4 space-y-3">
                  <p className="text-sm font-semibold text-primary-800">{t.appointmentDetail.rescheduleTitle}</p>

                  <div>
                    <label className="mb-1.5 block text-xs font-medium text-neutral-700">
                      {t.appointmentDetail.selectNewDate}
                    </label>
                    <input
                      type="date"
                      value={rescheduleDate}
                      onChange={(e) => setRescheduleDate(e.target.value)}
                      min={format(new Date(), 'yyyy-MM-dd')}
                      className="w-full rounded-lg border border-neutral-300 px-3 py-2.5 text-sm text-neutral-900 focus:border-primary-400 focus:outline-none focus:ring-2 focus:ring-primary-100"
                    />
                  </div>

                  {rescheduleDate && (
                    <div>
                      <label className="mb-1.5 block text-xs font-medium text-neutral-700">
                        {t.appointmentDetail.selectNewTime}
                      </label>
                      {loadingSlots ? (
                        <div className="flex items-center justify-center py-6">
                          <Spinner size="sm" />
                        </div>
                      ) : rescheduleSlots.length === 0 ? (
                        <p className="py-4 text-center text-sm text-neutral-400">
                          {t.appointmentDetail.noSlots}
                        </p>
                      ) : (
                        <div className="grid grid-cols-4 gap-2">
                          {rescheduleSlots.map((slot) => {
                            const timeStr = formatInTimeZone(slot.starts_at, tz, 'h:mm a', { locale: dateLocale });
                            const isSelected = selectedSlot?.starts_at === slot.starts_at;
                            return (
                              <button
                                key={slot.starts_at}
                                onClick={() => setSelectedSlot(slot)}
                                className={`rounded-lg border px-2 py-2 text-xs font-medium transition-all ${
                                  isSelected
                                    ? 'border-primary-400 bg-primary-50 text-primary-700 ring-1 ring-primary-200'
                                    : 'border-neutral-200 text-neutral-700 hover:border-primary-300 hover:bg-primary-50/50'
                                }`}
                              >
                                {timeStr}
                              </button>
                            );
                          })}
                        </div>
                      )}
                    </div>
                  )}

                  {error && (
                    <p className="rounded-lg bg-red-50 px-3 py-2 text-xs font-medium text-red-700">{error}</p>
                  )}

                  <div className="flex gap-2">
                    <button
                      onClick={() => { setMode('view'); setError(''); setRescheduleDate(''); setRescheduleSlots([]); setSelectedSlot(null); }}
                      className="rounded-lg border border-neutral-200 px-3 py-2 text-xs font-medium text-neutral-600 transition-colors hover:bg-neutral-50"
                    >
                      {t.appointmentDetail.back}
                    </button>
                    {selectedSlot && (
                      <button
                        onClick={handleReschedule}
                        disabled={submitting}
                        className="flex items-center gap-1.5 rounded-lg bg-primary-600 px-4 py-2 text-xs font-semibold text-white transition-colors hover:bg-primary-500 disabled:opacity-50"
                      >
                        {submitting && <Spinner size="sm" className="text-white" />}
                        {submitting ? t.appointmentDetail.rescheduling : t.appointmentDetail.confirmReschedule}
                      </button>
                    )}
                  </div>
                </div>
              )}

              {/* Error global (fuera de cancel/reschedule) */}
              {mode === 'view' && error && (
                <p className="rounded-lg bg-red-50 px-3 py-2 text-xs font-medium text-red-700 mb-4">{error}</p>
              )}
            </>
          )}
        </div>

        {/* Footer con acciones */}
        {!success && canAct && mode === 'view' && (
          <div className="flex flex-wrap items-center gap-2 border-t border-neutral-100 px-5 py-4">
            {appointment.status === 'pending' && (
              <button
                onClick={handleConfirm}
                disabled={submitting}
                className="flex items-center gap-1.5 rounded-lg bg-primary-600 px-4 py-2.5 text-xs font-semibold text-white transition-colors hover:bg-primary-500 disabled:opacity-50"
              >
                {submitting && <Spinner size="sm" className="text-white" />}
                {submitting ? t.appointmentDetail.confirming : t.appointmentDetail.confirm}
              </button>
            )}

            {appointment.status === 'confirmed' && (
              <>
                <button
                  onClick={handleComplete}
                  disabled={submitting}
                  className="flex items-center gap-1.5 rounded-lg bg-emerald-600 px-4 py-2.5 text-xs font-semibold text-white transition-colors hover:bg-emerald-500 disabled:opacity-50"
                >
                  {submitting ? t.appointmentDetail.completing : t.appointmentDetail.complete}
                </button>
                <button
                  onClick={handleNoShow}
                  disabled={submitting}
                  className="rounded-lg border border-neutral-200 px-3 py-2.5 text-xs font-medium text-neutral-600 transition-colors hover:bg-neutral-50 disabled:opacity-50"
                >
                  {t.appointmentDetail.markNoShow}
                </button>
              </>
            )}

            <button
              onClick={() => setMode('reschedule')}
              className="rounded-lg border border-primary-200 px-3 py-2.5 text-xs font-medium text-primary-600 transition-colors hover:bg-primary-50"
            >
              {t.appointmentDetail.reschedule}
            </button>

            <button
              onClick={() => setMode('cancel')}
              className="rounded-lg border border-red-200 px-3 py-2.5 text-xs font-medium text-red-600 transition-colors hover:bg-red-50"
            >
              {t.appointmentDetail.cancelAppt}
            </button>
          </div>
        )}
      </div>
    </div>
  );
}
