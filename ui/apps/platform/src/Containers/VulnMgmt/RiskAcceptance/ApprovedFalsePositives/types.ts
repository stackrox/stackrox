import { VulnerabilityRequest } from '../vulnerabilityRequests.graphql';

export type ApprovedFalsePositiveRequestAction = 'UNDO';
export type ApprovedFalsePositiveRequestType = 'FALSE_POSITIVE';

export type ApprovedFalsePositiveRequestsToBeAssessed = {
    type: ApprovedFalsePositiveRequestType;
    action: ApprovedFalsePositiveRequestAction;
    requests: VulnerabilityRequest[];
} | null;
