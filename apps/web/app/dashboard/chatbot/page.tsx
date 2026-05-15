'use client';

import { useState, useEffect, useCallback } from 'react';
import { useTranslations } from '@/lib/i18n';
import { chatbotApi, knowledge, type ChatbotConfig, type KnowledgeDocument } from '@/lib/api';
import { Spinner } from '@/components/ui/spinner';
import { ProgressChecklist } from './components/ProgressChecklist';
import { PersonalityTab } from './components/PersonalityTab';
import { KnowledgeTab } from './components/KnowledgeTab';
import { ValidationTab } from './components/ValidationTab';
import { TestChatPanel } from './components/TestChatPanel';

type Tab = 'personality' | 'knowledge' | 'validation';

export default function ChatbotPage() {
  const t = useTranslations();
  const [activeTab, setActiveTab] = useState<Tab>('personality');
  const [config, setConfig] = useState<ChatbotConfig | null>(null);
  const [docs, setDocs] = useState<KnowledgeDocument[]>([]);
  const [loading, setLoading] = useState(true);
  const [chatTested, setChatTested] = useState(false);
  const [chatPanelOpen, setChatPanelOpen] = useState(true);

  useEffect(() => {
    async function load() {
      try {
        const [cfgRes, docsRes] = await Promise.all([
          chatbotApi.config(),
          knowledge.list(),
        ]);
        setConfig(cfgRes);
        setDocs(docsRes.data);
      } catch (err) {
        console.error('Error loading chatbot data:', err);
      } finally {
        setLoading(false);
      }
    }
    load();
  }, []);

  const handleConfigUpdate = useCallback((updated: ChatbotConfig) => {
    setConfig(updated);
  }, []);

  const refreshDocs = useCallback(async () => {
    try {
      const res = await knowledge.list();
      setDocs(res.data);
    } catch (err) {
      console.error('Error refreshing docs:', err);
    }
  }, []);

  if (loading) {
    return (
      <div className="flex items-center justify-center h-96">
        <Spinner size="lg" />
      </div>
    );
  }

  const tabs: { key: Tab; label: string }[] = [
    { key: 'personality', label: t.chatbot.tabs.personality },
    { key: 'knowledge', label: t.chatbot.tabs.knowledge },
    { key: 'validation', label: t.chatbot.tabs.validation },
  ];

  return (
    <div className="flex flex-col h-[calc(100vh-3.5rem)]">
      {/* Header */}
      <div className="px-4 py-3 border-b border-neutral-200 shrink-0">
        <h1 className="text-lg font-semibold text-neutral-900">{t.chatbot.title}</h1>
        <p className="text-sm text-neutral-500">{t.chatbot.subtitle}</p>
      </div>

      {/* Layout de 3 columnas */}
      <div className="flex flex-1 overflow-hidden">
        {/* Izquierda: Progress Checklist */}
        <aside className="hidden lg:block w-52 border-r border-neutral-200 p-4 overflow-y-auto shrink-0">
          <ProgressChecklist
            config={config}
            docs={docs}
            chatTested={chatTested}
            onNavigate={(tab) => setActiveTab(tab as Tab)}
          />
        </aside>

        {/* Centro: Contenido con tabs */}
        <main className="flex-1 overflow-y-auto">
          {/* Tabs */}
          <div className="flex border-b border-neutral-200 px-4">
            {tabs.map((tab) => (
              <button
                key={tab.key}
                onClick={() => setActiveTab(tab.key)}
                className={`px-4 py-2.5 text-sm font-medium border-b-2 transition-colors ${
                  activeTab === tab.key
                    ? 'border-primary-600 text-primary-600'
                    : 'border-transparent text-neutral-500 hover:text-neutral-700'
                }`}
              >
                {tab.label}
              </button>
            ))}
          </div>

          {/* Contenido del tab activo */}
          <div className="p-4">
            {activeTab === 'personality' && config && (
              <PersonalityTab
                config={config}
                docs={docs}
                onConfigUpdate={handleConfigUpdate}
                onDocsRefresh={refreshDocs}
              />
            )}
            {activeTab === 'knowledge' && (
              <KnowledgeTab
                docs={docs}
                onRefresh={refreshDocs}
              />
            )}
            {activeTab === 'validation' && config && (
              <ValidationTab
                config={config}
                docs={docs}
                chatTested={chatTested}
              />
            )}
          </div>
        </main>

        {/* Derecha: Panel de chat de prueba */}
        {chatPanelOpen ? (
          <aside className="hidden xl:flex w-[350px] border-l border-neutral-200 flex-col shrink-0">
            <TestChatPanel
              onMessageSent={() => setChatTested(true)}
              onCollapse={() => setChatPanelOpen(false)}
            />
          </aside>
        ) : (
          <button
            onClick={() => setChatPanelOpen(true)}
            className="hidden xl:flex items-center justify-center w-10 border-l border-neutral-200 bg-neutral-50 hover:bg-neutral-100 transition-colors"
            title={t.chatbot.testChat.expand}
          >
            <span className="text-xs font-medium text-neutral-500 [writing-mode:vertical-lr] rotate-180">
              {t.chatbot.testChat.expand}
            </span>
          </button>
        )}
      </div>
    </div>
  );
}
