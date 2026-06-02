'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { ArrowRight } from 'lucide-react';
import { useTranslations } from '@/lib/i18n';

export function StickyMobileCTA() {
  const t = useTranslations();
  const l = t.landing;
  const [visible, setVisible] = useState(false);

  useEffect(() => {
    const onScroll = () => setVisible(window.scrollY > 600);
    window.addEventListener('scroll', onScroll, { passive: true });
    onScroll();
    return () => window.removeEventListener('scroll', onScroll);
  }, []);

  return (
    <div
      className={[
        'fixed inset-x-0 bottom-0 z-30 border-t border-lp-border bg-white/95 px-4 py-3 backdrop-blur-md transition-transform duration-300 md:hidden',
        visible ? 'translate-y-0' : 'translate-y-full',
      ].join(' ')}
    >
      <Link
        href="/waitlist"
        className="inline-flex w-full items-center justify-center gap-1.5 rounded-full bg-primary-600 px-6 py-3 text-sm font-semibold text-white shadow-lg shadow-primary-600/30"
      >
        {l.navCta}
        <ArrowRight size={15} />
      </Link>
    </div>
  );
}
