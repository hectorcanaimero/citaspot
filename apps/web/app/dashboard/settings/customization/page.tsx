'use client';

import { useState, useEffect, useRef } from 'react';
import { Image as ImageIcon, Upload, Trash2, AlertCircle } from 'lucide-react';
import { Card, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Spinner } from '@/components/ui/spinner';
import { settingsApi, brandingApi, TenantSettings, BrandingAssetKind, APIError } from '@/lib/api';

const ACCEPTED_MIME = ['image/jpeg', 'image/png', 'image/webp'];
const LOGO_MAX_BYTES = 2 * 1024 * 1024;
const COVER_MAX_BYTES = 5 * 1024 * 1024;

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
        <CardTitle>Personalización</CardTitle>
      </CardHeader>

      <div className="space-y-8">

        {/* ── Página pública ──────────────────────────────────────────────── */}
        <section className="space-y-4">
          <div>
            <h2 className="text-base font-semibold text-neutral-900">Página pública de reservas</h2>
            <p className="mt-0.5 text-xs text-neutral-500">
              Personalizá lo que ven tus clientes cuando entran a tu página de reservas.
            </p>
          </div>

          <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
            <AssetUploader
              kind="cover"
              label="Foto de portada"
              hint="Imagen ancha que se muestra arriba de la página. Recomendado 1500×500 px. Máx. 5 MB."
              aspect="aspect-[3/1]"
              maxBytes={COVER_MAX_BYTES}
              currentUrl={settings.cover_url}
              onUploaded={(url) => setSettings({ ...settings, cover_url: url })}
              onRemoved={() => setSettings({ ...settings, cover_url: '' })}
            />
            <AssetUploader
              kind="logo"
              label="Logo"
              hint="Aparece sobre la portada. Recomendado cuadrado, 512×512 px. Máx. 2 MB."
              aspect="aspect-square"
              maxBytes={LOGO_MAX_BYTES}
              currentUrl={settings.logo_url}
              onUploaded={(url) => setSettings({ ...settings, logo_url: url })}
              onRemoved={() => setSettings({ ...settings, logo_url: '' })}
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-neutral-700 mb-1">Descripción del negocio</label>
            <textarea
              className="w-full rounded-lg border border-neutral-300 px-3 py-2 text-sm"
              rows={3}
              maxLength={400}
              placeholder="Contale a tus clientes qué hacés, dónde estás, qué te diferencia..."
              value={settings.description ?? ''}
              onChange={e => setSettings({ ...settings, description: e.target.value })}
            />
            <p className="mt-1 text-xs text-neutral-500">
              Aparece debajo del nombre, en la página pública. Hasta 400 caracteres.
            </p>
          </div>
        </section>

        {/* ── Asistente IA ───────────────────────────────────────────────── */}
        <section className="space-y-4 border-t border-neutral-100 pt-6">
          <h2 className="text-base font-semibold text-neutral-900">Asistente IA por WhatsApp</h2>

          <div>
            <label className="block text-sm font-medium text-neutral-700 mb-1">Nombre del asistente (bot)</label>
            <input
              type="text"
              className="w-full rounded-lg border border-neutral-300 px-3 py-2 text-sm"
              placeholder="Ej: Luna, Asistente Virtual"
              value={settings.bot_name}
              onChange={e => setSettings({ ...settings, bot_name: e.target.value })}
            />
            <p className="text-xs text-neutral-500 mt-1">Nombre con el que se presentará el bot por WhatsApp</p>
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
        </section>

        {/* ── Textos del flujo de reserva ────────────────────────────────── */}
        <section className="space-y-4 border-t border-neutral-100 pt-6">
          <h2 className="text-base font-semibold text-neutral-900">Textos del flujo de reserva</h2>

          <div>
            <label className="block text-sm font-medium text-neutral-700 mb-1">Texto antes del formulario de reserva</label>
            <textarea
              className="w-full rounded-lg border border-neutral-300 px-3 py-2 text-sm"
              rows={3}
              placeholder="Texto que verán los clientes al abrir la página de reservas"
              value={settings.booking_intro_text}
              onChange={e => setSettings({ ...settings, booking_intro_text: e.target.value })}
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-neutral-700 mb-1">Texto después de confirmar reserva</label>
            <textarea
              className="w-full rounded-lg border border-neutral-300 px-3 py-2 text-sm"
              rows={3}
              placeholder="Texto que verán los clientes después de reservar"
              value={settings.booking_success_text}
              onChange={e => setSettings({ ...settings, booking_success_text: e.target.value })}
            />
          </div>
        </section>

        {/* ── Recordatorios ──────────────────────────────────────────────── */}
        <section className="space-y-4 border-t border-neutral-100 pt-6">
          <h2 className="text-base font-semibold text-neutral-900">Recordatorios automáticos</h2>
          <div>
            <label className="block text-sm font-medium text-neutral-700 mb-1">Minutos antes de la cita</label>
            <div className="flex flex-wrap gap-2">
              {[
                { label: '48 horas', value: 2880 },
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
            <p className="text-xs text-neutral-500 mt-1">Seleccioná cuándo enviar recordatorios por WhatsApp</p>
          </div>
        </section>

        <Button onClick={handleSave} loading={saving}>
          {saving ? 'Guardando...' : 'Guardar cambios'}
        </Button>
      </div>
    </Card>
  );
}

