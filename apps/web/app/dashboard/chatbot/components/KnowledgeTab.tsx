'use client';
import type { KnowledgeDocument } from '@/lib/api';

interface Props {
  docs: KnowledgeDocument[];
  onRefresh: () => void;
}

export function KnowledgeTab({ docs }: Props) {
  return <div>KnowledgeTab placeholder - {docs.length} docs</div>;
}
