'use client';

import { useState } from 'react';
import { useTranslations } from '@/lib/i18n';
import { chatbotApi, knowledge as knowledgeApi, type ChatbotConfig, type KnowledgeDocument } from '@/lib/api';
import { Check } from 'lucide-react';
import { CHATBOT_TEMPLATES, type ChatbotTemplate } from '../data/templates';

interface Props {
  config: ChatbotConfig;
  docs: KnowledgeDocument[];
  onConfigUpdate: (config: ChatbotConfig) => void;
  onDocsRefresh: () => void;
}

export function PersonalityTab({ config, docs, onConfigUpdate, onDocsRefresh }: Props) {
  const t = useTranslations();
  const [saving, setSaving] = useState<string | null>(null);
  const [showTemplates, setShowTemplates] = useState(!config.template_id);

  // Auto-save con debounce al hacer blur
  async function handleFieldSave(field: string, value: string) {
    setSaving(field);
    try {
      const updated = await chatbotApi.updateConfig({ [field]: value });
      onConfigUpdate(updated);
    } catch (err) {
      console.error(`Error saving ${field}:`, err);
    } finally {
      // Mostrar checkmark brevemente
      setTimeout(() => setSaving(null), 1000);
    }
  }

  // Seleccionar template
  async function handleTemplateSelect(template: ChatbotTemplate) {
    try {
      // 1. Actualizar config con defaults del template
      const updated = await chatbotApi.updateConfig({
        bot_name: template.defaults.bot_name,
        bot_greeting: template.defaults.bot_greeting,
        tone: template.defaults.tone,
        template_id: template.id,
      });
      onConfigUpdate(updated);

      // 2. Crear documentos sugeridos como drafts (is_active: false)
      for (const doc of template.suggested_docs) {
        try {
          await knowledgeApi.create({
            category: doc.category,
            title: doc.title,
            content: doc.content,
            is_active: false,
          });
        } catch (err) {
          console.error('Error creating suggested doc:', err);
        }
      }

      onDocsRefresh();
      setShowTemplates(false);
    } catch (err) {
      console.error('Error applying template:', err);
    }
  }

  const toneOptions = [
    { value: 'friendly', label: t.chatbot.personality.toneFriendly },
    { value: 'professional', label: t.chatbot.personality.toneProfessional },
    { value: 'premium', label: t.chatbot.personality.tonePremium },
    { value: 'casual', label: t.chatbot.personality.toneCasual },
  ] as const;

  return (
    <div className="space-y-6 max-w-2xl">
      {/* Template Selector */}
      {showTemplates ? (
        <section>
          <h2 className="text-sm font-semibold text-neutral-900 mb-3">
            {t.chatbot.personality.templateTitle}
          </h2>
          <div className="grid grid-cols-2 sm:grid-cols-3 gap-3">
            {CHATBOT_TEMPLATES.map((tmpl) => (
              <button
                key={tmpl.id}
                onClick={() => handleTemplateSelect(tmpl)}
                className="flex flex-col items-center gap-2 p-4 rounded-lg border border-neutral-200 hover:border-primary-400 hover:bg-primary-50 transition-colors text-center"
              >
                <span className="text-2xl">{tmpl.icon}</span>
                <span className="text-sm font-medium text-neutral-700">
                  {(t.chatbot.personality as Record<string, string>)[tmpl.id] ?? tmpl.name}
                </span>
              </button>
            ))}
          </div>
        </section>
      ) : (
        <button
          onClick={() => setShowTemplates(true)}
          className="text-sm text-primary-600 hover:underline"
        >
          {t.chatbot.personality.templateChange}
        </button>
      )}

      {/* Config Fields */}
      <section className="space-y-4">
        {/* Bot Name */}
        <div>
          <label className="block text-sm font-medium text-neutral-700 mb-1">
            {t.chatbot.personality.botName}
            {saving === 'bot_name' && (
              <Check className="inline w-4 h-4 text-green-500 ml-1" />
            )}
          </label>
          <input
            type="text"
            maxLength={30}
            className="w-full rounded-lg border border-neutral-300 px-3 py-2 text-sm focus:border-primary-500 focus:ring-1 focus:ring-primary-500"
            defaultValue={config.bot_name}
            onBlur={(e) => handleFieldSave('bot_name', e.target.value)}
          />
          <p className="text-xs text-neutral-500 mt-1">{t.chatbot.personality.botNameHint}</p>
        </div>

        {/* Bot Greeting */}
        <div>
          <label className="block text-sm font-medium text-neutral-700 mb-1">
            {t.chatbot.personality.botGreeting}
            {saving === 'bot_greeting' && (
              <Check className="inline w-4 h-4 text-green-500 ml-1" />
            )}
          </label>
          <textarea
            maxLength={500}
            rows={3}
            className="w-full rounded-lg border border-neutral-300 px-3 py-2 text-sm focus:border-primary-500 focus:ring-1 focus:ring-primary-500"
            defaultValue={config.bot_greeting}
            onBlur={(e) => handleFieldSave('bot_greeting', e.target.value)}
          />
          <p className="text-xs text-neutral-500 mt-1">{t.chatbot.personality.botGreetingHint}</p>
        </div>

        {/* Tone */}
        <div>
          <label className="block text-sm font-medium text-neutral-700 mb-1">
            {t.chatbot.personality.tone}
            {saving === 'tone' && (
              <Check className="inline w-4 h-4 text-green-500 ml-1" />
            )}
          </label>
          <div className="grid grid-cols-2 gap-2">
            {toneOptions.map((opt) => (
              <button
                key={opt.value}
                onClick={() => handleFieldSave('tone', opt.value)}
                className={`px-3 py-2 rounded-lg border text-sm font-medium transition-colors ${
                  config.tone === opt.value
                    ? 'border-primary-500 bg-primary-50 text-primary-700'
                    : 'border-neutral-200 text-neutral-600 hover:border-neutral-300'
                }`}
              >
                {opt.label}
              </button>
            ))}
          </div>
        </div>

        {/* Custom Instructions */}
        <div>
          <label className="block text-sm font-medium text-neutral-700 mb-1">
            {t.chatbot.personality.customInstructions}
            {saving === 'custom_instructions' && (
              <Check className="inline w-4 h-4 text-green-500 ml-1" />
            )}
          </label>
          <textarea
            maxLength={1000}
            rows={4}
            className="w-full rounded-lg border border-neutral-300 px-3 py-2 text-sm focus:border-primary-500 focus:ring-1 focus:ring-primary-500"
            defaultValue={config.custom_instructions}
            onBlur={(e) => handleFieldSave('custom_instructions', e.target.value)}
          />
          <p className="text-xs text-neutral-500 mt-1">{t.chatbot.personality.customInstructionsHint}</p>
        </div>
      </section>
    </div>
  );
}
