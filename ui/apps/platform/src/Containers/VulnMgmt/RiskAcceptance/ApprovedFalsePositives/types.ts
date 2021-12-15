export type ApprovedFalsePositiveRequestAction = 'UNDO';
export type ApprovedFalsePositiveRequestType = 'FALSE_POSITIVE';

export type ApprovedFalsePositiveRequestsToBeAssessed = {
    type: ApprovedFalsePositiveRequestType;
    action: ApprovedFalsePositiveRequestAction;
    requestIDs: string[];
} | null;
