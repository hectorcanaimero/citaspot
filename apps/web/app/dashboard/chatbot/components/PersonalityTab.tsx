'use client';
import type { ChatbotConfig, KnowledgeDocument } from '@/lib/api';

interface Props {
  config: ChatbotConfig;
  docs: KnowledgeDocument[];
  onConfigUpdate: (config: ChatbotConfig) => void;
  onDocsRefresh: () => void;
}

export function PersonalityTab({ config }: Props) {
  return <div>PersonalityTab placeholder</div>;
}
