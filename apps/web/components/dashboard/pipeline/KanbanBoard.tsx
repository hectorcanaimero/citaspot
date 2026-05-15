'use client';

import { useCallback } from 'react';
import { DragDropContext, DropResult } from '@hello-pangea/dnd';
import { PipelineStage, Customer } from '@/lib/api';
import { KanbanColumn } from './KanbanColumn';

interface KanbanBoardProps {
  stages: PipelineStage[];
  customersByStage: Record<string, Customer[]>;
  unassignedTitle: string;
  onMoveCustomer: (customerId: string, newStageId: string | null, stageName: string) => void;
}

const UNASSIGNED_ID = 'unassigned';

export function KanbanBoard({ stages, customersByStage, unassignedTitle, onMoveCustomer }: KanbanBoardProps) {
  const handleDragEnd = useCallback(
    (result: DropResult) => {
      const { draggableId, destination, source } = result;
      if (!destination) return;
      if (destination.droppableId === source.droppableId) return;

      const newStageId = destination.droppableId === UNASSIGNED_ID ? null : destination.droppableId;
      const stageName =
        destination.droppableId === UNASSIGNED_ID
          ? unassignedTitle
          : stages.find(s => s.id === destination.droppableId)?.name ?? '';

      onMoveCustomer(draggableId, newStageId, stageName);
    },
    [stages, unassignedTitle, onMoveCustomer],
  );

  return (
    <DragDropContext onDragEnd={handleDragEnd}>
      <div className="flex gap-3 overflow-x-auto pb-4">
        {/* Unassigned column */}
        <KanbanColumn
          stageId={UNASSIGNED_ID}
          title={unassignedTitle}
          color="#94a3b8"
          customers={customersByStage[UNASSIGNED_ID] ?? []}
        />

        {/* Stage columns */}
        {stages.map(stage => (
          <KanbanColumn
            key={stage.id}
            stageId={stage.id}
            title={stage.name}
            color={stage.color}
            customers={customersByStage[stage.id] ?? []}
          />
        ))}
      </div>
    </DragDropContext>
  );
}
