'use client';

import { Sparkles } from 'lucide-react';
import { useTranslations } from '@/lib/i18n';

type PainPoint = {
  number: string;
  problem: string;
  solution: string;
  desc: string;
};

export function PainPoints() {
  const t = useTranslations();
  const l = t.landing;
  const items = l.painPoints as readonly PainPoint[];

  return (
    <section id="beneficios" className="bg-lp-bg-subtle py-24 md:py-32">
      <div className="mx-auto max-w-6xl px-6">
        <div className="mx-auto max-w-2xl text-center">
          <div className="inline-flex items-center gap-1.5 rounded-full border border-lp-border bg-white px-3.5 py-1.5 text-[11px] font-semibold uppercase tracking-wider text-lp-ink-soft">
            <Sparkles size={12} className="text-primary-600" />
            {l.painPointsPill}
          </div>
          <h2 className="mt-6 font-serif font-semibold leading-[1.05] tracking-tight text-lp-ink text-[clamp(36px,4.5vw,56px)]">
            {l.painPointsTitle}
            <span className="italic text-primary-600">{l.painPointsTitleHighlight}</span>
          </h2>
        </div>

        <div className="mt-20 flex flex-col gap-24 md:gap-32">
          {items.map((item, i) => {
            const flip = i % 2 === 1;
            return (
              <div
                key={item.number}
                className={[
                  'grid items-center gap-10 md:gap-16',
                  'md:grid-cols-2',
                ].join(' ')}
              >
                <div className={flip ? 'md:order-2' : ''}>
                  <span className="font-serif text-[64px] font-semibold leading-none text-primary-600/30">
                    {item.number}
                  </span>
                  <div className="mt-4 text-[11px] font-semibold uppercase tracking-wider text-lp-ink-muted">
                    {item.problem}
                  </div>
                  <h3 className="mt-3 font-serif font-semibold leading-[1.1] tracking-tight text-lp-ink text-[clamp(28px,3.5vw,42px)]">
                    {item.solution}
                  </h3>
                  <p className="mt-5 max-w-md text-base leading-relaxed text-lp-ink-soft">
                    {item.desc}
                  </p>
                </div>

                <div className={flip ? 'md:order-1' : ''}>
                  <PainVisual index={i} />
                </div>
              </div>
            );
          })}
        </div>
      </div>
    </section>
  );
}

