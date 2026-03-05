import type { HistoryAction } from 'hooks/useURLParameter';

export const userWorkloadWorkflowView = 'Applications view';
export const platformWorkflowView = 'Platform view';
export const nodeWorkflowView = 'Node view';
export const fullWorkflowView = 'Full view';

export const filteredWorkflowViews = [
    userWorkloadWorkflowView,
    platformWorkflowView,
    nodeWorkflowView,
    fullWorkflowView,
] as const;

export type FilteredWorkflowView = (typeof filteredWorkflowViews)[number];

export type SetFilteredWorkflowView = (nextValue: unknown, historyAction?: HistoryAction) => void;
