'use client';

import Link from 'next/link';
import { ArrowRight } from 'lucide-react';
import { useTranslations } from '@/lib/i18n';
import { PhoneMockup } from './PhoneMockup';

export function Hero() {
  const t = useTranslations();
  const l = t.landing;

  return (
    <section className="relative overflow-hidden pb-16 pt-32 md:pb-24 md:pt-40">
      {/* Soft background tint */}
      <div
        aria-hidden
        className="pointer-events-none absolute inset-0"
        style={{
          background:
            'radial-gradient(ellipse 65% 50% at 75% 30%, rgba(99,102,241,0.07) 0%, transparent 60%), radial-gradient(ellipse 50% 60% at 10% 70%, rgba(37,211,102,0.05) 0%, transparent 55%)',
        }}
      />

      <div className="relative mx-auto max-w-6xl px-6">
        <div className="grid items-center gap-12 md:grid-cols-[1fr_auto]">
          <div>
            {/* Badge */}
            <div className="inline-flex items-center gap-2 rounded-full border border-primary-100 bg-primary-50 px-3.5 py-1.5 text-[11px] font-semibold uppercase tracking-wider text-primary-700">
              <span className="relative flex h-2 w-2">
                <span className="absolute inset-0 animate-ping rounded-full bg-success opacity-60" />
                <span className="relative inline-flex h-2 w-2 rounded-full bg-success" />
              </span>
              {l.heroBadge}
            </div>

            {/* Title */}
            <h1 className="mt-7 font-serif font-semibold leading-[0.95] tracking-tight text-lp-ink text-[clamp(48px,7vw,84px)]">
              {l.heroTitle1}
              <br />
              {l.heroTitle2}
              <br />
              <span className="italic text-primary-600">{l.heroTitle3}</span>
            </h1>

            {/* Sub */}
            <p className="mt-7 max-w-md text-lg leading-relaxed text-lp-ink-muted">
              {l.heroSub}
            </p>

            {/* CTAs */}
            <div className="mt-9 flex flex-wrap items-center gap-3">
              <Link
                href="/waitlist"
                className="inline-flex items-center gap-2 rounded-full bg-primary-600 px-7 py-3.5 text-base font-semibold text-white shadow-lg shadow-primary-600/20 transition hover:bg-primary-700 hover:shadow-xl hover:shadow-primary-600/30"
              >
                {l.heroCtaPrimary}
                <ArrowRight size={16} />
              </Link>
              <a
                href="#como-funciona"
                className="inline-flex items-center gap-2 rounded-full border border-lp-border bg-white px-6 py-3.5 text-base font-medium text-lp-ink transition hover:border-lp-border-strong hover:bg-lp-bg-subtle"
              >
                {l.heroCtaSecondary}
              </a>
            </div>

            {/* Note */}
            <p className="mt-6 text-sm text-lp-ink-muted">{l.heroNote}</p>
          </div>

          {/* Phone */}
          <div className="hidden justify-center md:flex">
            <PhoneMockup />
          </div>
        </div>
      </div>
    </section>
  );
}
