'use client';

import { useState, useEffect, useCallback } from 'react';
import { useTranslations } from '@/lib/i18n';

export function PhoneMockup() {
  const t = useTranslations();
  const l = t.landing;
  const [step, setStep] = useState(0);

  const runAnimation = useCallback(() => {
    setStep(0);
    const timers = [
      setTimeout(() => setStep(1), 600),
      setTimeout(() => setStep(2), 1800),
      setTimeout(() => setStep(3), 3000),
      setTimeout(() => setStep(4), 4200),
      setTimeout(() => setStep(5), 5400),
      setTimeout(() => setStep(6), 6600),
      setTimeout(() => setStep(7), 7800),
      setTimeout(runAnimation, 11000),
    ];
    return timers;
  }, []);

  useEffect(() => {
    const timers = runAnimation();
    return () => timers.forEach(clearTimeout);
  }, [runAnimation]);

  const messages = [
    { text: l.phoneMsg1, from: 'bot', maxW: '88%', time: '9:41' },
    { text: l.phoneMsg2, from: 'user', maxW: '78%', time: '9:41' },
    { text: l.phoneMsg3, from: 'bot', maxW: '82%', time: '9:41' },
    { text: l.phoneMsg4, from: 'user', maxW: '58%', time: '9:42' },
    { text: l.phoneMsg5, from: 'bot', maxW: '80%', time: '9:42' },
    { text: l.phoneMsg6, from: 'user', maxW: '62%', time: '9:42' },
    { text: l.phoneMsg7, from: 'bot', maxW: '92%', time: '9:42' },
  ];

  return (
    <div
      className="relative flex shrink-0 flex-col overflow-hidden rounded-[44px] border-[9px] border-neutral-900 bg-neutral-950 shadow-[0_30px_60px_-15px_rgba(99,102,241,0.25),0_18px_36px_-18px_rgba(0,0,0,0.3)]"
      style={{ width: 272, height: 600, transform: 'rotate(2deg)' }}
    >
      {/* Status bar */}
      <div className="flex items-center justify-between bg-[#075E54] px-4 pb-1 pt-2">
        <span className="text-[11px] font-bold text-white">9:41</span>
        <span className="text-[10px] text-white/80">●●●</span>
      </div>

      {/* WhatsApp header */}
      <div className="flex items-center gap-2.5 bg-[#075E54] px-3.5 pb-2.5 pt-1.5">
        <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-gradient-to-br from-[#25D366] to-[#128C7E] text-[17px]">
          🤖
        </div>
        <div className="min-w-0 flex-1">
          <div className="text-[13px] font-semibold leading-tight text-white">{l.phoneAgentName}</div>
          <div className="mt-0.5 text-[9.5px] leading-snug text-white/75">{l.phoneOnline}</div>
        </div>
      </div>

      {/* Chat body */}
      <div className="flex flex-1 flex-col gap-[7px] overflow-hidden bg-[#ECE5DD] px-2.5 py-3">
        {messages.map((m, i) => {
          const visible = step >= i + 1;
          const isBot = m.from === 'bot';
          return (
            <div
              key={i}
              className={[
                'px-2.5 py-[7px] text-[12px] leading-snug text-neutral-900',
                'shadow-[0_1px_0.5px_rgba(0,0,0,0.08)] transition-all duration-300 ease-out',
                isBot
                  ? 'self-start rounded-[12px] rounded-bl-[4px] bg-white'
                  : 'self-end rounded-[12px] rounded-br-[4px] bg-[#DCF8C6]',
                visible ? 'translate-y-0 opacity-100' : 'translate-y-2 opacity-0',
                i > 0 ? 'mt-1.5' : '',
              ].join(' ')}
              style={{ maxWidth: m.maxW }}
            >
              {m.text}
              <div
                className={[
                  'mt-1 text-right text-[10px]',
                  isBot ? 'text-neutral-400' : 'text-[#7BAE7E]',
                ].join(' ')}
              >
                {m.time}
                {!isBot ? ' ✓✓' : ''}
              </div>
            </div>
          );
        })}
      </div>

      {/* Input bar */}
      <div className="flex h-12 items-center gap-2 bg-[#F0F0F0] px-2.5">
        <div className="flex h-[34px] flex-1 items-center rounded-[20px] bg-white px-3 text-[11px] text-neutral-400">
          {l.phonePlaceholder}
        </div>
        <div className="flex h-[34px] w-[34px] items-center justify-center rounded-full bg-[#075E54]">
          <span className="text-[14px] text-white">→</span>
        </div>
      </div>
    </div>
  );
}
