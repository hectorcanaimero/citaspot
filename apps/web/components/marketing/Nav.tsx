'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import Image from 'next/image';
import { ArrowRight, Menu, X } from 'lucide-react';
import { useTranslations } from '@/lib/i18n';

export function Nav() {
  const t = useTranslations();
  const l = t.landing;
  const [scrolled, setScrolled] = useState(false);
  const [menuOpen, setMenuOpen] = useState(false);

  useEffect(() => {
    const onScroll = () => setScrolled(window.scrollY > 24);
    window.addEventListener('scroll', onScroll, { passive: true });
    return () => window.removeEventListener('scroll', onScroll);
  }, []);

  useEffect(() => {
    if (menuOpen) document.body.style.overflow = 'hidden';
    else document.body.style.overflow = '';
    return () => { document.body.style.overflow = ''; };
  }, [menuOpen]);

  return (
    <>
      <nav
        className={[
          'fixed inset-x-0 top-0 z-40 transition-all duration-300',
          scrolled
            ? 'border-b border-lp-border bg-white/85 py-3 backdrop-blur-md'
            : 'border-b border-transparent py-5',
        ].join(' ')}
      >
        <div className="mx-auto flex max-w-6xl items-center justify-between px-6">
          <Link href="/" className="flex items-center gap-2" aria-label="CitaSpot">
            <Image
              src="/logo.jpeg"
              alt="CitaSpot"
              width={160}
              height={56}
              priority
              className="h-9 w-auto"
            />
          </Link>

          {/* Desktop nav */}
          <div className="hidden items-center gap-1 md:flex">
            <a
              href="#para-quien"
              className="rounded-full px-4 py-2 text-sm font-medium text-lp-ink-soft transition hover:text-lp-ink"
            >
              {l.navForWho}
            </a>
            <a
              href="#beneficios"
              className="rounded-full px-4 py-2 text-sm font-medium text-lp-ink-soft transition hover:text-lp-ink"
            >
              {l.navBenefits}
            </a>
            <a
              href="#como-funciona"
              className="rounded-full px-4 py-2 text-sm font-medium text-lp-ink-soft transition hover:text-lp-ink"
            >
              {l.navHowItWorks}
            </a>
            <Link
              href="/login"
              className="rounded-full px-4 py-2 text-sm font-medium text-lp-ink-soft transition hover:text-lp-ink"
            >
              {l.navLogin}
            </Link>
            <Link
              href="/waitlist"
              className="ml-2 inline-flex items-center gap-1.5 rounded-full bg-primary-600 px-5 py-2.5 text-sm font-semibold text-white shadow-md shadow-primary-600/20 transition hover:bg-primary-700 hover:shadow-lg hover:shadow-primary-600/30"
            >
              {l.navCta}
              <ArrowRight size={14} />
            </Link>
          </div>

          {/* Mobile button */}
          <button
            type="button"
            onClick={() => setMenuOpen(true)}
            className="rounded-full p-2 text-lp-ink md:hidden"
            aria-label={l.openMenu}
          >
            <Menu size={22} />
          </button>
        </div>
      </nav>

      {/* Mobile menu */}
      {menuOpen && (
        <div className="fixed inset-0 z-50 flex flex-col bg-white md:hidden">
          <div className="flex items-center justify-between border-b border-lp-border px-6 py-5">
            <Image src="/logo.jpeg" alt="CitaSpot" width={140} height={48} className="h-9 w-auto" />
            <button
              type="button"
              onClick={() => setMenuOpen(false)}
              className="rounded-full p-2 text-lp-ink"
              aria-label={l.closeMenu}
            >
              <X size={22} />
            </button>
          </div>
          <div className="flex flex-1 flex-col gap-2 px-6 py-8">
            <a
              href="#para-quien"
              onClick={() => setMenuOpen(false)}
              className="font-serif text-3xl font-medium text-lp-ink"
            >
              {l.navForWho}
            </a>
            <a
              href="#beneficios"
              onClick={() => setMenuOpen(false)}
              className="font-serif text-3xl font-medium text-lp-ink"
            >
              {l.navBenefits}
            </a>
            <a
              href="#como-funciona"
              onClick={() => setMenuOpen(false)}
              className="font-serif text-3xl font-medium text-lp-ink"
            >
              {l.navHowItWorks}
            </a>
            <Link
              href="/login"
              onClick={() => setMenuOpen(false)}
              className="font-serif text-3xl font-medium text-lp-ink"
            >
              {l.navLogin}
            </Link>
          </div>
          <div className="border-t border-lp-border px-6 py-6">
            <Link
              href="/waitlist"
              onClick={() => setMenuOpen(false)}
              className="inline-flex w-full items-center justify-center gap-1.5 rounded-full bg-primary-600 px-6 py-3.5 text-base font-semibold text-white"
            >
              {l.navCta}
              <ArrowRight size={16} />
            </Link>
          </div>
        </div>
      )}
    </>
  );
}