// ── AssetUploader ─────────────────────────────────────────────────────────────

interface AssetUploaderProps {
  kind: BrandingAssetKind;
  label: string;
  hint: string;
  aspect: string;
  maxBytes: number;
  currentUrl?: string;
  onUploaded: (url: string) => void;
  onRemoved: () => void;
}

function AssetUploader({ kind, label, hint, aspect, maxBytes, currentUrl, onUploaded, onRemoved }: AssetUploaderProps) {
  const inputRef = useRef<HTMLInputElement>(null);
  const [uploading, setUploading] = useState(false);
  const [error, setError] = useState('');

  function pickFile() {
    setError('');
    inputRef.current?.click();
  }

  async function handleChange(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    e.target.value = '';
    if (!file) return;

    if (!ACCEPTED_MIME.includes(file.type)) {
      setError('Formato no soportado. Usá JPG, PNG o WebP.');
      return;
    }
    if (file.size > maxBytes) {
      const mb = Math.round(maxBytes / (1024 * 1024));
      setError(`El archivo supera el límite de ${mb} MB.`);
      return;
    }

    setUploading(true);
    setError('');
    try {
      const { url } = kind === 'logo'
        ? await brandingApi.uploadLogo(file)
        : await brandingApi.uploadCover(file);
      onUploaded(url);
    } catch (err) {
      setError(err instanceof APIError ? err.message : 'Error al subir el archivo');
    } finally {
      setUploading(false);
    }
  }

  async function handleRemove() {
    if (!currentUrl) return;
    setUploading(true);
    setError('');
    try {
      await brandingApi.remove(kind);
      onRemoved();
    } catch (err) {
      setError(err instanceof APIError ? err.message : 'Error al eliminar');
    } finally {
      setUploading(false);
    }
  }

  return (
    <div>
      <label className="block text-sm font-medium text-neutral-700 mb-1">{label}</label>
      <div
        className={`relative w-full ${aspect} overflow-hidden rounded-xl border-2 border-dashed border-neutral-200 bg-neutral-50`}
      >
        {currentUrl ? (
          <>
            {/* eslint-disable-next-line @next/next/no-img-element */}
            <img src={currentUrl} alt={label} className="h-full w-full object-cover" />
            {!uploading && (
              <div className="absolute inset-0 flex items-end justify-end gap-2 bg-gradient-to-t from-black/40 via-transparent to-transparent p-3 opacity-0 transition-opacity hover:opacity-100">
                <button
                  type="button"
                  onClick={pickFile}
                  className="flex items-center gap-1.5 rounded-md bg-white/95 px-2.5 py-1.5 text-xs font-semibold text-neutral-700 shadow hover:bg-white"
                >
                  <Upload className="h-3.5 w-3.5" /> Reemplazar
                </button>
                <button
                  type="button"
                  onClick={handleRemove}
                  className="flex items-center gap-1.5 rounded-md bg-white/95 px-2.5 py-1.5 text-xs font-semibold text-red-600 shadow hover:bg-white"
                >
                  <Trash2 className="h-3.5 w-3.5" /> Eliminar
                </button>
              </div>
            )}
          </>
        ) : (
          <button
            type="button"
            onClick={pickFile}
            className="flex h-full w-full flex-col items-center justify-center gap-1 text-neutral-400 transition-colors hover:bg-primary-50/40 hover:text-primary-600"
          >
            <ImageIcon className="h-7 w-7" />
            <span className="text-xs font-medium">Click para subir</span>
            <span className="text-[10px] uppercase tracking-wide text-neutral-400">JPG · PNG · WebP</span>
          </button>
        )}

        {uploading && (
          <div className="absolute inset-0 flex items-center justify-center bg-white/70 backdrop-blur-sm">
            <Spinner />
          </div>
        )}
      </div>

      <input
        ref={inputRef}
        type="file"
        accept={ACCEPTED_MIME.join(',')}
        className="hidden"
        onChange={handleChange}
      />

      <p className="mt-1 text-xs text-neutral-500">{hint}</p>
      {error && (
        <p className="mt-1 flex items-center gap-1 text-xs text-red-600">
          <AlertCircle className="h-3 w-3" /> {error}
        </p>
      )}
    </div>
  );
}
