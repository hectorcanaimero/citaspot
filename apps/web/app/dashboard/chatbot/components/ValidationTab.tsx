'use client';
import type { ChatbotConfig, KnowledgeDocument } from '@/lib/api';

interface Props {
  config: ChatbotConfig;
  docs: KnowledgeDocument[];
  chatTested: boolean;
}

export function ValidationTab({ config }: Props) {
  return <div>ValidationTab placeholder</div>;
}
