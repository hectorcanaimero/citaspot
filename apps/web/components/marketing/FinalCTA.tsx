'use client';

import { useState, FormEvent } from 'react';
import { useRouter } from 'next/navigation';
import Link from 'next/link';
import { ArrowRight, Check } from 'lucide-react';
import { useTranslations } from '@/lib/i18n';

export function FinalCTA() {
  const t = useTranslations();
  const l = t.landing;
  const router = useRouter();
  const [email, setEmail] = useState('');
  const bullets = l.ctaBullets as readonly string[];

  function onSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const trimmed = email.trim();
    const qs = trimmed ? `?email=${encodeURIComponent(trimmed)}` : '';
    router.push(`/waitlist${qs}`);
  }

  return (
    <section className="relative overflow-hidden bg-gradient-to-br from-primary-950 via-primary-900 to-primary-800 py-24 text-white md:py-32">
      <div
        aria-hidden
        className="pointer-events-none absolute inset-0 opacity-50"
        style={{
          background:
            'radial-gradient(ellipse 50% 60% at 50% 40%, rgba(199,210,254,0.18) 0%, transparent 60%)',
        }}
      />

      <div className="relative mx-auto max-w-3xl px-6 text-center">
        <div className="inline-flex items-center gap-1.5 rounded-full border border-white/15 bg-white/5 px-3.5 py-1.5 text-[11px] font-semibold uppercase tracking-wider text-white/85">
          <span className="h-1.5 w-1.5 rounded-full bg-emerald-400" />
          {l.ctaPill}
        </div>
        <h2 className="mt-6 font-serif font-semibold leading-[1.05] tracking-tight text-[clamp(38px,5vw,64px)]">
          {l.ctaTitle}
          <br />
          <span className="italic text-primary-200">{l.ctaTitleHighlight}</span>
        </h2>
        <p className="mt-5 text-lg text-white/75">{l.ctaSub}</p>

        <form
          onSubmit={onSubmit}
          className="mx-auto mt-9 flex w-full max-w-md flex-col gap-2 sm:flex-row"
        >
          <input
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            placeholder={l.ctaEmailPlaceholder}
            className="flex-1 rounded-full border border-white/15 bg-white/10 px-5 py-3.5 text-base text-white placeholder:text-white/50 focus:border-white/30 focus:outline-none focus:ring-2 focus:ring-white/20"
            aria-label={l.ctaEmailPlaceholder}
            required
          />
          <button
            type="submit"
            className="inline-flex items-center justify-center gap-2 rounded-full bg-white px-6 py-3.5 text-base font-semibold text-primary-700 shadow-lg shadow-black/20 transition hover:bg-primary-50"
          >
            {l.ctaSubmitBtn}
            <ArrowRight size={16} />
          </button>
        </form>

        <ul className="mx-auto mt-7 flex max-w-md flex-wrap justify-center gap-x-5 gap-y-2 text-sm text-white/75">
          {bullets.map((b, i) => (
            <li key={i} className="inline-flex items-center gap-1.5">
              <Check size={14} className="text-emerald-400" />
              {b}
            </li>
          ))}
        </ul>

        <div className="mt-10">
          <Link
            href="/login"
            className="text-sm font-medium text-white/70 underline-offset-4 hover:text-white hover:underline"
          >
            {l.ctaLoginBtn}
          </Link>
        </div>
      </div>
    </section>
  );
}
