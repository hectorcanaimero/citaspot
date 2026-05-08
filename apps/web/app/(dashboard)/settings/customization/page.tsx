'use client';

import { useState, useEffect } from 'react';
import { Card, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Spinner } from '@/components/ui/spinner';
import { settingsApi, TenantSettings } from '@/lib/api';

export default function CustomizationPage() {
  const [settings, setSettings] = useState<TenantSettings | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    settingsApi.get()
      .then(setSettings)
      .catch(console.error)
      .finally(() => setLoading(false));
  }, []);

  const handleSave = async () => {
    if (!settings) return;
    setSaving(true);
    try {
      await settingsApi.update(settings);
    } catch (err) {
      console.error(err);
    } finally {
      setSaving(false);
    }
  };

  if (loading) return <div className="flex justify-center py-16"><Spinner size="lg" /></div>;
  if (!settings) return null;

  return (
    <Card>
      <CardHeader>
        <CardTitle>Personalizaci\u00f3n</CardTitle>
      </CardHeader>

      <div className="space-y-6">
        <div>
          <label className="block text-sm font-medium text-neutral-700 mb-1">Nombre del asistente (bot)</label>
          <input
            type="text"
            className="w-full rounded-lg border border-neutral-300 px-3 py-2 text-sm"
            placeholder="Ej: Luna, Asistente Virtual"
            value={settings.bot_name}
            onChange={e => setSettings({ ...settings, bot_name: e.target.value })}
          />
          <p className="text-xs text-neutral-500 mt-1">Nombre con el que se presentar\u00e1 el bot por WhatsApp</p>
        </div>

        <div>
          <label className="block text-sm font-medium text-neutral-700 mb-1">Saludo inicial del bot</label>
          <textarea
            className="w-full rounded-lg border border-neutral-300 px-3 py-2 text-sm"
            rows={2}
            placeholder="Mensaje de bienvenida personalizado"
            value={settings.bot_greeting}
            onChange={e => setSettings({ ...settings, bot_greeting: e.target.value })}
          />
        </div>

        <div>
          <label className="block text-sm font-medium text-neutral-700 mb-1">Texto antes del formulario de reserva</label>
          <textarea
            className="w-full rounded-lg border border-neutral-300 px-3 py-2 text-sm"
            rows={3}
            placeholder="Texto que ver\u00e1n los clientes al abrir la p\u00e1gina de reservas"
            value={settings.booking_intro_text}
            onChange={e => setSettings({ ...settings, booking_intro_text: e.target.value })}
          />
        </div>

        <div>
          <label className="block text-sm font-medium text-neutral-700 mb-1">Texto despu\u00e9s de confirmar reserva</label>
          <textarea
            className="w-full rounded-lg border border-neutral-300 px-3 py-2 text-sm"
            rows={3}
            placeholder="Texto que ver\u00e1n los clientes despu\u00e9s de reservar"
            value={settings.booking_success_text}
            onChange={e => setSettings({ ...settings, booking_success_text: e.target.value })}
          />
        </div>

        <div>
          <label className="block text-sm font-medium text-neutral-700 mb-1">Recordatorios (minutos antes de la cita)</label>
          <div className="flex flex-wrap gap-2">
            {[
              { label: '24 horas', value: 1440 },
              { label: '12 horas', value: 720 },
              { label: '2 horas', value: 120 },
              { label: '1 hora', value: 60 },
              { label: '30 min', value: 30 },
            ].map(opt => (
              <button
                key={opt.value}
                type="button"
                className={`px-3 py-1.5 rounded-full text-sm border transition-colors ${
                  settings.reminder_minutes.includes(opt.value)
                    ? 'bg-primary-100 border-primary-500 text-primary-700'
                    : 'bg-white border-neutral-300 text-neutral-600 hover:border-neutral-400'
                }`}
                onClick={() => {
                  const mins = settings.reminder_minutes.includes(opt.value)
                    ? settings.reminder_minutes.filter(m => m !== opt.value)
                    : [...settings.reminder_minutes, opt.value];
                  setSettings({ ...settings, reminder_minutes: mins });
                }}
              >
                {opt.label}
              </button>
            ))}
          </div>
          <p className="text-xs text-neutral-500 mt-1">Selecciona cu\u00e1ndo enviar recordatorios por WhatsApp</p>
        </div>

        <Button onClick={handleSave} loading={saving}>
          {saving ? 'Guardando...' : 'Guardar cambios'}
        </Button>
      </div>
    </Card>
  );
}
