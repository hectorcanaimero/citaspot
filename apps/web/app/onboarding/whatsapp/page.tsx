'use client';

// Paso 3 del onboarding: conectar WhatsApp Business.
// Muestra QR code inline y polling de estado cada 5s.
import { useState, useEffect, useRef } from 'react';
import { useRouter } from 'next/navigation';
import { CheckCircle2, XCircle, RefreshCw, QrCode } from 'lucide-react';
import { Button }  from '@/components/ui/button';
import { Card }    from '@/components/ui/card';
import { Badge }   from '@/components/ui/badge';
import { Spinner } from '@/components/ui/spinner';
import { whatsapp, APIError } from '@/lib/api';
import { useTranslations } from '@/lib/i18n';

export default function OnboardingWhatsAppPage() {
  const t      = useTranslations();
  const router = useRouter();
  const [status, setStatus] = useState<'CONNECTED' | 'DISCONNECTED' | 'CONNECTING' | null>(null);
  const [qrCode, setQrCode] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [connectError, setConnectError] = useState(false);

  const [qrCountdown, setQrCountdown] = useState(30);
  const statusIntervalRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const countdownRef      = useRef<ReturnType<typeof setInterval> | null>(null);
  const waitForQRRef      = useRef<ReturnType<typeof setInterval> | null>(null);
  // qrRef permite a fetchStatus ignorar DISCONNECTED transitorio de Evolution
  const qrRef             = useRef<string | null>(null);

  async function fetchStatus() {
    try {
      const res = await whatsapp.getStatus();
      setError('');
      if (res.status === 'CONNECTED') {
        setStatus('CONNECTED');
        setQrCode(null); qrRef.current = null;
        if (countdownRef.current)  { clearInterval(countdownRef.current);  countdownRef.current = null; }
        if (waitForQRRef.current)  { clearInterval(waitForQRRef.current);  waitForQRRef.current = null; }
      } else if (res.status === 'DISCONNECTED' && qrRef.current) {
        // Ignorar: Evolution puede reportar DISCONNECTED transitoriamente mientras genera QR
      } else {
        setStatus(res.status);
      }
    } catch (err) {
      setError(err instanceof APIError ? err.message : t.onboarding.waQueryError);
    } finally {
      setLoading(false);
    }
  }

  function startCountdown() {
    if (countdownRef.current) clearInterval(countdownRef.current);
    setQrCountdown(30);
    let remaining = 30;
    countdownRef.current = setInterval(async () => {
      remaining -= 1;
      setQrCountdown(remaining);
      if (remaining <= 0) {
        clearInterval(countdownRef.current!);
        countdownRef.current = null;
        // Pedir QR fresco sin reconectar
        try {
          const fresh = await whatsapp.getQR();
          if (fresh?.qr) {
            setQrCode(fresh.qr); qrRef.current = fresh.qr;
            startCountdown();
            return;
          }
        } catch { /* ignorar */ }
        // Si getQR falla, reconectar completo
        void initConnection();
      }
    }, 1000);
  }

  async function initConnection() {
    setConnectError(false);
    setLoading(true);
    setQrCode(null); qrRef.current = null;
    if (countdownRef.current) { clearInterval(countdownRef.current); countdownRef.current = null; }
    if (waitForQRRef.current) { clearInterval(waitForQRRef.current); waitForQRRef.current = null; }
    try {
      await whatsapp.connect();
      // Polling de 1s hasta recibir QR, luego arranca countdown
      let attempts = 0;
      waitForQRRef.current = setInterval(async () => {
        attempts++;
        try {
          const res = await whatsapp.getQR();
          if (res?.qr) {
            clearInterval(waitForQRRef.current!); waitForQRRef.current = null;
            setQrCode(res.qr); qrRef.current = res.qr;
            startCountdown();
          }
        } catch { /* ignorar */ }
        if (attempts > 20) { clearInterval(waitForQRRef.current!); waitForQRRef.current = null; }
      }, 1000);
    } catch {
      setConnectError(true);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    initConnection();
    // Status polling cada 5s
    statusIntervalRef.current = setInterval(fetchStatus, 5000);
    // Fetch status inicial
    fetchStatus();

    return () => {
      if (statusIntervalRef.current) clearInterval(statusIntervalRef.current);
      if (countdownRef.current)      clearInterval(countdownRef.current);
      if (waitForQRRef.current)      clearInterval(waitForQRRef.current);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const isConnected = status === 'CONNECTED';

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
        ) : connectError ? (
          <div className="flex flex-col items-center gap-4 py-8">
            <XCircle className="h-10 w-10 text-red-400" />
            <p className="text-sm text-neutral-700">{t.onboarding.waConnectError}</p>
            <Button variant="secondary" size="sm" onClick={initConnection}>
              <RefreshCw className="mr-2 h-4 w-4" />
              {t.onboarding.waRetryQR}
            </Button>
          </div>
        ) : (
          <div className="flex flex-col gap-4">
            {/* Estado actual */}
            <div className="flex items-center gap-3">
              {isConnected ? (
                <CheckCircle2 className="h-8 w-8 text-emerald-500 flex-shrink-0" />
              ) : (
                <QrCode className="h-8 w-8 text-neutral-400 flex-shrink-0" />
              )}
              <div className="flex-1">
                <p className="text-sm font-medium text-neutral-900">
                  {isConnected
                    ? t.onboarding.waConnected
                    : status === 'CONNECTING'
                    ? t.onboarding.waConnecting
                    : t.onboarding.waDisconnected}
                </p>
                <p className="text-xs text-neutral-500">
                  {isConnected ? t.onboarding.waAiReady : t.onboarding.waFollowInstructions}
                </p>
              </div>
              <Badge variant={isConnected ? 'success' : status === 'CONNECTING' ? 'warning' : 'error'}>
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

            {/* QR Code inline */}
            {!isConnected && (
              <div className="flex flex-col items-center gap-4 rounded-lg border border-dashed border-neutral-200 bg-neutral-50 p-6">
                {qrCode ? (
                  <>
                    <div className="relative">
                      <div className="rounded-xl bg-white p-4 shadow-sm">
                        <img
                          src={qrCode}
                          alt="QR code para vincular WhatsApp"
                          className="h-64 w-64"
                        />
                      </div>
                      {/* Countdown badge */}
                      <div className="absolute -bottom-3 left-1/2 -translate-x-1/2 flex items-center gap-1.5 rounded-full bg-white border border-neutral-200 px-2.5 py-1 shadow-sm">
                        <div
                          className={`h-2 w-2 rounded-full ${qrCountdown <= 5 ? 'bg-red-400 animate-pulse' : 'bg-emerald-400'}`}
                        />
                        <span className={`text-xs font-semibold tabular-nums ${qrCountdown <= 5 ? 'text-red-500' : 'text-neutral-600'}`}>
                          {qrCountdown}s
                        </span>
                      </div>
                    </div>
                    <p className="text-center text-sm text-neutral-600 mt-2">
                      {t.onboarding.waScanQR}
                    </p>
                  </>
                ) : (
                  <div className="flex flex-col items-center gap-3 py-4">
                    <Spinner />
                    <p className="text-sm text-neutral-500">{t.onboarding.waGeneratingQR}</p>
                  </div>
                )}
                <Button
                  variant="secondary"
                  size="sm"
                  onClick={initConnection}
                >
                  <RefreshCw className="mr-2 h-4 w-4" />
                  {t.onboarding.waRetryQR}
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
