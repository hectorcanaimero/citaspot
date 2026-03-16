'use client';

// Paso 3 del onboarding: conectar WhatsApp Business.
// Usa polling cada 5s hacia GET /api/v1/whatsapp/status.
import { useState, useEffect, useRef } from 'react';
import { useRouter } from 'next/navigation';
import { CheckCircle2, XCircle, RefreshCw, MessageCircle } from 'lucide-react';
import { Button }  from '@/components/ui/button';
import { Card }    from '@/components/ui/card';
import { Badge }   from '@/components/ui/badge';
import { Spinner } from '@/components/ui/spinner';
import { whatsapp, APIError } from '@/lib/api';
import { useTranslations } from '@/lib/i18n';

export default function OnboardingWhatsAppPage() {
  const t      = useTranslations();
  const router = useRouter();
  const [status,  setStatus]  = useState<'CONNECTED' | 'DISCONNECTED' | 'CONNECTING' | null>(null);
  const [loading, setLoading] = useState(true);
  const [error,   setError]   = useState('');
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

  async function fetchStatus() {
    try {
      const res = await whatsapp.getStatus();
      setStatus(res.status);
      setError('');
    } catch (err) {
      setError(err instanceof APIError ? err.message : t.onboarding.waQueryError);
      setStatus('DISCONNECTED');
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    fetchStatus();
    intervalRef.current = setInterval(fetchStatus, 5000);
    return () => {
      if (intervalRef.current) clearInterval(intervalRef.current);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const isConnected  = status === 'CONNECTED';
  const isConnecting = status === 'CONNECTING';
  const evolutionUrl = (process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:3001').replace(':3001', ':8080');

  return (
    <div className="mx-auto max-w-lg px-4 py-10">
      {/* Progreso */}
      <div className="mb-2 flex gap-2">
        {[1, 2, 3, 4].map((s) => (
          <div
            key={s}
            className={`h-1.5 flex-1 rounded-full transition-colors duration-300 ${
              s <= 3 ? 'bg-primary-600' : 'bg-neutral-200'
            }`}
          />
        ))}
      </div>
      <p className="mb-6 text-xs text-neutral-400">
        {t.onboarding.stepOf
          .replace('{step}', '3')
          .replace('{total}', '4')
          .replace('{label}', t.onboarding.whatsappLabel)}
      </p>

      <h1 className="mb-1 text-xl font-bold text-neutral-900">{t.onboarding.whatsappTitle}</h1>
      <p className="mb-6 text-sm text-neutral-500">{t.onboarding.whatsappDesc}</p>

      <Card>
        {loading ? (
          <div className="flex justify-center py-8"><Spinner /></div>
        ) : (
          <div className="flex flex-col gap-4">
            {/* Estado actual */}
            <div className="flex items-center gap-3">
              {isConnected ? (
                <CheckCircle2 className="h-8 w-8 text-emerald-500 flex-shrink-0" />
              ) : (
                <XCircle className="h-8 w-8 text-red-400 flex-shrink-0" />
              )}
              <div className="flex-1">
                <p className="text-sm font-medium text-neutral-900">
                  {isConnected
                    ? t.onboarding.waConnected
                    : isConnecting
                    ? t.onboarding.waConnecting
                    : t.onboarding.waDisconnected}
                </p>
                <p className="text-xs text-neutral-500">
                  {isConnected ? t.onboarding.waAiReady : t.onboarding.waFollowInstructions}
                </p>
              </div>
              <Badge variant={isConnected ? 'success' : isConnecting ? 'warning' : 'error'}>
                {status ?? 'VERIFICANDO'}
              </Badge>
              <button
                type="button"
                onClick={fetchStatus}
                className="ml-1 text-neutral-400 hover:text-neutral-600"
              >
                <RefreshCw className="h-4 w-4" />
              </button>
            </div>

            {error && (
              <p className="rounded-md bg-amber-50 px-3 py-2 text-sm text-amber-700">{error}</p>
            )}

            {/* Instrucciones si no está conectado */}
            {!isConnected && (
              <div className="rounded-lg border border-dashed border-neutral-200 bg-neutral-50 p-4">
                <MessageCircle className="mb-2 h-6 w-6 text-neutral-400" />
                <p className="mb-1 text-sm font-medium text-neutral-700">{t.onboarding.howToConnect}</p>
                <ol className="mb-3 list-decimal pl-4 text-xs text-neutral-500 space-y-1">
                  <li>{t.onboarding.waStep1}</li>
                  <li>{t.onboarding.waStep2}</li>
                  <li>{t.onboarding.waStep3}</li>
                  <li>{t.onboarding.waStep4}</li>
                </ol>
                <Button
                  variant="secondary"
                  size="sm"
                  onClick={() => window.open(evolutionUrl, '_blank')}
                >
                  {t.onboarding.openEvolutionApi}
                </Button>
              </div>
            )}

            {isConnected && (
              <div className="rounded-lg bg-emerald-50 px-4 py-3">
                <p className="text-sm text-emerald-800">{t.onboarding.waAiActive}</p>
              </div>
            )}
          </div>
        )}
      </Card>

      <div className="mt-4 flex gap-3">
        <Button variant="secondary" onClick={() => router.push('/onboarding/schedule')}>
          {t.onboarding.backButton}
        </Button>
        <Button
          className="flex-1"
          onClick={() => router.push('/onboarding/done')}
        >
          {isConnected ? t.onboarding.continueButton : t.onboarding.skipForNow}
        </Button>
      </div>
    </div>
  );
}
