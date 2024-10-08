import { HistoryAction } from 'hooks/useURLParameter';

export const filteredWorkflowViews = ['Application view', 'Platform view', 'Full view'] as const;

export type FilteredWorkflowView = (typeof filteredWorkflowViews)[number];

export type SetFilteredWorkflowView = (nextValue: unknown, historyAction?: HistoryAction) => void;
