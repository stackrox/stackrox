import { VulnerabilityRequest } from './pendingApprovals.graphql';

export type RiskAssessmentAction =
    | 'APPROVE_DEFERRAL'
    | 'DENY_DEFERRAL'
    | 'APPROVE_FALSE_POSITIVE'
    | 'DENY_FALSE_POSITIVE';

export type RequestsToBeAssessed = {
    type: RiskAssessmentAction;
    requests: VulnerabilityRequest[];
} | null;
