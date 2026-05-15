'use client';

import { useTranslations } from '@/lib/i18n';
import type { ChatbotConfig, KnowledgeDocument } from '@/lib/api';
import { CheckCircle2, Circle } from 'lucide-react';

interface Props {
  config: ChatbotConfig | null;
  docs: KnowledgeDocument[];
  chatTested: boolean;
  onNavigate: (tab: string) => void;
}

export function ProgressChecklist({ config, docs, chatTested, onNavigate }: Props) {
  const t = useTranslations();

  const activeDocs = docs.filter(d => d.is_active);

  const items = [
    {
      label: t.chatbot.progress.botName,
      done: Boolean(config?.bot_name),
      tab: 'personality',
    },
    {
      label: t.chatbot.progress.greeting,
      done: Boolean(config?.bot_greeting),
      tab: 'personality',
    },
    {
      label: t.chatbot.progress.docs,
      done: activeDocs.length > 0,
      tab: 'knowledge',
    },
    {
      label: t.chatbot.progress.tested,
      done: chatTested,
      tab: 'personality',
    },
    {
      label: t.chatbot.progress.validated,
      done: false,
      tab: 'validation',
    },
  ];

  const completed = items.filter(i => i.done).length;

  return (
    <div>
      <h2 className="text-sm font-semibold text-neutral-900 mb-3">
        {t.chatbot.progress.title}
      </h2>

      <ul className="space-y-2">
        {items.map((item, idx) => (
          <li key={idx}>
            <button
              onClick={() => onNavigate(item.tab)}
              className="flex items-center gap-2 text-sm w-full text-left hover:bg-neutral-50 rounded px-1.5 py-1 transition-colors"
            >
              {item.done ? (
                <CheckCircle2 className="w-4 h-4 text-green-500 shrink-0" />
              ) : (
                <Circle className="w-4 h-4 text-neutral-300 shrink-0" />
              )}
              <span className={item.done ? 'text-neutral-700' : 'text-neutral-400'}>
                {item.label}
              </span>
            </button>
          </li>
        ))}
      </ul>

      <div className="mt-4 pt-3 border-t border-neutral-100">
        <p className="text-xs text-neutral-400">
          {t.chatbot.progress.completed
            .replace('{n}', String(completed))
            .replace('{total}', String(items.length))}
        </p>
      </div>
    </div>
  );
}