function PainVisual({ index }: { index: number }) {
  const wrap = 'relative aspect-[4/3] w-full overflow-hidden rounded-3xl border border-lp-border bg-white p-6 shadow-xl shadow-primary-600/[0.07]';

  if (index === 0) {
    // 01 — Multiple channels converging to WhatsApp
    return (
      <div className={wrap}>
        <div className="grid h-full grid-cols-3 grid-rows-3 gap-2">
          {/* Scattered chat bubbles */}
          <FakeBubble label="IG" tone="rose" className="col-start-1 row-start-1" />
          <FakeBubble label="FB" tone="indigo" className="col-start-3 row-start-1" />
          <FakeBubble label="Web" tone="amber" className="col-start-1 row-start-3" />
          <FakeBubble label="?" tone="neutral" className="col-start-3 row-start-3" />
          {/* Center WhatsApp */}
          <div className="col-start-2 row-start-2 flex items-center justify-center">
            <div className="flex h-16 w-16 items-center justify-center rounded-2xl bg-[#25D366] shadow-lg shadow-[#25D366]/30">
              <svg width="32" height="32" viewBox="0 0 24 24" fill="white">
                <path d="M12.04 2C6.58 2 2.13 6.45 2.13 11.91c0 1.75.46 3.45 1.32 4.95L2.05 22l5.25-1.38c1.45.79 3.08 1.21 4.74 1.21h.01c5.46 0 9.91-4.45 9.91-9.91 0-2.65-1.03-5.14-2.9-7.01a9.816 9.816 0 00-7.02-2.91M12.05 4.69c1.94 0 3.76.75 5.13 2.12a7.23 7.23 0 012.13 5.13c0 4.01-3.26 7.27-7.27 7.27-1.42 0-2.81-.4-4-1.15l-.29-.17-2.99.78.8-2.91-.18-.3a7.22 7.22 0 01-1.12-3.88c0-4.01 3.26-7.27 7.27-7.27" />
              </svg>
            </div>
          </div>
        </div>
        <div className="absolute bottom-6 left-6 right-6 text-center text-xs font-semibold text-lp-ink-soft">
          Todo en un canal
        </div>
      </div>
    );
  }

  if (index === 1) {
    // 02 — Clock 2 AM with notification
    return (
      <div className={wrap}>
        <div className="flex h-full flex-col items-center justify-center gap-5">
          <div className="relative">
            <div className="flex h-32 w-32 items-center justify-center rounded-full border-4 border-lp-border bg-white">
              <div className="font-serif text-3xl font-semibold text-lp-ink">02:14</div>
            </div>
            <div className="absolute -right-2 -top-2 flex h-7 w-7 items-center justify-center rounded-full bg-rose-500 text-xs font-bold text-white shadow-lg shadow-rose-500/40">
              3
            </div>
          </div>
          <div className="w-full max-w-[260px] space-y-1.5">
            <FakeMsg from="user" text="¿Tienen turno para mañana?" />
            <FakeMsg from="bot" text="Sí, ¿qué hora prefieres? 🌙" />
          </div>
        </div>
      </div>
    );
  }

  if (index === 2) {
    // 03 — Mini calendar with availability
    const slots = [
      { time: '09:00', taken: true },
      { time: '10:00', taken: false },
      { time: '11:00', taken: false },
      { time: '12:00', taken: true },
      { time: '14:00', taken: false },
      { time: '15:00', taken: true },
      { time: '16:00', taken: false },
      { time: '17:00', taken: false },
    ];
    return (
      <div className={wrap}>
        <div className="flex h-full flex-col">
          <div className="mb-4 flex items-center justify-between border-b border-lp-border pb-3">
            <div className="font-serif text-lg font-semibold text-lp-ink">Mar 12 mar</div>
            <div className="text-xs text-lp-ink-muted">5 slots libres</div>
          </div>
          <div className="grid flex-1 grid-cols-4 gap-2">
            {slots.map((s) => (
              <div
                key={s.time}
                className={[
                  'flex items-center justify-center rounded-lg border py-2 text-xs font-semibold',
                  s.taken
                    ? 'border-lp-border bg-lp-bg-subtle text-lp-ink-muted line-through'
                    : 'border-primary-200 bg-primary-50 text-primary-700',
                ].join(' ')}
              >
                {s.time}
              </div>
            ))}
          </div>
        </div>
      </div>
    );
  }

  // 04 — Reminder notification with confirm
  return (
    <div className={wrap}>
      <div className="flex h-full flex-col justify-center gap-3">
        <FakeMsg
          from="bot"
          text="Hola María 👋 Te recordamos tu cita mañana 10:00 AM - Limpieza facial. ¿Confirmas?"
        />
        <div className="flex gap-2 self-end">
          <button className="rounded-full border border-lp-border bg-white px-4 py-2 text-xs font-semibold text-lp-ink-soft">
            Reagendar
          </button>
          <button className="rounded-full bg-success px-4 py-2 text-xs font-semibold text-white">
            ✓ Confirmar
          </button>
        </div>
        <FakeMsg from="user" text="✓ Confirmado, gracias!" />
      </div>
    </div>
  );
}

function FakeBubble({
  label,
  tone,
  className,
}: {
  label: string;
  tone: 'rose' | 'indigo' | 'amber' | 'neutral';
  className?: string;
}) {
  const tones = {
    rose: 'bg-rose-50 text-rose-600 border-rose-100',
    indigo: 'bg-primary-50 text-primary-600 border-primary-100',
    amber: 'bg-amber-50 text-amber-600 border-amber-100',
    neutral: 'bg-neutral-100 text-neutral-500 border-neutral-200',
  } as const;
  return (
    <div className={[className, 'flex items-start justify-start'].join(' ')}>
      <div
        className={[
          'inline-flex items-center gap-1 rounded-2xl border px-2.5 py-1.5 text-[10px] font-bold',
          tones[tone],
        ].join(' ')}
      >
        {label}
        <span className="ml-0.5 inline-block h-1.5 w-1.5 rounded-full bg-current opacity-50" />
      </div>
    </div>
  );
}

function FakeMsg({ from, text }: { from: 'bot' | 'user'; text: string }) {
  const isBot = from === 'bot';
  return (
    <div
      className={[
        'max-w-[80%] rounded-2xl px-3 py-2 text-[12px] leading-snug',
        isBot
          ? 'self-start rounded-bl-md bg-white text-lp-ink shadow-sm'
          : 'self-end rounded-br-md bg-[#DCF8C6] text-neutral-900',
      ].join(' ')}
    >
      {text}
    </div>
  );
}
