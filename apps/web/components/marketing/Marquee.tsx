'use client';

import { useTranslations } from '@/lib/i18n';

export function Marquee() {
  const t = useTranslations();
  const items = t.landing.marquee as readonly string[];

  return (
    <div className="overflow-hidden border-y border-lp-border bg-lp-bg-subtle py-4">
      <div className="flex w-max animate-[marquee_28s_linear_infinite]">
        {[0, 1].map((n) => (
          <ul
            key={n}
            className="flex shrink-0 whitespace-nowrap"
            aria-hidden={n === 1}
          >
            {items.map((item, i) => (
              <li
                key={`${n}-${i}`}
                className={[
                  'flex items-center gap-2 px-7 text-[13px] font-medium',
                  i % 3 === 1 ? 'text-primary-600' : 'text-lp-ink-muted',
                ].join(' ')}
              >
                <span className="text-primary-400">✦</span>
                {item}
              </li>
            ))}
          </ul>
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
