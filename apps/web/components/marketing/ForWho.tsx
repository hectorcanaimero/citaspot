'use client';

import { Sparkles } from 'lucide-react';
import { useTranslations } from '@/lib/i18n';

export function ForWho() {
  const t = useTranslations();
  const l = t.landing;
  const items = l.forWho as readonly string[];

  return (
    <section id="para-quien" className="py-24 md:py-32">
      <div className="mx-auto max-w-6xl px-6">
        <div className="mx-auto max-w-2xl text-center">
          <div className="inline-flex items-center gap-1.5 rounded-full border border-lp-border bg-lp-bg-subtle px-3.5 py-1.5 text-[11px] font-semibold uppercase tracking-wider text-lp-ink-soft">
            <Sparkles size={12} className="text-primary-600" />
            {l.forWhoPill}
          </div>
          <h2 className="mt-6 font-serif font-semibold leading-[1.05] tracking-tight text-lp-ink text-[clamp(36px,4.5vw,56px)]">
            {l.forWhoTitle}
            <span className="italic text-primary-600">{l.forWhoTitleHighlight}</span>
          </h2>
          <p className="mt-5 text-lg text-lp-ink-muted">{l.forWhoSub}</p>
        </div>

        <ul className="mt-14 grid grid-cols-1 gap-x-10 gap-y-3 sm:grid-cols-2 md:grid-cols-3">
          {items.map((item, i) => (
            <li
              key={i}
              className="flex items-center gap-3 border-b border-lp-border py-3 text-[15px] font-medium text-lp-ink"
            >
              <span className="inline-flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-primary-50 text-primary-700">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round">
                  <polyline points="20 6 9 17 4 12" />
                </svg>
              </span>
              {item}
            </li>
          ))}
        </ul>
      </div>
    </section>
  );
}
