'use client';

import { Droppable } from '@hello-pangea/dnd';
import { Customer } from '@/lib/api';
import { useTranslations } from '@/lib/i18n';
import { KanbanCard } from './KanbanCard';

interface KanbanColumnProps {
  stageId: string;
  title: string;
  color: string;
  customers: Customer[];
}

export function KanbanColumn({ stageId, title, color, customers }: KanbanColumnProps) {
  const t = useTranslations();

  return (
    <div className="flex w-72 flex-shrink-0 flex-col rounded-xl bg-neutral-50 border border-neutral-200">
      {/* Column header */}
      <div className="flex items-center gap-2 px-3 py-2.5 border-b border-neutral-200">
        <div className="h-3 w-3 rounded-full flex-shrink-0" style={{ backgroundColor: color }} />
        <h3 className="text-sm font-semibold text-neutral-800 truncate">{title}</h3>
        <span className="ml-auto rounded-full bg-neutral-200 px-2 py-0.5 text-[11px] font-medium text-neutral-600">
          {customers.length}
        </span>
      </div>

      {/* Droppable area */}
      <Droppable droppableId={stageId}>
        {(provided, snapshot) => (
          <div
            ref={provided.innerRef}
            {...provided.droppableProps}
            className={`flex-1 space-y-2 overflow-y-auto p-2 transition-colors ${
              snapshot.isDraggingOver ? 'bg-primary-50/50' : ''
            }`}
            style={{ minHeight: 80, maxHeight: 'calc(100vh - 220px)' }}
          >
            {customers.length === 0 && !snapshot.isDraggingOver && (
              <p className="py-6 text-center text-xs text-neutral-400">{t.crm.pipeline.noCustomers}</p>
            )}
            {customers.map((customer, index) => (
              <KanbanCard key={customer.id} customer={customer} index={index} />
            ))}
            {provided.placeholder}
          </div>
        )}
      </Droppable>
    </div>
  );
}
