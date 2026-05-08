'use client';

import { useState, useEffect, useRef, useCallback } from 'react';
import { MessageCircle, CheckCircle2, XCircle, RefreshCw, Loader2, Unplug, Smartphone, Settings2, QrCode } from 'lucide-react';
import { Button }  from '@/components/ui/button';
import { Card, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge }   from '@/components/ui/badge';
import { Spinner } from '@/components/ui/spinner';
import { whatsapp, APIError } from '@/lib/api';
import { useTranslations } from '@/lib/i18n';

type WAStatus = 'CONNECTED' | 'DISCONNECTED' | 'CONNECTING';

const QR_POLL_MS     = 2_000;
const STATUS_POLL_MS = 5_000;

export default function WhatsAppPage() {
  const t = useTranslations();
  const [status,     setStatus]     = useState<WAStatus | null>(null);
  const [instance,   setInstance]   = useState('');
  const [qr,         setQr]         = useState('');
  const [loading,    setLoading]    = useState(true);
  const [connecting, setConnecting] = useState(false);
  const [error,      setError]      = useState('');

  const qrIntervalRef     = useRef<ReturnType<typeof setInterval> | null>(null);
  const statusIntervalRef = useRef<ReturnType<typeof setInterval> | null>(null);
  // Auto-connect on mount: dispara handleConnect una sola vez si entras disconnected.
  // No se resetea ante una desconexion manual ni ante errores → evita loops.
  const hasAutoConnectedRef = useRef(false);

  function stopPolling() {
    if (qrIntervalRef.current)     { clearInterval(qrIntervalRef.current);     qrIntervalRef.current = null; }
    if (statusIntervalRef.current) { clearInterval(statusIntervalRef.current); statusIntervalRef.current = null; }
  }

  const pollQR = useCallback(async () => {
    try {
      const res = await whatsapp.getQR();
      if (res?.qr) setQr(res.qr);
    } catch { /* ignorar */ }
  }, []);

  const pollStatus = useCallback(async () => {
    try {
      const res = await whatsapp.getStatus();
      setStatus(res.status);
      setInstance(res.instance);
      if (res.status === 'CONNECTED') {
        setQr('');
        stopPolling();
      }
    } catch { /* ignorar */ }
  }, []);

  const fetchStatus = useCallback(async (silent = false) => {
    if (!silent) setLoading(true);
    setError('');
    try {
      const res = await whatsapp.getStatus();
      setStatus(res.status);
      setInstance(res.instance);
      if (res.status === 'CONNECTED') {
        setQr('');
        stopPolling();
      }
    } catch (e) {
      setError(e instanceof APIError ? e.message : t.whatsapp.queryError);
      setStatus('DISCONNECTED');
    } finally {
      if (!silent) setLoading(false);
    }
  }, [t]);

  function startPolling() {
    stopPolling();
    qrIntervalRef.current     = setInterval(pollQR, QR_POLL_MS);
    statusIntervalRef.current = setInterval(pollStatus, STATUS_POLL_MS);
  }

  useEffect(() => {
    fetchStatus();
    return () => stopPolling();
  }, [fetchStatus]);

  // Si despues del primer fetch el estado es DISCONNECTED, disparar connect automaticamente.
  // Se ejecuta solo una vez por mount via hasAutoConnectedRef (no loop si connect falla).
  useEffect(() => {
    if (loading) return;
    if (hasAutoConnectedRef.current) return;
    if (status !== 'DISCONNECTED') return;
    if (error) return;
    handleConnect();
  }, [loading, status, error, handleConnect]);

  const handleConnect = useCallback(async () => {
    hasAutoConnectedRef.current = true;
    setConnecting(true);
    setError('');
    setQr('');
    try {
      const res = await whatsapp.connect();
      setStatus('CONNECTING');
      setInstance(res.instance ?? '');
      startPolling();
    } catch (e) {
      setError(e instanceof APIError ? e.message : t.whatsapp.connectError);
    } finally {
      setConnecting(false);
    }
  }, [t]);

  async function handleDisconnect() {
    setConnecting(true);
    setError('');
    try {
      await whatsapp.disconnect();
      setStatus('DISCONNECTED');
      setQr('');
      stopPolling();
    } catch (e) {
      setError(e instanceof APIError ? e.message : t.whatsapp.disconnectError);
    } finally {
      setConnecting(false);
    }
  }

  const isConnected    = status === 'CONNECTED';
  const isConnecting   = status === 'CONNECTING';
  const isDisconnected = status === 'DISCONNECTED' || status === null;

  return (
    <div className="p-6">
      <div className="mb-6">
        <h1 className="text-xl font-semibold text-neutral-900">{t.whatsapp.title}</h1>
        <p className="mt-0.5 text-sm text-neutral-500">{t.whatsapp.description}</p>
      </div>

      <div className="max-w-lg space-y-4">
        {/* ── Estado ─────────────────────────────────────────────────────── */}
        <Card>
          <CardHeader>
            <CardTitle>{t.whatsapp.sessionStatus}</CardTitle>
            <Button variant="ghost" size="sm" onClick={() => fetchStatus()} disabled={loading || connecting}>
              <RefreshCw className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
            </Button>
          </CardHeader>

          {loading ? (
            <div className="flex justify-center py-8"><Spinner /></div>
          ) : (
            <div className="flex flex-col gap-4">
              {/* Estado actual */}
              <div className="flex items-center gap-3">
                {isConnected ? (
                  <CheckCircle2 className="h-8 w-8 text-emerald-500 shrink-0" />
                ) : isConnecting ? (
                  <Loader2 className="h-8 w-8 text-primary-500 animate-spin shrink-0" />
                ) : (
                  <XCircle className="h-8 w-8 text-red-400 shrink-0" />
                )}
                <div className="min-w-0">
                  <p className="text-sm font-medium text-neutral-900">
                    {isConnected ? t.whatsapp.connected : isConnecting ? t.whatsapp.waitingQR : t.whatsapp.disconnected}
                  </p>
                  {instance && (
                    <p className="text-xs text-neutral-500 truncate">{t.whatsapp.instance}: {instance}</p>
                  )}
                </div>
                <Badge
                  variant={isConnected ? 'success' : isConnecting ? 'warning' : 'error'}
                  className="ml-auto shrink-0"
                >
                  {status ?? 'UNKNOWN'}
                </Badge>
              </div>

              {error && (
                <p className="rounded-md bg-red-50 px-3 py-2 text-sm text-red-700">{error}</p>
              )}

              {/* ── Desconectado ─────────────────────────────────────────── */}
              {/* Auto-connect dispara solo, asi que este estado solo se ve si fallo o si el user desconecto manualmente. */}
              {isDisconnected && (
                <div className="rounded-lg border border-dashed border-neutral-200 bg-neutral-50 p-5 text-center">
                  {connecting ? (
                    <>
                      <Loader2 className="mx-auto mb-2 h-10 w-10 text-primary-500 animate-spin" />
                      <p className="text-sm font-medium text-neutral-700">{t.whatsapp.waitingQR}</p>
                    </>
                  ) : (
                    <>
                      <MessageCircle className="mx-auto mb-2 h-10 w-10 text-neutral-300" />
                      <p className="text-sm font-medium text-neutral-700 mb-1">{t.whatsapp.noConnection}</p>
                      <p className="text-xs text-neutral-500 mb-4">{t.whatsapp.noConnectionDesc}</p>
                      <Button onClick={handleConnect} loading={connecting}>
                        {t.whatsapp.connectWA}
                      </Button>
                    </>
                  )}
                </div>
              )}

              {/* ── Conectando — QR inline ───────────────────────────────── */}
              {isConnecting && (
                <div className="flex flex-col items-center gap-3 rounded-lg border border-primary-100 bg-primary-50 p-5">
                  <p className="text-sm font-medium text-primary-800">{t.whatsapp.scanQR}</p>
                  {qr ? (
                    // eslint-disable-next-line @next/next/no-img-element
                    <img
                      src={qr}
                      alt="QR WhatsApp"
                      className="h-52 w-52 rounded-lg border border-primary-200 bg-white p-1"
                    />
                  ) : (
                    <div className="flex h-52 w-52 flex-col items-center justify-center gap-2 rounded-lg border border-primary-200 bg-white">
                      <Spinner />
                      <p className="text-xs text-neutral-400">{t.whatsapp.generatingQR}</p>
                    </div>
                  )}
                  <ol className="w-full space-y-2 mt-2">
                    {[
                      { Icon: Smartphone, text: t.whatsapp.step1 },
                      { Icon: Settings2,  text: t.whatsapp.step2Detail },
                      { Icon: QrCode,     text: t.whatsapp.step3Detail },
                    ].map(({ Icon, text }, i) => (
                      <li key={i} className="flex items-center gap-3 rounded-md bg-white px-3 py-2 border border-primary-100">
                        <span className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-primary-500 text-[11px] font-semibold text-white">
                          {i + 1}
                        </span>
                        <Icon className="h-4 w-4 shrink-0 text-primary-600" />
                        <span className="text-xs text-primary-800">
                          {text.replace(/^\d+\.\s*/, '')}
                        </span>
                      </li>
                    ))}
                  </ol>
                  <p className="text-xs text-primary-500">{t.whatsapp.verifyingAuto}</p>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={handleConnect}
                    loading={connecting}
                    className="text-primary-600"
                  >
                    {t.whatsapp.generateNewQR}
                  </Button>
                </div>
              )}

              {/* ── Conectado ────────────────────────────────────────────── */}
              {isConnected && (
                <div className="flex flex-col gap-3">
                  <div className="rounded-lg bg-emerald-50 px-4 py-3">
                    <p className="text-sm text-emerald-800">{t.whatsapp.aiActive}</p>
                  </div>
                  <Button
                    variant="secondary"
                    size="sm"
                    onClick={handleDisconnect}
                    loading={connecting}
                    className="self-start text-red-600 hover:text-red-700 border-red-200 hover:bg-red-50"
                  >
                    <Unplug className="h-4 w-4 mr-1.5" />
                    {t.whatsapp.disconnect}
                  </Button>
                </div>
              )}
            </div>
          )}
        </Card>

        {/* ── Ayuda ─────────────────────────────────────────────────────────── */}
        <div className="rounded-lg border border-neutral-100 bg-neutral-50 px-4 py-3 text-xs text-neutral-500">
          <p className="font-medium text-neutral-600 mb-1">{t.whatsapp.troubleTitle}</p>
          <ul className="space-y-0.5 list-disc list-inside">
            <li>{t.whatsapp.trouble1}</li>
            <li>{t.whatsapp.trouble2}</li>
            <li>{t.whatsapp.trouble3}</li>
          </ul>
        </div>
      </div>
    </div>
  );
}
