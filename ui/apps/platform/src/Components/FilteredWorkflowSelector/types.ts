import { HistoryAction } from 'hooks/useURLParameter';

export const filteredWorkflowStates = ['Application view', 'Platform view', 'Full view'] as const;

export type FilteredWorkflowState = (typeof filteredWorkflowStates)[number];

export type SetFilteredWorkflowState = (nextValue: unknown, historyAction?: HistoryAction) => void;
