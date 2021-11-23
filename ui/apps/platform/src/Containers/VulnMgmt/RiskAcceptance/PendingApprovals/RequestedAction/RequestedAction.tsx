import React, { ReactElement } from 'react';

import { VulnerabilityState } from 'types/cve.proto';
import { DeferralRequest } from 'types/vuln_request.proto';
import { getDistanceStrict } from 'utils/dateUtils';

export type RequestedActionProps = {
    targetState: VulnerabilityState;
    deferralReq: DeferralRequest;
};

function getRequestedAction(targetState: VulnerabilityState, deferralReq: DeferralRequest) {
    if (targetState === 'FALSE_POSITIVE') {
        return 'Mark false positive';
    }
    if (deferralReq?.expiresWhenFixed) {
        return 'Expire when fixed';
    }
    if (deferralReq?.expiresOn) {
        return getDistanceStrict(deferralReq.expiresOn, new Date());
    }
    return 'N/A';
}

function RequestedAction({ targetState, deferralReq }: RequestedActionProps): ReactElement {
    const requestedAction = getRequestedAction(targetState, deferralReq);
    return <div>{requestedAction}</div>;
}

export default RequestedAction;
