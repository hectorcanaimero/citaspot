'use client';

// Modal para crear citas manualmente desde el dashboard.
// Flujo: profesional -> servicio -> fecha -> horario -> datos opcionales -> crear.

import { useState, useEffect, useCallback } from 'react';
import { format } from 'date-fns';
import { X, ChevronRight, Check } from 'lucide-react';
import {
  appointments,
  professionals,
  services,
  auth,
  Professional,
  Service,
  TimeSlot,
  APIError,
} from '@/lib/api';
import { useTranslations, useDateLocale } from '@/lib/i18n';
import { Spinner } from '@/components/ui/spinner';
import {
  PhoneInput,
  validatePhone,
  CountryCode,
  PHONE_COUNTRIES,
} from '@/components/ui/PhoneInput';

interface NewAppointmentModalProps {
  open: boolean;
  onClose: () => void;
  onCreated: () => void;
  defaultDate?: string;
}

type Step = 'professional' | 'service' | 'datetime' | 'details';

export default function NewAppointmentModal({
  open,
  onClose,
  onCreated,
  defaultDate,
}: NewAppointmentModalProps) {
  const t = useTranslations();
  const dateLocale = useDateLocale();

  // Datos maestros
  const [profList, setProfList] = useState<Professional[]>([]);
  const [svcList, setSvcList] = useState<Service[]>([]);
  const [loadingData, setLoadingData] = useState(true);

  // Selecciones
  const [step, setStep] = useState<Step>('professional');
  const [selectedProf, setSelectedProf] = useState<Professional | null>(null);
  const [selectedSvc, setSelectedSvc] = useState<Service | null>(null);
  const [selectedDate, setSelectedDate] = useState(defaultDate ?? '');
  const [selectedSlot, setSelectedSlot] = useState<TimeSlot | null>(null);

  // Slots
  const [slots, setSlots] = useState<TimeSlot[]>([]);
  const [loadingSlots, setLoadingSlots] = useState(false);

  // Datos del cliente (nombre y teléfono obligatorios)
  const [customerName, setCustomerName] = useState('');
  const [customerPhone, setCustomerPhone] = useState('');
  const [phoneError, setPhoneError] = useState('');
  const [notes, setNotes] = useState('');

  // País del tenant para preseleccionar prefijo en PhoneInput
  const [tenantCountry, setTenantCountry] = useState<string | undefined>(undefined);

  // Estado de submit
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState(false);

  // Cargar profesionales y servicios al abrir
  useEffect(() => {
    if (!open) return;
    setLoadingData(true);
    Promise.all([professionals.list(), services.list()])
      .then(([profRes, svcRes]) => {
        setProfList((profRes.data ?? []).filter((p) => p.is_active));
        setSvcList((svcRes.data ?? []).filter((s) => s.is_active));
      })
      .catch(() => {})
      .finally(() => setLoadingData(false));
    // Resolver el país del tenant para el PhoneInput (no bloquea la UI)
    auth.me().then(({ tenant }) => setTenantCountry(tenant.country)).catch(() => {});
  }, [open]);

  // Reset al cerrar
  useEffect(() => {
    if (!open) {
      setStep('professional');
      setSelectedProf(null);
      setSelectedSvc(null);
      setSelectedDate(defaultDate ?? '');
      setSelectedSlot(null);
      setSlots([]);
      setCustomerName('');
      setCustomerPhone('');
      setPhoneError('');
      setNotes('');
      setError('');
      setSuccess(false);
    }
  }, [open, defaultDate]);

  // Cargar disponibilidad cuando hay prof + servicio + fecha
  const loadAvailability = useCallback(async () => {
    if (!selectedProf || !selectedSvc || !selectedDate) {
      setSlots([]);
      return;
    }
    setLoadingSlots(true);
    setSelectedSlot(null);
    try {
      const res = await appointments.availability(
        selectedProf.id,
        selectedSvc.id,
        selectedDate,
      );
      setSlots(res.data ?? []);
    } catch {
      setSlots([]);
    } finally {
      setLoadingSlots(false);
    }
  }, [selectedProf, selectedSvc, selectedDate]);

  useEffect(() => {
    if (step === 'datetime') {
      loadAvailability();
    }
  }, [step, loadAvailability]);

  // Submit
  async function handleSubmit() {
    if (!selectedProf || !selectedSvc || !selectedSlot) return;

    // Validar teléfono: detectar país por prefijo y verificar cantidad de dígitos
    let detectedCountry: CountryCode = 'DO';
    for (const code of Object.keys(PHONE_COUNTRIES) as CountryCode[]) {
      if (customerPhone.startsWith(PHONE_COUNTRIES[code].prefix)) {
        detectedCountry = code;
        break;
      }
    }
    const localDigits = customerPhone.slice(PHONE_COUNTRIES[detectedCountry].prefix.length);
    if (!validatePhone(detectedCountry, localDigits)) {
      setPhoneError(t.booking.phoneInvalid);
      return;
    }
    setPhoneError('');

    setSubmitting(true);
    setError('');
    try {
      await appointments.create({
        professional_id: selectedProf.id,
        service_id: selectedSvc.id,
        starts_at: selectedSlot.starts_at,
        customer_name: customerName.trim(),
        customer_phone: customerPhone,
        notes: notes || undefined,
        source: 'dashboard',
      });
      setSuccess(true);
      setTimeout(() => {
        onCreated();
        onClose();
      }, 1200);
    } catch (err) {
      setError(
        err instanceof APIError
          ? err.message
          : t.newAppointmentModal.error,
      );
    } finally {
      setSubmitting(false);
    }
  }

  if (!open) return null;

  // Capa de fondo oscuro + centrado
  return (
    <div
      className="fixed inset-0 z-50 flex items-end justify-center bg-black/50 backdrop-blur-sm sm:items-center"
      onClick={(e) => {
        if (e.target === e.currentTarget) onClose();
      }}
    >
      <div className="relative w-full max-w-md rounded-t-2xl bg-white shadow-2xl sm:rounded-2xl">
        {/* Header */}
        <div className="flex items-center justify-between border-b border-neutral-100 px-5 py-4">
          <h2 className="text-base font-bold text-neutral-900">
            {t.newAppointmentModal.title}
          </h2>
          <button
            onClick={onClose}
            className="rounded-lg p-1.5 text-neutral-400 transition-colors hover:bg-neutral-100 hover:text-neutral-600"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        {/* Indicador de pasos */}
        <div className="flex gap-1 px-5 pt-4">
          {(['professional', 'service', 'datetime', 'details'] as Step[]).map(
            (s, i) => (
              <div
                key={s}
                className={`h-1 flex-1 rounded-full transition-colors ${
                  i <=
                  ['professional', 'service', 'datetime', 'details'].indexOf(
                    step,
                  )
                    ? 'bg-primary-500'
                    : 'bg-neutral-200'
                }`}
              />
            ),
          )}
        </div>

        {/* Contenido */}
        <div className="max-h-[60vh] min-h-[280px] overflow-y-auto px-5 py-4">
          {/* Estado de exito */}
          {success ? (
            <div className="flex flex-col items-center py-10">
              <div className="mb-3 flex h-14 w-14 items-center justify-center rounded-full bg-emerald-100">
                <Check className="h-7 w-7 text-emerald-600" />
              </div>
              <p className="text-sm font-semibold text-emerald-700">
                {t.newAppointmentModal.success}
              </p>
            </div>
          ) : loadingData ? (
            <div className="flex items-center justify-center py-16">
              <Spinner size="lg" />
            </div>
          ) : (
            <>
              {/* Paso 1: Profesional */}
              {step === 'professional' && (
                <div className="space-y-2">
                  <p className="mb-3 text-sm font-medium text-neutral-600">
                    {t.newAppointmentModal.selectProfessional}
                  </p>
                  {profList.map((prof) => (
                    <button
                      key={prof.id}
                      onClick={() => {
                        setSelectedProf(prof);
                        setStep('service');
                      }}
                      className="flex w-full items-center gap-3 rounded-xl border border-neutral-200 px-4 py-3 text-left transition-all hover:border-primary-300 hover:bg-primary-50"
                    >
                      <span
                        className="flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-full text-sm font-bold text-white"
                        style={{
                          backgroundColor: prof.color || '#8b5cf6',
                        }}
                      >
                        {prof.name[0].toUpperCase()}
                      </span>
                      <div className="min-w-0 flex-1">
                        <p className="truncate text-sm font-semibold text-neutral-900">
                          {prof.name}
                        </p>
                        {prof.specialty && (
                          <p className="truncate text-xs text-neutral-500">
                            {prof.specialty}
                          </p>
                        )}
                      </div>
                      <ChevronRight className="h-4 w-4 flex-shrink-0 text-neutral-400" />
                    </button>
                  ))}
                </div>
              )}

              {/* Paso 2: Servicio */}
              {step === 'service' && (
                <div className="space-y-2">
                  <p className="mb-3 text-sm font-medium text-neutral-600">
                    {t.newAppointmentModal.selectService}
                  </p>
                  {svcList.map((svc) => (
                    <button
                      key={svc.id}
                      onClick={() => {
                        setSelectedSvc(svc);
                        setStep('datetime');
                      }}
                      className="flex w-full items-center justify-between rounded-xl border border-neutral-200 px-4 py-3 text-left transition-all hover:border-primary-300 hover:bg-primary-50"
                    >
                      <div className="min-w-0 flex-1">
                        <p className="truncate text-sm font-semibold text-neutral-900">
                          {svc.name}
                        </p>
                        <p className="text-xs text-neutral-500">
                          {svc.duration_min} {t.common.minutes}
                          {svc.price != null &&
                            ` · $${svc.price.toFixed(2)} ${svc.currency}`}
                        </p>
                      </div>
                      <ChevronRight className="h-4 w-4 flex-shrink-0 text-neutral-400" />
                    </button>
                  ))}
                </div>
              )}

              {/* Paso 3: Fecha y horario */}
              {step === 'datetime' && (
                <div className="space-y-4">
                  {/* Resumen de seleccion */}
                  <div className="flex items-center gap-2 text-xs text-neutral-500">
                    <span
                      className="flex h-5 w-5 items-center justify-center rounded-full text-[10px] font-bold text-white"
                      style={{
                        backgroundColor:
                          selectedProf?.color || '#8b5cf6',
                      }}
                    >
                      {selectedProf?.name[0].toUpperCase()}
                    </span>
                    <span className="font-medium text-neutral-700">
                      {selectedProf?.name}
                    </span>
                    <span>·</span>
                    <span>{selectedSvc?.name}</span>
                  </div>

                  {/* Selector de fecha */}
                  <div>
                    <label className="mb-1.5 block text-sm font-medium text-neutral-700">
                      {t.newAppointmentModal.selectDate}
                    </label>
                    <input
                      type="date"
                      value={selectedDate}
                      onChange={(e) => setSelectedDate(e.target.value)}
                      min={format(new Date(), 'yyyy-MM-dd')}
                      className="w-full rounded-lg border border-neutral-300 px-3 py-2.5 text-sm text-neutral-900 focus:border-primary-400 focus:outline-none focus:ring-2 focus:ring-primary-100"
                    />
                  </div>

                  {/* Slots */}
                  {selectedDate && (
                    <div>
                      <label className="mb-1.5 block text-sm font-medium text-neutral-700">
                        {t.newAppointmentModal.selectTime}
                      </label>
                      {loadingSlots ? (
                        <div className="flex items-center justify-center py-6">
                          <Spinner size="sm" />
                        </div>
                      ) : slots.length === 0 ? (
                        <p className="py-4 text-center text-sm text-neutral-400">
                          {t.newAppointmentModal.noSlots}
                        </p>
                      ) : (
                        <div className="grid grid-cols-4 gap-2">
                          {slots.map((slot) => {
                            const timeStr = format(
                              new Date(slot.starts_at),
                              'h:mm a',
                              { locale: dateLocale },
                            );
                            const isSelected =
                              selectedSlot?.starts_at ===
                              slot.starts_at;
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
                </div>
              )}

              {/* Paso 4: Datos del cliente */}
              {step === 'details' && (
                <div className="space-y-3">
                  {/* Resumen */}
                  <div className="rounded-lg bg-neutral-50 px-3 py-2.5 text-xs text-neutral-600">
                    <span className="font-semibold text-neutral-800">
                      {selectedProf?.name}
                    </span>
                    {' · '}
                    {selectedSvc?.name}
                    {' · '}
                    {selectedSlot &&
                      format(
                        new Date(selectedSlot.starts_at),
                        "d MMM, h:mm a",
                        { locale: dateLocale },
                      )}
                  </div>

                  <div>
                    <label className="mb-1 block text-xs font-medium text-neutral-600">
                      {t.newAppointmentModal.customerName}
                    </label>
                    <input
                      type="text"
                      value={customerName}
                      onChange={(e) => setCustomerName(e.target.value)}
                      required
                      className="w-full rounded-lg border border-neutral-300 px-3 py-2.5 text-sm text-neutral-900 focus:border-primary-400 focus:outline-none focus:ring-2 focus:ring-primary-100"
                    />
                  </div>

                  <div>
                    <PhoneInput
                      label={t.newAppointmentModal.customerPhone}
                      defaultCountry={tenantCountry}
                      value={customerPhone}
                      onChange={(fullPhone) => {
                        setCustomerPhone(fullPhone);
                        setPhoneError('');
                      }}
                      error={phoneError}
                      required
                    />
                  </div>

                  <div>
                    <label className="mb-1 block text-xs font-medium text-neutral-600">
                      {t.newAppointmentModal.notes}
                    </label>
                    <textarea
                      value={notes}
                      onChange={(e) => setNotes(e.target.value)}
                      rows={2}
                      className="w-full resize-none rounded-lg border border-neutral-300 px-3 py-2.5 text-sm text-neutral-900 focus:border-primary-400 focus:outline-none focus:ring-2 focus:ring-primary-100"
                    />
                  </div>

                  {error && (
                    <p className="rounded-lg bg-red-50 px-3 py-2 text-xs font-medium text-red-700">
                      {error}
                    </p>
                  )}
                </div>
              )}
            </>
          )}
        </div>

        {/* Footer */}
        {!success && !loadingData && (
          <div className="flex items-center gap-3 border-t border-neutral-100 px-5 py-4">
            {step !== 'professional' ? (
              <button
                onClick={() => {
                  if (step === 'service') setStep('professional');
                  else if (step === 'datetime') setStep('service');
                  else if (step === 'details') setStep('datetime');
                }}
                className="rounded-lg border border-neutral-200 px-4 py-2.5 text-sm font-medium text-neutral-600 transition-colors hover:bg-neutral-50"
              >
                {t.common.back}
              </button>
            ) : (
              <button
                onClick={onClose}
                className="rounded-lg border border-neutral-200 px-4 py-2.5 text-sm font-medium text-neutral-600 transition-colors hover:bg-neutral-50"
              >
                {t.newAppointmentModal.cancel}
              </button>
            )}

            <div className="flex-1" />

            {step === 'datetime' && selectedSlot && (
              <button
                onClick={() => setStep('details')}
                className="rounded-lg bg-primary-600 px-5 py-2.5 text-sm font-semibold text-white transition-colors hover:bg-primary-500"
              >
                {t.common.continue}
              </button>
            )}

            {step === 'details' && (
              <button
                onClick={handleSubmit}
                disabled={submitting || !customerName.trim() || !customerPhone}
                className="flex items-center gap-2 rounded-lg bg-primary-600 px-5 py-2.5 text-sm font-semibold text-white transition-colors hover:bg-primary-500 disabled:opacity-50"
              >
                {submitting && <Spinner size="sm" className="text-white" />}
                {submitting
                  ? t.newAppointmentModal.creating
                  : t.newAppointmentModal.create}
              </button>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
