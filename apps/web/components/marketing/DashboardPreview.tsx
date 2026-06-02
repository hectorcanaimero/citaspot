'use client';

import { useTranslations } from '@/lib/i18n';
import { DashboardMockup } from './DashboardMockup';

export function DashboardPreview() {
  const t = useTranslations();
  const l = t.landing;

  return (
    <section className="bg-lp-bg-subtle py-24 md:py-32">
      <div className="mx-auto max-w-6xl px-6">
        <div className="mx-auto max-w-2xl text-center">
          <div className="inline-flex items-center gap-1.5 rounded-full border border-lp-border bg-white px-3.5 py-1.5 text-[11px] font-semibold uppercase tracking-wider text-lp-ink-soft">
            <span className="h-1.5 w-1.5 rounded-full bg-primary-600" />
            {l.dashboardPill}
          </div>
          <h2 className="mt-6 font-serif font-semibold leading-[1.05] tracking-tight text-lp-ink text-[clamp(36px,4.5vw,56px)]">
            {l.dashboardTitle}
            <span className="italic text-primary-600">{l.dashboardTitleHighlight}</span>
          </h2>
          <p className="mt-5 text-lg text-lp-ink-muted">{l.dashboardSub}</p>
        </div>

        <div className="mt-16">
          <DashboardMockup />
        </div>
      </div>
    </section>
  );
}
