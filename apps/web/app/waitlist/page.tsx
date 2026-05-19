'use client';

// Página pública /waitlist — pre-launch.
// Look & feel: alineado con LandingPage.tsx (Cormorant Garamond + Plus Jakarta Sans,
// paleta cream/gold/bg sobre fondo --bg). Mobile-first, touch targets >= 44px.

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { useForm } from 'react-hook-form';
import { z } from 'zod';
import { ArrowRight, ArrowLeft, Check, Sparkles } from 'lucide-react';
import { waitlist as waitlistApi, APIError } from '@/lib/api';
import { useTranslations } from '@/lib/i18n';

const SUCCESS_KEY = 'citaspot_waitlist_joined';

// Schema con zod; validamos manualmente vía formState (sin instalar @hookform/resolvers).
function buildSchema(messages: { email: string; business: string }) {
  return z.object({
    email: z.string().trim().email({ message: messages.email }),
    businessName: z.string().trim().min(1, { message: messages.business }),
  });
}

interface FormValues {
  email: string;
  businessName: string;
}

export default function WaitlistPage() {
  const t = useTranslations();
  const w = t.waitlist;
  const [submitState, setSubmitState] = useState<'idle' | 'submitting' | 'success' | 'error'>('idle');
  const [errorMsg, setErrorMsg] = useState<string>('');

  const schema = buildSchema({ email: w.errorInvalidEmail, business: w.errorBusinessRequired });

  const {
    register,
    handleSubmit,
    formState: { errors },
    setError,
  } = useForm<FormValues>({
    mode: 'onSubmit',
    defaultValues: { email: '', businessName: '' },
  });

  // Si ya se registró en esta sesión, mostrar success directamente.
  useEffect(() => {
    if (typeof window === 'undefined') return;
    if (sessionStorage.getItem(SUCCESS_KEY) === '1') {
      setSubmitState('success');
    }
  }, []);

  async function onSubmit(values: FormValues) {
    // Validación zod manual (evita dependencia @hookform/resolvers).
    const parsed = schema.safeParse(values);
    if (!parsed.success) {
      for (const issue of parsed.error.issues) {
        const field = issue.path[0];
        if (field === 'email' || field === 'businessName') {
          setError(field, { type: 'manual', message: issue.message });
        }
      }
      return;
    }

    setSubmitState('submitting');
    setErrorMsg('');
    try {
      await waitlistApi.join({
        email: parsed.data.email,
        businessName: parsed.data.businessName,
      });
      if (typeof window !== 'undefined') {
        sessionStorage.setItem(SUCCESS_KEY, '1');
      }
      setSubmitState('success');
    } catch (err) {
      if (err instanceof APIError && err.code === 'duplicate') {
        setErrorMsg(w.errorDuplicate);
      } else {
        setErrorMsg(w.errorGeneric);
      }
      setSubmitState('error');
    }
  }

  return (
    <>
      <style dangerouslySetInnerHTML={{ __html: `
        @import url('https://fonts.googleapis.com/css2?family=Cormorant+Garamond:ital,wght@0,500;0,600;0,700;1,500;1,600&family=Plus+Jakarta+Sans:wght@400;500;600;700&display=swap');

        :root {
          --bg: #0C0A08;
          --bg-card: #131009;
          --gold: #C9A96A;
          --gold-lt: #E4C98B;
          --cream: #F2EDE4;
          --muted: #8A7E74;
          --green: #25D366;
          --border: rgba(201,169,106,0.18);
          --border-s: rgba(242,237,228,0.07);
        }

        .wl { background: var(--bg); color: var(--cream); font-family: 'Plus Jakarta Sans', system-ui, sans-serif; min-height: 100vh; overflow-x: hidden; display: flex; flex-direction: column; }
        .wl *, .wl *::before, .wl *::after { box-sizing: border-box; }

        /* Grain (igual que landing) */
        .wl::before {
          content: '';
          position: fixed; inset: 0; pointer-events: none; z-index: 9999;
          background-image: url("data:image/svg+xml,%3Csvg viewBox='0 0 256 256' xmlns='http://www.w3.org/2000/svg'%3E%3Cfilter id='n'%3E%3CfeTurbulence type='fractalNoise' baseFrequency='0.9' numOctaves='4' stitchTiles='stitch'/%3E%3C/filter%3E%3Crect width='100%25' height='100%25' filter='url(%23n)'/%3E%3C/svg%3E");
          opacity: 0.032;
        }

        .wl-serif { font-family: 'Cormorant Garamond', Georgia, serif; }
        .wl-h     { font-family: 'Cormorant Garamond', Georgia, serif; font-weight: 600; font-size: clamp(40px, 6.5vw, 64px); line-height: 1.04; letter-spacing: -0.01em; }
        .wl-gold  { color: var(--gold); }

        .wl-c { max-width: 560px; width: 100%; margin: 0 auto; padding: 0 24px; }

        .wl-nav { padding: 22px 0; border-bottom: 1px solid var(--border-s); }
        .wl-nav-c { max-width: 1160px; margin: 0 auto; padding: 0 24px; display: flex; align-items: center; justify-content: space-between; }
        .wl-logo { font-family: 'Cormorant Garamond', Georgia, serif; font-weight: 700; font-size: 26px; color: var(--cream); letter-spacing: -0.01em; text-decoration: none; }
        .wl-logo span { color: var(--gold); }
        .wl-back { display: inline-flex; align-items: center; gap: 6px; font-size: 14px; color: var(--muted); text-decoration: none; transition: color .2s; padding: 10px 4px; min-height: 44px; }
        .wl-back:hover { color: var(--cream); }

        .wl-pill { display: inline-flex; align-items: center; gap: 7px; background: rgba(201,169,106,.07); border: 1px solid var(--border); color: var(--gold); font-size: 11px; font-weight: 700; letter-spacing: .08em; text-transform: uppercase; padding: 6px 14px; border-radius: 100px; }

        .wl-card { background: var(--bg-card); border: 1px solid var(--border-s); border-radius: 20px; padding: 32px; }
        @media (min-width: 640px) { .wl-card { padding: 40px; } }

        .wl-field { display: flex; flex-direction: column; gap: 8px; }
        .wl-label { font-size: 13px; font-weight: 600; color: var(--cream); letter-spacing: .01em; }
        .wl-input {
          width: 100%;
          background: rgba(242,237,228,.03);
          border: 1px solid var(--border-s);
          color: var(--cream);
          font-family: 'Plus Jakarta Sans', sans-serif;
          font-size: 16px; /* >= 16px evita zoom iOS */
          padding: 14px 16px;
          border-radius: 12px;
          min-height: 48px;
          transition: border-color .2s, background .2s;
          outline: none;
        }
        .wl-input::placeholder { color: var(--muted); opacity: .55; }
        .wl-input:hover { border-color: var(--border); }
        .wl-input:focus { border-color: var(--gold); background: rgba(242,237,228,.05); }
        .wl-input.is-error { border-color: #E55A5A; }
        .wl-field-err { font-size: 13px; color: #E89797; margin-top: 2px; }

        .wl-submit {
          display: inline-flex; align-items: center; justify-content: center; gap: 8px;
          width: 100%;
          background: var(--gold); color: #0C0A08;
          font-family: 'Plus Jakarta Sans', sans-serif;
          font-size: 16px; font-weight: 700;
          padding: 16px 26px; border-radius: 100px; border: none;
          cursor: pointer;
          min-height: 52px;
          letter-spacing: .01em;
          transition: background .2s, transform .2s, box-shadow .2s;
        }
        .wl-submit:hover:not(:disabled) { background: var(--gold-lt); transform: translateY(-1px); box-shadow: 0 8px 28px rgba(201,169,106,.28); }
        .wl-submit:disabled { opacity: .65; cursor: not-allowed; }

        .wl-alert {
          background: rgba(229,90,90,.08);
          border: 1px solid rgba(229,90,90,.32);
          color: #F2B5B5;
          font-size: 14px;
          padding: 12px 14px;
          border-radius: 10px;
          line-height: 1.5;
        }

        .wl-success-ic {
          width: 64px; height: 64px; border-radius: 50%;
          background: rgba(37,211,102,.13); border: 1px solid rgba(37,211,102,.32);
          display: flex; align-items: center; justify-content: center;
          margin: 0 auto 22px;
          color: var(--green);
        }

        .wl-foot { padding: 28px 0; border-top: 1px solid var(--border-s); margin-top: auto; }
        .wl-foot-c { max-width: 1160px; margin: 0 auto; padding: 0 24px; text-align: center; font-size: 13px; color: var(--muted); }

        .wl-spin { width: 16px; height: 16px; border-radius: 50%; border: 2px solid rgba(12,10,8,.25); border-top-color: #0C0A08; animation: wl-rot .8s linear infinite; }
        @keyframes wl-rot { to { transform: rotate(360deg); } }
      ` }} />

      <div className="wl">
        {/* Nav simple */}
        <nav className="wl-nav">
          <div className="wl-nav-c">
            <Link href="/" className="wl-logo">CitaSpot<span>.</span></Link>
            <Link href="/" className="wl-back">
              <ArrowLeft size={14} /> {w.backToHome}
            </Link>
          </div>
        </nav>

        {/* Main */}
        <main style={{ flex: 1, display: 'flex', alignItems: 'center', padding: '64px 0' }}>
          <div className="wl-c">
            <div style={{ textAlign: 'center', marginBottom: 36 }}>
              <div className="wl-pill" style={{ marginBottom: 22 }}>
                <Sparkles size={11} /> {w.pageTitle}
              </div>
              <h1 className="wl-h" style={{ marginBottom: 18 }}>
                {submitState === 'success' ? (
                  <span className="wl-gold" style={{ fontStyle: 'italic' }}>
                    {w.successTitle}
                  </span>
                ) : (
                  w.pageTitle
                )}
              </h1>
              <p style={{ fontSize: 17, color: 'var(--muted)', lineHeight: 1.65, maxWidth: 460, margin: '0 auto' }}>
                {submitState === 'success' ? w.successDesc : w.pageSub}
              </p>
            </div>

            <div className="wl-card">
              {submitState === 'success' ? (
                <div style={{ textAlign: 'center', padding: '8px 0 4px' }}>
                  <div className="wl-success-ic">
                    <Check size={28} strokeWidth={2.5} />
                  </div>
                  <p className="wl-serif" style={{ fontSize: 22, color: 'var(--cream)', marginBottom: 8, fontStyle: 'italic' }}>
                    {w.successDesc}
                  </p>
                  <div style={{ marginTop: 28 }}>
                    <Link
                      href="/"
                      className="wl-back"
                      style={{ justifyContent: 'center', fontSize: 14 }}
                    >
                      <ArrowLeft size={14} /> {w.backToHome}
                    </Link>
                  </div>
                </div>
              ) : (
                <form onSubmit={handleSubmit(onSubmit)} noValidate style={{ display: 'flex', flexDirection: 'column', gap: 20 }}>
                  <div className="wl-field">
                    <label htmlFor="wl-email" className="wl-label">
                      {w.emailLabel}
                    </label>
                    <input
                      id="wl-email"
                      type="email"
                      inputMode="email"
                      autoComplete="email"
                      placeholder={w.emailPlaceholder}
                      className={`wl-input ${errors.email ? 'is-error' : ''}`}
                      aria-invalid={!!errors.email}
                      aria-describedby={errors.email ? 'wl-email-err' : undefined}
                      disabled={submitState === 'submitting'}
                      {...register('email')}
                    />
                    {errors.email && (
                      <span id="wl-email-err" className="wl-field-err" role="alert">
                        {errors.email.message}
                      </span>
                    )}
                  </div>

                  <div className="wl-field">
                    <label htmlFor="wl-business" className="wl-label">
                      {w.businessLabel}
                    </label>
                    <input
                      id="wl-business"
                      type="text"
                      autoComplete="organization"
                      placeholder={w.businessPlaceholder}
                      className={`wl-input ${errors.businessName ? 'is-error' : ''}`}
                      aria-invalid={!!errors.businessName}
                      aria-describedby={errors.businessName ? 'wl-business-err' : undefined}
                      disabled={submitState === 'submitting'}
                      {...register('businessName')}
                    />
                    {errors.businessName && (
                      <span id="wl-business-err" className="wl-field-err" role="alert">
                        {errors.businessName.message}
                      </span>
                    )}
                  </div>

                  {submitState === 'error' && errorMsg && (
                    <div className="wl-alert" role="alert">
                      {errorMsg}
                    </div>
                  )}

                  <button
                    type="submit"
                    className="wl-submit"
                    disabled={submitState === 'submitting'}
                  >
                    {submitState === 'submitting' ? (
                      <>
                        <span className="wl-spin" /> {w.submitting}
                      </>
                    ) : (
                      <>
                        {w.submitBtn} <ArrowRight size={16} />
                      </>
                    )}
                  </button>
                </form>
              )}
            </div>
          </div>
        </main>

        {/* Footer */}
        <footer className="wl-foot">
          <div className="wl-foot-c">© 2025 CitaSpot</div>
        </footer>
      </div>
    </>
  );
}
