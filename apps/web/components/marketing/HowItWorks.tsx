'use client';

import { MessageCircle, Calendar, Share2 } from 'lucide-react';
import { useTranslations } from '@/lib/i18n';

const STEP_ICONS = [MessageCircle, Calendar, Share2];

export function HowItWorks() {
  const t = useTranslations();
  const l = t.landing;
  const steps = l.steps as readonly { title: string; desc: string }[];

  return (
    <section id="como-funciona" className="py-24 md:py-32">
      <div className="mx-auto max-w-6xl px-6">
        <div className="mx-auto max-w-2xl text-center">
          <div className="inline-flex items-center gap-1.5 rounded-full border border-lp-border bg-lp-bg-subtle px-3.5 py-1.5 text-[11px] font-semibold uppercase tracking-wider text-lp-ink-soft">
            <span className="h-1.5 w-1.5 rounded-full bg-primary-600" />
            {l.howPill}
          </div>
          <h2 className="mt-6 font-serif font-semibold leading-[1.05] tracking-tight text-lp-ink text-[clamp(36px,4.5vw,56px)]">
            {l.howTitle}
            <span className="italic text-primary-600">{l.howTitleHighlight}</span>
          </h2>
          <p className="mt-5 text-lg text-lp-ink-muted">{l.howSub}</p>
        </div>

        <ol className="mt-16 grid gap-6 md:grid-cols-3">
          {steps.map((step, i) => {
            const Icon = STEP_ICONS[i];
            return (
              <li
                key={i}
                className="relative overflow-hidden rounded-2xl border border-lp-border bg-white p-7 transition hover:border-lp-border-strong hover:shadow-lg hover:shadow-primary-600/[0.05]"
              >
                <span className="absolute -right-2 -top-4 font-serif text-[90px] font-semibold leading-none text-primary-50 select-none">
                  {i + 1}
                </span>
                <div className="relative">
                  <div className="inline-flex h-11 w-11 items-center justify-center rounded-xl border border-primary-100 bg-primary-50 text-primary-700">
                    {Icon ? <Icon size={20} /> : null}
                  </div>
                  <h3 className="mt-5 font-serif text-2xl font-semibold text-lp-ink">{step.title}</h3>
                  <p className="mt-2 text-[15px] leading-relaxed text-lp-ink-soft">{step.desc}</p>
                </div>
              </li>
            );
          })}
        </ol>
      </div>
    </section>
  );
}
