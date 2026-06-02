'use client';

import Link from 'next/link';
import Image from 'next/image';
import { useTranslations } from '@/lib/i18n';

export function Footer() {
  const t = useTranslations();
  const l = t.landing;

  return (
    <footer className="border-t border-lp-border bg-white py-14">
      <div className="mx-auto max-w-6xl px-6">
        <div className="grid gap-10 md:grid-cols-[1fr_auto] md:items-end">
          <div>
            <Image
              src="/logo-vertical.jpeg"
              alt="CitaSpot"
              width={120}
              height={120}
              className="h-20 w-auto"
            />
            <p className="mt-4 text-sm text-lp-ink-muted">{l.footerTagline}</p>
          </div>

          <nav className="flex flex-wrap items-center gap-6 text-sm">
            <a href="#para-quien" className="text-lp-ink-soft transition hover:text-lp-ink">
              {l.navForWho}
            </a>
            <a href="#beneficios" className="text-lp-ink-soft transition hover:text-lp-ink">
              {l.navBenefits}
            </a>
            <a href="#como-funciona" className="text-lp-ink-soft transition hover:text-lp-ink">
              {l.navHowItWorks}
            </a>
            <Link href="/login" className="text-lp-ink-soft transition hover:text-lp-ink">
              {l.navLogin}
            </Link>
            <Link href="/waitlist" className="text-lp-ink-soft transition hover:text-lp-ink">
              {l.navCta}
            </Link>
          </nav>
        </div>

        <div className="mt-10 flex flex-wrap items-center justify-between gap-4 border-t border-lp-border pt-6 text-xs text-lp-ink-muted">
          <div className="flex flex-wrap items-center gap-5">
            <Link href="/privacy" className="hover:text-lp-ink">
              {l.footerPrivacy}
            </Link>
            <Link href="/terms" className="hover:text-lp-ink">
              {l.footerTerms}
            </Link>
            <a href="mailto:hola@citaspot.com" className="hover:text-lp-ink">
              hola@citaspot.com
            </a>
          </div>
          <span>{l.footerCopyright}</span>
        </div>
      </div>
    </footer>
  );
}
