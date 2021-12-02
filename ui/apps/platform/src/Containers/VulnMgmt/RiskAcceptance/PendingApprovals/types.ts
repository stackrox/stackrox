import { VulnerabilityRequest } from './pendingApprovals.graphql';

export type RiskAssessmentAction = 'APPROVE' | 'DENY' | 'CANCEL';
export type RiskAssessmentType = 'DEFERRAL' | 'FALSE_POSITIVE';

export type RequestsToBeAssessed = {
    type: RiskAssessmentType;
    action: RiskAssessmentAction;
    requests: VulnerabilityRequest[];
} | null;
