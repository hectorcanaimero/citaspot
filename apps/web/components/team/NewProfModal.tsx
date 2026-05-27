'use client';

// Modal para crear un nuevo profesional desde la lista del equipo.
// Adaptado de NewProfForm original, agregando campo email.

import { useState } from 'react';
import { X } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { PhoneInput, validatePhone, CountryCode, PHONE_COUNTRIES } from '@/components/ui/PhoneInput';
import { professionals as profsApi, Professional, ProfessionalInput, APIError } from '@/lib/api';
import { useTranslations } from '@/lib/i18n';

const COLORS = ['#8b5cf6', '#ec4899', '#f59e0b', '#10b981', '#3b82f6', '#ef4444', '#14b8a6'];

interface NewProfModalProps {
  onCreated: (p: Professional) => void;
  onClose: () => void;
  tenantCountry?: string;
}

export function NewProfModal({ onCreated, onClose, tenantCountry }: NewProfModalProps) {
  const t = useTranslations();
  const [name, setName] = useState('');
  const [spec, setSpec] = useState('');
  const [email, setEmail] = useState('');
  const [phone, setPhone] = useState('');
  const [phoneError, setPhoneError] = useState('');
  const [color, setColor] = useState(COLORS[0]);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!name.trim()) return;

    // Teléfono es opcional — solo validar si el usuario ingresó algo.
    if (phone) {
      let detected: CountryCode = 'DO';
      for (const code of Object.keys(PHONE_COUNTRIES) as CountryCode[]) {
        if (phone.startsWith(PHONE_COUNTRIES[code].prefix)) {
          detected = code;
          break;
        }
      }
      const local = phone.slice(PHONE_COUNTRIES[detected].prefix.length);
      if (!validatePhone(detected, local)) {
        setPhoneError(t.booking.phoneInvalid);
        return;
      }
    }
    setPhoneError('');

    setSaving(true);
    setError('');
    try {
      const input: ProfessionalInput = {
        name: name.trim(),
        specialty: spec.trim() || undefined,
        email: email.trim() || undefined,
        phone: phone || undefined,
        color,
        is_active: true,
      };
      const prof = await profsApi.create(input);
      onCreated(prof);
    } catch (err) {
      setError(err instanceof APIError ? err.message : t.team.addProfError);
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
      <div className="w-full max-w-md overflow-hidden rounded-xl border border-neutral-200 bg-white shadow-xl">
        <div className="flex items-center justify-between border-b border-neutral-100 px-5 py-4">
          <h3 className="text-sm font-semibold text-neutral-900">{t.team.newProfessional}</h3>
          <button
            type="button"
            onClick={onClose}
            className="rounded-lg p-1 text-neutral-400 hover:bg-neutral-100 hover:text-neutral-700 transition-colors"
          >
            <X className="h-4 w-4" />
          </button>
        </div>
        <div className="px-5 py-4">
          <form onSubmit={handleSubmit} className="flex flex-col gap-3">
            <Input
              label={t.team.nameLabel}
              placeholder={t.team.namePlaceholder}
              value={name}
              onChange={(e) => setName(e.target.value)}
              required
            />
            <Input
              label={t.team.specialtyLabel}
              placeholder={t.team.specialtyPlaceholder}
              value={spec}
              onChange={(e) => setSpec(e.target.value)}
            />
            <Input
              label={t.team.email}
              type="email"
              placeholder={t.team.emailPlaceholder}
              value={email}
              onChange={(e) => setEmail(e.target.value)}
            />
            <PhoneInput
              label={t.team.phoneLabel}
              defaultCountry={tenantCountry}
              value={phone}
              onChange={(full) => {
                setPhone(full);
                setPhoneError('');
              }}
              error={phoneError}
            />
            <div className="flex flex-col gap-1.5">
              <span className="text-sm font-medium text-neutral-700">{t.team.colorLabel}</span>
              <div className="flex gap-2">
                {COLORS.map((c) => (
                  <button
                    key={c}
                    type="button"
                    onClick={() => setColor(c)}
                    className={`h-7 w-7 rounded-full transition-transform ${
                      color === c ? 'ring-2 ring-offset-2 ring-primary-500 scale-110' : ''
                    }`}
                    style={{ backgroundColor: c }}
                  />
                ))}
              </div>
            </div>
            {error && (
              <p className="rounded-md bg-red-50 px-3 py-1.5 text-sm text-red-700">{error}</p>
            )}
            <div className="flex justify-end gap-2 pt-1">
              <button
                type="button"
                onClick={onClose}
                className="rounded-lg border border-neutral-200 px-4 py-2 text-sm text-neutral-700 hover:bg-neutral-50 transition-colors"
              >
                {t.common.cancel}
              </button>
              <Button type="submit" size="sm" loading={saving} disabled={!name.trim()}>
                {t.team.addProfessional}
              </Button>
            </div>
          </form>
        </div>
      </div>
    </div>
  );
}
