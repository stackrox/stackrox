export type ApprovedDeferralRequestAction = 'UPDATE' | 'UNDO';
export type ApprovedDeferralRequestType = 'DEFERRAL';

export type ApprovedDeferralRequestsToBeAssessed = {
    type: ApprovedDeferralRequestType;
    action: ApprovedDeferralRequestAction;
    requestIDs: string[];
} | null;
