import React, { ReactElement } from 'react';

import { VulnerabilityState } from 'types/cve.proto';
import { DeferralRequest, RequestStatus } from 'types/vuln_request.proto';
import { getDistanceStrict } from 'utils/dateUtils';

export type RequestedActionProps = {
    targetState: VulnerabilityState;
    requestStatus: RequestStatus;
    deferralReq: DeferralRequest;
    updatedDeferralReq?: DeferralRequest;
};

function RequestedAction({
    targetState,
    requestStatus,
    deferralReq,
    updatedDeferralReq,
}: RequestedActionProps): ReactElement {
    let type = '';
    let action = '';

    if (targetState === 'FALSE_POSITIVE') {
        type = 'False positive';
    } else if (
        targetState === 'DEFERRED' &&
        requestStatus === 'APPROVED_PENDING_UPDATE' &&
        updatedDeferralReq
    ) {
        type = 'Deferral pending update';
    } else if (targetState === 'DEFERRED') {
        type = 'Deferral';
    }

    // if "updatedDeferralReq" is not passed then default to "deferralReq"
    const { expiresWhenFixed, expiresOn } =
        requestStatus === 'APPROVED_PENDING_UPDATE' && updatedDeferralReq
            ? updatedDeferralReq
            : deferralReq;
    if (expiresWhenFixed) {
        action = '(until fixed)';
    } else if (expiresOn) {
        action = `(${getDistanceStrict(expiresOn, new Date())})`;
    }

    return (
        <div>
            {type} {action}
        </div>
    );
}

export default RequestedAction;
