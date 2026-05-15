'use client';

import { Draggable } from '@hello-pangea/dnd';
import { Phone, Mail } from 'lucide-react';
import { useRouter } from 'next/navigation';
import { Customer } from '@/lib/api';
import { useTranslations } from '@/lib/i18n';

interface KanbanCardProps {
  customer: Customer;
  index: number;
}

export function KanbanCard({ customer, index }: KanbanCardProps) {
  const t = useTranslations();
  const router = useRouter();

  return (
    <Draggable draggableId={customer.id} index={index}>
      {(provided, snapshot) => (
        <div
          ref={provided.innerRef}
          {...provided.draggableProps}
          {...provided.dragHandleProps}
          onClick={() => router.push(`/dashboard/clients/${customer.id}`)}
          className={`rounded-lg border bg-white p-3 shadow-sm transition-shadow cursor-pointer hover:shadow-md ${
            snapshot.isDragging ? 'shadow-lg ring-2 ring-primary-300' : 'border-neutral-200'
          }`}
        >
          {/* Name */}
          <p className="text-sm font-medium text-neutral-900 truncate">{customer.name}</p>

          {/* Contact */}
          <div className="mt-1.5 space-y-0.5">
            {customer.phone && (
              <div className="flex items-center gap-1.5 text-xs text-neutral-500">
                <Phone className="h-3 w-3 flex-shrink-0" />
                <span className="truncate">{customer.phone}</span>
              </div>
            )}
            {customer.email && (
              <div className="flex items-center gap-1.5 text-xs text-neutral-500">
                <Mail className="h-3 w-3 flex-shrink-0" />
                <span className="truncate">{customer.email}</span>
              </div>
            )}
          </div>

          {/* Stats row */}
          <div className="mt-2 flex items-center gap-2 text-xs text-neutral-400">
            {customer.total_visits > 0 && (
              <span>{t.crm.pipeline.visits.replace('{n}', String(customer.total_visits))}</span>
            )}
            {(customer.lifetime_value ?? 0) > 0 && (
              <span className="font-medium text-emerald-600">${customer.lifetime_value?.toFixed(0)}</span>
            )}
          </div>

          {/* Tags */}
          {customer.tags && customer.tags.length > 0 && (
            <div className="mt-2 flex flex-wrap gap-1">
              {customer.tags.slice(0, 3).map(tag => (
                <span
                  key={tag}
                  className="inline-block rounded-full bg-neutral-100 px-2 py-0.5 text-[10px] font-medium text-neutral-600"
                >
                  {tag}
                </span>
              ))}
              {customer.tags.length > 3 && (
                <span className="text-[10px] text-neutral-400">+{customer.tags.length - 3}</span>
              )}
            </div>
          )}
        </div>
      )}
    </Draggable>
  );
}
