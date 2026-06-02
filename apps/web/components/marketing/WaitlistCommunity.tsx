'use client';

import { useTranslations } from '@/lib/i18n';
import { Rocket, Heart, MessageSquareQuote } from 'lucide-react';

const ICONS = [Rocket, Heart, MessageSquareQuote];

type Benefit = { title: string; desc: string };

export function WaitlistCommunity() {
  const t = useTranslations();
  const l = t.landing;
  const benefits = l.communityBenefits as readonly Benefit[];

  return (
    <section className="py-24 md:py-32">
      <div className="mx-auto max-w-6xl px-6">
        <div className="mx-auto max-w-2xl text-center">
          <div className="inline-flex items-center gap-1.5 rounded-full border border-primary-100 bg-primary-50 px-3.5 py-1.5 text-[11px] font-semibold uppercase tracking-wider text-primary-700">
            <span className="h-1.5 w-1.5 rounded-full bg-primary-600" />
            {l.communityPill}
          </div>
          <h2 className="mt-6 font-serif font-semibold leading-[1.05] tracking-tight text-lp-ink text-[clamp(36px,4.5vw,56px)]">
            {l.communityTitle}
            <span className="italic text-primary-600">{l.communityTitleHighlight}</span>
          </h2>
          <p className="mt-5 text-lg text-lp-ink-muted">{l.communitySub}</p>
        </div>

        <div className="mt-14 grid gap-5 md:grid-cols-3">
          {benefits.map((b, i) => {
            const Icon = ICONS[i] ?? Rocket;
            return (
              <div
                key={i}
                className="rounded-2xl border border-lp-border bg-white p-7 transition hover:border-primary-200 hover:shadow-lg hover:shadow-primary-600/[0.05]"
              >
                <div className="inline-flex h-11 w-11 items-center justify-center rounded-xl border border-primary-100 bg-primary-50 text-primary-700">
                  <Icon size={20} />
                </div>
                <h3 className="mt-5 font-serif text-2xl font-semibold text-lp-ink">{b.title}</h3>
                <p className="mt-2 text-[15px] leading-relaxed text-lp-ink-soft">{b.desc}</p>
              </div>
            );
          })}
        </div>
      </div>
    </section>
  );
}
