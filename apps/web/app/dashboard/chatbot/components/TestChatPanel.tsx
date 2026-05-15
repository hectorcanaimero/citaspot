'use client';

import { useState, useRef, useEffect } from 'react';
import { useTranslations } from '@/lib/i18n';
import { chatbotApi, type ChatbotTestResponse } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Send, Trash2, ChevronDown, ChevronRight, PanelRightClose } from 'lucide-react';

interface ChatMessage {
  id: string;
  role: 'user' | 'bot';
  content: string;
  timestamp: Date;
  debug?: {
    intent: string;
    sources: string[];
    timeMs: number;
  };
}

interface Props {
  onMessageSent: () => void;
  onCollapse: () => void;
}

export function TestChatPanel({ onMessageSent, onCollapse }: Props) {
  const t = useTranslations();
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [input, setInput] = useState('');
  const [sending, setSending] = useState(false);
  const [expandedDebug, setExpandedDebug] = useState<string | null>(null);
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  // Auto-scroll al final cuando hay nuevos mensajes
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  async function handleSend() {
    if (!input.trim() || sending) return;

    const userMsg: ChatMessage = {
      id: `user-${Date.now()}`,
      role: 'user',
      content: input.trim(),
      timestamp: new Date(),
    };

    setMessages(prev => [...prev, userMsg]);
    setInput('');
    setSending(true);
    onMessageSent();

    try {
      const resp: ChatbotTestResponse = await chatbotApi.test(userMsg.content);
      const botMsg: ChatMessage = {
        id: `bot-${Date.now()}`,
        role: 'bot',
        content: resp.response,
        timestamp: new Date(),
        debug: {
          intent: resp.intent_detected,
          sources: resp.rag_sources_used,
          timeMs: resp.processing_time_ms,
        },
      };
      setMessages(prev => [...prev, botMsg]);
    } catch {
      const errorMsg: ChatMessage = {
        id: `error-${Date.now()}`,
        role: 'bot',
        content: 'Error al procesar el mensaje. Verificá que el servicio de IA esté activo.',
        timestamp: new Date(),
      };
      setMessages(prev => [...prev, errorMsg]);
    } finally {
      setSending(false);
      inputRef.current?.focus();
    }
  }

  function handleClear() {
    setMessages([]);
    setExpandedDebug(null);
  }

  function handleKeyDown(e: React.KeyboardEvent) {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  }

  return (
    <div className="flex flex-col h-full">
      {/* Header */}
      <div className="flex items-center justify-between px-3 py-2 border-b border-neutral-200 shrink-0">
        <h3 className="text-sm font-semibold text-neutral-900">{t.chatbot.testChat.title}</h3>
        <div className="flex gap-1">
          <button onClick={handleClear} className="p-1 rounded hover:bg-neutral-100" title={t.chatbot.testChat.clear}>
            <Trash2 className="w-4 h-4 text-neutral-400" />
          </button>
          <button onClick={onCollapse} className="p-1 rounded hover:bg-neutral-100" title={t.chatbot.testChat.collapse}>
            <PanelRightClose className="w-4 h-4 text-neutral-400" />
          </button>
        </div>
      </div>

      {/* Banner */}
      <div className="px-3 py-1.5 bg-amber-50 border-b border-amber-100 shrink-0">
        <p className="text-[11px] text-amber-700">{t.chatbot.testChat.banner}</p>
      </div>

      {/* Messages */}
      <div className="flex-1 overflow-y-auto p-3 space-y-3">
        {messages.length === 0 ? (
          <p className="text-sm text-neutral-400 text-center mt-8">{t.chatbot.testChat.empty}</p>
        ) : (
          messages.map(msg => (
            <div key={msg.id}>
              <div className={`flex ${msg.role === 'user' ? 'justify-end' : 'justify-start'}`}>
                <div className={`max-w-[85%] rounded-lg px-3 py-2 text-sm ${
                  msg.role === 'user'
                    ? 'bg-primary-600 text-white'
                    : 'bg-neutral-100 text-neutral-800'
                }`}>
                  <p className="whitespace-pre-wrap">{msg.content}</p>
                  <p className={`text-[10px] mt-1 ${
                    msg.role === 'user' ? 'text-primary-200' : 'text-neutral-400'
                  }`}>
                    {msg.timestamp.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
                  </p>
                </div>
              </div>

              {/* Debug metadata (colapsable) */}
              {msg.debug && (
                <div className="ml-1 mt-1">
                  <button
                    onClick={() => setExpandedDebug(expandedDebug === msg.id ? null : msg.id)}
                    className="flex items-center gap-1 text-[10px] text-neutral-400 hover:text-neutral-600"
                  >
                    {expandedDebug === msg.id ? (
                      <ChevronDown className="w-3 h-3" />
                    ) : (
                      <ChevronRight className="w-3 h-3" />
                    )}
                    {t.chatbot.testChat.debug}
                  </button>
                  {expandedDebug === msg.id && (
                    <div className="mt-1 ml-4 text-[10px] text-neutral-400 space-y-0.5">
                      <p>{t.chatbot.testChat.intent}: <span className="text-neutral-600">{msg.debug.intent}</span></p>
                      {msg.debug.sources.length > 0 && (
                        <p>{t.chatbot.testChat.sources}: <span className="text-neutral-600">{msg.debug.sources.join(', ')}</span></p>
                      )}
                      <p>{t.chatbot.testChat.time}: <span className="text-neutral-600">{msg.debug.timeMs}ms</span></p>
                    </div>
                  )}
                </div>
              )}
            </div>
          ))
        )}
        <div ref={messagesEndRef} />
      </div>

      {/* Input */}
      <div className="px-3 py-2 border-t border-neutral-200 shrink-0">
        <div className="flex gap-2">
          <input
            ref={inputRef}
            type="text"
            className="flex-1 rounded-lg border border-neutral-300 px-3 py-2 text-sm focus:border-primary-500 focus:ring-1 focus:ring-primary-500"
            placeholder={t.chatbot.testChat.placeholder}
            value={input}
            onChange={e => setInput(e.target.value)}
            onKeyDown={handleKeyDown}
            disabled={sending}
          />
          <Button size="sm" onClick={handleSend} loading={sending} disabled={!input.trim()}>
            <Send className="w-4 h-4" />
          </Button>
        </div>
      </div>
    </div>
  );
}
