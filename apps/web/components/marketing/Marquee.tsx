'use client';

import { useTranslations } from '@/lib/i18n';

export function Marquee() {
  const t = useTranslations();
  const items = t.landing.marquee as readonly string[];

  return (
    <div className="overflow-hidden border-y border-lp-border bg-lp-bg-subtle py-4">
      <div className="flex w-[200%] animate-[marquee_24s_linear_infinite]">
        {[0, 1].map((n) => (
          <div key={n} className="flex w-1/2 shrink-0 whitespace-nowrap">
            {items.map((item, i) => (
              <span
                key={`${n}-${i}`}
                className={[
                  'px-7 text-[13px] font-medium',
                  i % 3 === 1 ? 'text-primary-600' : 'text-lp-ink-muted',
                ].join(' ')}
              >
                <span className="mr-2 text-primary-400">✦</span>
                {item}
              </span>
            ))}
          </div>
        ))}
      </div>
      <style jsx>{`
        @keyframes marquee {
          from { transform: translateX(0); }
          to   { transform: translateX(-50%); }
        }
      `}</style>
    </div>
  );
}
