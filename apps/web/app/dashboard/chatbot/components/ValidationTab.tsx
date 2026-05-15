'use client';

import { useState } from 'react';
import { useTranslations } from '@/lib/i18n';
import { chatbotApi, type ChatbotConfig, type KnowledgeDocument, type ChatbotValidateResponse } from '@/lib/api';
import { Card } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { CheckCircle2, Circle, AlertCircle, ArrowRight, Sparkles } from 'lucide-react';

interface Props {
  config: ChatbotConfig;
  docs: KnowledgeDocument[];
  chatTested: boolean;
}

interface StaticRule {
  key: string;
  label: string;
  passed: boolean;
  suggestion: string;
  tab?: string;
}

export function ValidationTab({ config, docs, chatTested }: Props) {
  const t = useTranslations();
  const [aiReport, setAiReport] = useState<ChatbotValidateResponse | null>(null);
  const [validating, setValidating] = useState(false);
  const [rateLimited, setRateLimited] = useState(false);

  const activeDocs = docs.filter(d => d.is_active);
  const hasActiveByCategory = (cat: string) => activeDocs.some(d => d.category === cat);

  const staticRules: StaticRule[] = [
    {
      key: 'botName',
      label: t.chatbot.progress.botName,
      passed: Boolean(config.bot_name),
      suggestion: t.chatbot.validation.rules.botName,
      tab: 'personality',
    },
    {
      key: 'greeting',
      label: t.chatbot.progress.greeting,
      passed: Boolean(config.bot_greeting),
      suggestion: t.chatbot.validation.rules.greeting,
      tab: 'personality',
    },
    {
      key: 'activeDocs',
      label: t.chatbot.progress.docs,
      passed: activeDocs.length > 0,
      suggestion: t.chatbot.validation.rules.activeDocs,
      tab: 'knowledge',
    },
    {
      key: 'faq',
      label: 'FAQ',
      passed: hasActiveByCategory('faq'),
      suggestion: t.chatbot.validation.rules.faqCategory,
      tab: 'knowledge',
    },
    {
      key: 'pricing',
      label: 'Precios',
      passed: hasActiveByCategory('pricing') || hasActiveByCategory('services'),
      suggestion: t.chatbot.validation.rules.pricingDocs,
      tab: 'knowledge',
    },
    {
      key: 'policies',
      label: 'Políticas',
      passed: hasActiveByCategory('policies'),
      suggestion: t.chatbot.validation.rules.policyDocs,
      tab: 'knowledge',
    },
    {
      key: 'chatTested',
      label: t.chatbot.progress.tested,
      passed: chatTested,
      suggestion: t.chatbot.validation.rules.chatTested,
    },
  ];

  async function handleValidate() {
    setValidating(true);
    setRateLimited(false);
    try {
      const report = await chatbotApi.validate();
      setAiReport(report);
    } catch (err: unknown) {
      if ((err as { status?: number })?.status === 429) {
        setRateLimited(true);
      } else {
        console.error('Error validating:', err);
      }
    } finally {
      setValidating(false);
    }
  }

  const canValidate = activeDocs.length > 0;

  return (
    <div className="space-y-6 max-w-2xl">
      {/* Static Checklist */}
      <section>
        <h2 className="text-sm font-semibold text-neutral-900 mb-3">
          {t.chatbot.validation.staticTitle}
        </h2>
        <Card className="divide-y divide-neutral-100">
          {staticRules.map(rule => (
            <div key={rule.key} className="flex items-start gap-3 px-4 py-3">
              {rule.passed ? (
                <CheckCircle2 className="w-5 h-5 text-green-500 shrink-0 mt-0.5" />
              ) : (
                <Circle className="w-5 h-5 text-neutral-300 shrink-0 mt-0.5" />
              )}
              <div className="flex-1">
                <p className={`text-sm ${rule.passed ? 'text-neutral-700' : 'text-neutral-500'}`}>
                  {rule.label}
                </p>
                {!rule.passed && (
                  <p className="text-xs text-amber-600 mt-0.5">{rule.suggestion}</p>
                )}
              </div>
            </div>
          ))}
        </Card>
      </section>

      {/* AI Validator */}
      <section>
        <h2 className="text-sm font-semibold text-neutral-900 mb-3">
          {t.chatbot.validation.aiTitle}
        </h2>

        <Button
          onClick={handleValidate}
          disabled={!canValidate || validating}
          loading={validating}
        >
          <Sparkles className="w-4 h-4 mr-1" />
          {validating ? t.chatbot.validation.aiRunning : t.chatbot.validation.aiButton}
        </Button>

        {!canValidate && (
          <p className="text-xs text-neutral-400 mt-2">{t.chatbot.validation.aiDisabled}</p>
        )}

        {rateLimited && (
          <p className="text-xs text-amber-600 mt-2">{t.chatbot.validation.aiRateLimit}</p>
        )}

        {/* AI Report */}
        {aiReport && (
          <Card className="mt-4">
            <div className="px-4 py-3 border-b border-neutral-100">
              <p className="text-sm font-medium text-neutral-900">
                {t.chatbot.validation.result
                  .replace('{passed}', String(aiReport.passed))
                  .replace('{total}', String(aiReport.total_questions))}
              </p>
              <div className="w-full bg-neutral-100 rounded-full h-1.5 mt-2">
                <div
                  className="bg-green-500 h-1.5 rounded-full transition-all"
                  style={{ width: `${(aiReport.passed / aiReport.total_questions) * 100}%` }}
                />
              </div>
            </div>

            <div className="divide-y divide-neutral-100">
              {aiReport.results.map((result, idx) => (
                <div key={idx} className="px-4 py-3">
                  <div className="flex items-start gap-2">
                    {result.passed ? (
                      <CheckCircle2 className="w-4 h-4 text-green-500 shrink-0 mt-0.5" />
                    ) : (
                      <AlertCircle className="w-4 h-4 text-red-400 shrink-0 mt-0.5" />
                    )}
                    <div className="flex-1">
                      <p className="text-sm text-neutral-700">"{result.question}"</p>
                      <p className="text-xs text-neutral-500 mt-1 line-clamp-2">
                        {result.response}
                      </p>
                      {!result.passed && result.suggestion && (
                        <p className="text-xs text-amber-600 mt-1 flex items-center gap-1">
                          <ArrowRight className="w-3 h-3" /> {result.suggestion}
                        </p>
                      )}
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </Card>
        )}
      </section>
    </div>
  );
}
