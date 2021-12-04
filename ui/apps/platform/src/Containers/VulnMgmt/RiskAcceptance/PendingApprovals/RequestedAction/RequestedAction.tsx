import React, { ReactElement } from 'react';

import { VulnerabilityState } from 'types/cve.proto';
import { DeferralRequest, RequestStatus } from 'types/vuln_request.proto';
import { getDistanceStrict } from 'utils/dateUtils';

export type RequestedActionProps = {
    targetState: VulnerabilityState;
    requestStatus: RequestStatus;
    deferralReq: DeferralRequest;
    updatedDeferralReq: DeferralRequest;
};

function getRequestedAction(
    targetState: VulnerabilityState,
    requestStatus: RequestStatus,
    deferralReq: DeferralRequest,
    updatedDeferralReq: DeferralRequest
) {
    if (targetState === 'FALSE_POSITIVE') {
        return 'Mark false positive';
    }
    const { expiresWhenFixed, expiresOn } =
        requestStatus === 'APPROVED_PENDING_UPDATE' ? updatedDeferralReq : deferralReq;
    if (expiresWhenFixed) {
        return 'Expire when fixed';
    }
    if (expiresOn) {
        return `Defer for ${getDistanceStrict(expiresOn, new Date())}`;
    }
    return 'N/A';
}

function RequestedAction({
    targetState,
    requestStatus,
    deferralReq,
    updatedDeferralReq,
}: RequestedActionProps): ReactElement {
    const requestedAction = getRequestedAction(
        targetState,
        requestStatus,
        deferralReq,
        updatedDeferralReq
    );
    return <div>{requestedAction}</div>;
}

export default RequestedAction;
