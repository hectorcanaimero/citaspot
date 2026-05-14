'use client';

import { useState, useEffect, useCallback } from 'react';
import { useParams } from 'next/navigation';
import { format, parseISO } from 'date-fns';
import { CheckCircle2, Calendar, Clock, User, Phone } from 'lucide-react';
import { Button }  from '@/components/ui/button';
import { Input }   from '@/components/ui/input';
import { Card }    from '@/components/ui/card';
import { Badge }   from '@/components/ui/badge';
import { Spinner } from '@/components/ui/spinner';
import { publicApi, PublicProfile, Service, Professional, TimeSlot, APIError } from '@/lib/api';
import { useTranslations, useDateLocale } from '@/lib/i18n';

type Step = 'service' | 'professional' | 'datetime' | 'contact' | 'confirm' | 'success';

export default function BookingPage() {
  const { slug }     = useParams<{ slug: string }>();
  const t            = useTranslations();
  const dateLocale   = useDateLocale();

  const [profile,        setProfile]        = useState<PublicProfile | null>(null);
  const [loadingProfile, setLoadingProfile] = useState(true);
  const [profileError,   setProfileError]   = useState('');

  const [step,                 setStep]                 = useState<Step>('service');
  const [selectedService,      setSelectedService]      = useState<Service | null>(null);
  const [selectedProfessional, setSelectedProfessional] = useState<Professional | null>(null);
  const [selectedDate,         setSelectedDate]         = useState('');
  const [slots,                setSlots]                = useState<TimeSlot[]>([]);
  const [loadingSlots,         setLoadingSlots]         = useState(false);
  const [selectedSlot,         setSelectedSlot]         = useState<TimeSlot | null>(null);

  const [name,  setName]  = useState('');
  const [phone, setPhone] = useState('');
  const [email, setEmail] = useState('');
  const [notes, setNotes] = useState('');

  const [booking,   setBooking]   = useState(false);
  const [bookError, setBookError] = useState('');
  const [booked,    setBooked]    = useState<{ starts_at: string } | null>(null);

  useEffect(() => {
    publicApi.getProfile(slug)
      .then(setProfile)
      .catch(() => setProfileError(t.booking.notFound))
      .finally(() => setLoadingProfile(false));
  }, [slug, t]);

  const loadSlots = useCallback(async () => {
    if (!selectedProfessional || !selectedService || !selectedDate) return;
    setLoadingSlots(true);
    try {
      const res = await publicApi.getAvailability(slug, selectedProfessional.id, selectedService.id, selectedDate);
      setSlots(res.data);
    } catch {
      setSlots([]);
    } finally {
      setLoadingSlots(false);
    }
  }, [slug, selectedProfessional, selectedService, selectedDate]);

  useEffect(() => {
    if (step === 'datetime' && selectedDate) loadSlots();
  }, [step, selectedDate, loadSlots]);

  async function handleBook(e: React.FormEvent) {
    e.preventDefault();
    if (!selectedSlot || !selectedProfessional || !selectedService) return;
    setBooking(true);
    setBookError('');
    try {
      const res = await publicApi.book(slug, {
        professional_id: selectedProfessional.id,
        service_id:      selectedService.id,
        starts_at:       selectedSlot.starts_at,
        customer_name:   name,
        customer_phone:  phone,
        customer_email:  email || undefined,
        notes:           notes || undefined,
      });
      setBooked(res);
      setStep('success');
    } catch (err) {
      setBookError(err instanceof APIError ? err.message : t.booking.bookingError);
    } finally {
      setBooking(false);
    }
  }

  if (loadingProfile) {
    return <div className="flex min-h-screen items-center justify-center"><Spinner size="lg" /></div>;
  }
  if (profileError || !profile) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <p className="text-neutral-500">{profileError || t.booking.notFound}</p>
      </div>
    );
  }

  // Header del negocio
  const Header = () => {
    const hasCover = Boolean(profile.cover_url);
    const hasLogo  = Boolean(profile.logo_url);

    return (
      <div className="mb-6">
        {hasCover ? (
          <div
            className="relative h-32 w-full overflow-hidden rounded-2xl bg-neutral-200 shadow-sm sm:h-40"
            style={{ backgroundImage: `url(${profile.cover_url})`, backgroundSize: 'cover', backgroundPosition: 'center' }}
          >
            <div className="absolute inset-0 bg-gradient-to-t from-black/40 via-black/0 to-black/0" />
          </div>
        ) : (
          <div className="h-20 w-full rounded-2xl bg-gradient-to-br from-primary-100 via-primary-50 to-sky-50 sm:h-24" />
        )}

        <div className={`relative z-10 flex flex-col items-center text-center ${hasCover ? '-mt-10' : '-mt-12'}`}>
          {hasLogo ? (
            // eslint-disable-next-line @next/next/no-img-element
            <img
              src={profile.logo_url}
              alt={profile.name}
              className="h-20 w-20 rounded-2xl border-4 border-white bg-white object-cover shadow-lg"
            />
          ) : (
            <div className="flex h-20 w-20 items-center justify-center rounded-2xl border-4 border-white bg-primary-600 text-3xl font-bold text-white shadow-lg">
              {profile.name[0]}
            </div>
          )}
          <h1 className="mt-3 text-xl font-bold text-neutral-900">{profile.name}</h1>
          {profile.city && (
            <p className="mt-0.5 text-sm text-neutral-500">{profile.city}</p>
          )}
          {profile.description && (
            <p className="mt-2 max-w-md px-2 text-sm text-neutral-600">{profile.description}</p>
          )}
          {profile.booking_intro_text && (
            <p className="mt-2 max-w-md px-2 text-sm text-neutral-600">{profile.booking_intro_text}</p>
          )}
        </div>
      </div>
    );
  };

  // ── Success ──────────────────────────────────────────────────────────────────
  if (step === 'success' && booked) {
    const dateStr = format(parseISO(booked.starts_at), "d 'de' MMMM 'a las' h:mm a", { locale: dateLocale });
    return (
      <div className="flex min-h-screen items-center justify-center bg-neutral-50 px-4">
        <div className="w-full max-w-md animate-slide-up">
          <Header />
          <Card className="text-center">
            <CheckCircle2 className="mx-auto mb-3 h-12 w-12 text-emerald-500" />
            <h2 className="text-lg font-semibold text-neutral-900">{t.booking.confirmedTitle}</h2>
            <p className="mt-2 text-sm text-neutral-500">
              {t.booking.confirmedDesc
                .replace('{service}', selectedService?.name ?? '')
                .replace('{professional}', selectedProfessional?.name ?? '')
                .replace('{date}', dateStr)}
            </p>
            <p className="mt-3 text-sm text-neutral-500">{t.booking.reminderDesc}</p>
            {profile?.booking_success_text && (
              <p className="text-neutral-600 mt-4 text-center">{profile.booking_success_text}</p>
            )}
          </Card>
        </div>
      </div>
    );
  }

  // Progress steps
  const STEPS: Step[] = ['service', 'professional', 'datetime', 'contact', 'confirm'];
  const stepIdx = STEPS.indexOf(step);

  return (
    <div className="min-h-screen bg-neutral-50 px-4 py-8">
      <div className="mx-auto max-w-lg">
        <Header />

        {/* Progress bar */}
        <div className="mb-6 flex gap-1">
          {STEPS.map((s, i) => (
            <div
              key={s}
              className={`h-1 flex-1 rounded-full transition-colors duration-300 ${
                i <= stepIdx ? 'bg-primary-600' : 'bg-neutral-200'
              }`}
            />
          ))}
        </div>

        <Card>
          {/* Paso 1: Servicio */}
          {step === 'service' && (
            <div>
              <h2 className="mb-4 text-base font-semibold text-neutral-900">{t.booking.whatService}</h2>
              <div className="flex flex-col gap-2">
                {profile.services.map((s) => (
                  <button
                    key={s.id}
                    onClick={() => {
                      setSelectedService(s);
                      const available = (profile.service_professionals ?? [])
                        .filter((sp) => sp.service_id === s.id)
                        .map((sp) => profile.professionals.find((p) => p.id === sp.professional_id))
                        .filter(Boolean);
                      if (available.length === 1) {
                        setSelectedProfessional(available[0]!);
                        setStep('datetime');
                      } else {
                        setStep('professional');
                      }
                    }}
                    className="flex items-center justify-between rounded-lg border border-neutral-200 px-4 py-3 text-left transition-colors hover:border-primary-300 hover:bg-primary-50"
                  >
                    <div>
                      <p className="text-sm font-medium text-neutral-900">{s.name}</p>
                      <p className="text-xs text-neutral-500">{s.duration_min} min</p>
                    </div>
                    {s.price && (
                      <Badge variant="primary">${s.price} {s.currency}</Badge>
                    )}
                  </button>
                ))}
              </div>
            </div>
          )}

          {/* Paso 2: Profesional */}
          {step === 'professional' && (() => {
            const availableProfessionals = profile.professionals.filter((p) =>
              (profile.service_professionals ?? []).some(
                (sp) => sp.service_id === selectedService?.id && sp.professional_id === p.id
              )
            );
            return (
            <div>
              <h2 className="mb-4 text-base font-semibold text-neutral-900">{t.booking.whoPrefer}</h2>
              <div className="flex flex-col gap-2">
                {availableProfessionals.length === 0 ? (
                  <p className="text-center text-sm text-neutral-500 py-4">
                    {t.booking.noProfessionalsForService}
                  </p>
                ) : (
                  availableProfessionals.map((p) => (
                    <button
                      key={p.id}
                      onClick={() => { setSelectedProfessional(p); setStep('datetime'); }}
                      className="flex items-center gap-3 rounded-lg border border-neutral-200 px-4 py-3 text-left transition-colors hover:border-primary-300 hover:bg-primary-50"
                    >
                      <div
                        className="flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-full text-white text-sm font-semibold"
                        style={{ backgroundColor: p.color || '#8b5cf6' }}
                      >
                        {p.name[0]}
                      </div>
                      <div>
                        <p className="text-sm font-medium text-neutral-900">{p.name}</p>
                        {p.specialty && <p className="text-xs text-neutral-500">{p.specialty}</p>}
                      </div>
                    </button>
                  ))
                )}
              </div>
              <Button variant="ghost" size="sm" onClick={() => setStep('service')} className="mt-4">
                {t.booking.backButton}
              </Button>
            </div>
            );
          })()}

          {/* Paso 3: Fecha y hora */}
          {step === 'datetime' && (
            <div>
              <h2 className="mb-4 text-base font-semibold text-neutral-900">{t.booking.whenCita}</h2>
              <Input
                label={t.booking.dateLabel}
                type="date"
                value={selectedDate}
                min={new Date().toISOString().split('T')[0]}
                onChange={(e) => { setSelectedDate(e.target.value); setSlots([]); setSelectedSlot(null); }}
              />
              {selectedDate && (
                <div className="mt-4">
                  {loadingSlots ? (
                    <div className="flex justify-center py-4"><Spinner /></div>
                  ) : slots.length === 0 ? (
                    <p className="text-center text-sm text-neutral-500 py-4">{t.booking.noSlots}</p>
                  ) : (
                    <>
                      <p className="mb-2 text-sm font-medium text-neutral-700">{t.booking.availableSlots}</p>
                      <div className="grid grid-cols-3 gap-2">
                        {slots.map((s) => {
                          const timeLabel = format(parseISO(s.starts_at), 'h:mm a');
                          const sel = selectedSlot?.starts_at === s.starts_at;
                          return (
                            <button
                              key={s.starts_at}
                              onClick={() => setSelectedSlot(s)}
                              className={`rounded-lg border px-2 py-2 text-sm font-medium transition-colors ${
                                sel
                                  ? 'border-primary-500 bg-primary-50 text-primary-700'
                                  : 'border-neutral-200 text-neutral-700 hover:border-primary-300'
                              }`}
                            >
                              {timeLabel}
                            </button>
                          );
                        })}
                      </div>
                    </>
                  )}
                </div>
              )}
              <div className="mt-4 flex gap-3">
                <Button variant="ghost" size="sm" onClick={() => setStep('professional')}>
                  {t.booking.backButton}
                </Button>
                <Button
                  size="md"
                  disabled={!selectedSlot}
                  onClick={() => setStep('contact')}
                  className="flex-1"
                >
                  {t.booking.reviewAppointment}
                </Button>
              </div>
            </div>
          )}

          {/* Paso 4: Datos de contacto */}
          {step === 'contact' && (
            <div className="flex flex-col gap-4">
              <h2 className="text-base font-semibold text-neutral-900">{t.booking.contactData}</h2>
              <Input label={t.booking.fullNameLabel} placeholder={t.booking.fullNamePlaceholder}
                value={name} onChange={(e) => setName(e.target.value)} required />
              <Input label={t.booking.phoneLabel} type="tel" placeholder={t.booking.phonePlaceholder}
                value={phone} onChange={(e) => setPhone(e.target.value)} required />
              <Input label={t.booking.emailOptionalLabel} type="email" placeholder="tu@correo.com"
                value={email} onChange={(e) => setEmail(e.target.value)} />
              <div className="flex gap-3">
                <Button variant="ghost" size="sm" onClick={() => setStep('datetime')}>
                  {t.booking.backButton}
                </Button>
                <Button size="md" disabled={!name || !phone} onClick={() => setStep('confirm')} className="flex-1">
                  {t.booking.reviewAppointment}
                </Button>
              </div>
            </div>
          )}

          {/* Paso 5: Confirmar */}
          {step === 'confirm' && selectedSlot && (
            <form onSubmit={handleBook}>
              <h2 className="mb-4 text-base font-semibold text-neutral-900">{t.booking.confirmAppointment}</h2>
              <div className="mb-4 flex flex-col gap-3 rounded-lg bg-neutral-50 p-4 text-sm">
                <div className="flex items-center gap-2 text-neutral-700">
                  <ScissorsIcon className="h-4 w-4 text-primary-500" />
                  <span>{selectedService?.name}</span>
                </div>
                <div className="flex items-center gap-2 text-neutral-700">
                  <User className="h-4 w-4 text-primary-500" />
                  <span>{selectedProfessional?.name}</span>
                </div>
                <div className="flex items-center gap-2 text-neutral-700">
                  <Calendar className="h-4 w-4 text-primary-500" />
                  <span>{format(parseISO(selectedSlot.starts_at), "d 'de' MMMM yyyy", { locale: dateLocale })}</span>
                </div>
                <div className="flex items-center gap-2 text-neutral-700">
                  <Clock className="h-4 w-4 text-primary-500" />
                  <span>{format(parseISO(selectedSlot.starts_at), 'h:mm a')}</span>
                </div>
                <div className="flex items-center gap-2 text-neutral-700">
                  <Phone className="h-4 w-4 text-primary-500" />
                  <span>{phone}</span>
                </div>
              </div>
              {bookError && (
                <p className="mb-3 rounded-md bg-red-50 px-3 py-2 text-sm text-red-700">{bookError}</p>
              )}
              <div className="flex gap-3">
                <Button type="button" variant="secondary" onClick={() => setStep('contact')} className="flex-1">
                  {t.booking.editButton}
                </Button>
                <Button type="submit" loading={booking} className="flex-1">
                  {t.booking.confirmButton}
                </Button>
              </div>
            </form>
          )}
        </Card>
      </div>
    </div>
  );
}

function ScissorsIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" stroke="currentColor" strokeWidth="2" viewBox="0 0 24 24">
      <circle cx="6" cy="6" r="3"/><circle cx="6" cy="18" r="3"/>
      <line x1="20" y1="4" x2="8.12" y2="15.88"/><line x1="14.47" y1="14.48" x2="20" y2="20"/>
      <line x1="8.12" y1="8.12" x2="12" y2="12"/>
    </svg>
  );
}
