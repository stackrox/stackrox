import React, { ReactElement } from 'react';

import { VulnerabilityState } from 'types/cve.proto';
import { DeferralRequest, RequestStatus } from 'types/vuln_request.proto';
import { getDistanceStrict } from 'utils/dateUtils';

export type VulnRequestedActionProps = {
    targetState: VulnerabilityState;
    requestStatus: RequestStatus;
    deferralReq: DeferralRequest | null;
    updatedDeferralReq?: DeferralRequest;
    currentDate: Date;
};

function VulnRequestedAction({
    targetState,
    requestStatus,
    deferralReq = { expiresWhenFixed: undefined, expiresOn: undefined },
    updatedDeferralReq = { expiresWhenFixed: undefined, expiresOn: undefined },
    currentDate,
}: VulnRequestedActionProps): ReactElement {
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

    const chosenDeferralReq =
        requestStatus === 'APPROVED_PENDING_UPDATE' && updatedDeferralReq
            ? updatedDeferralReq
            : deferralReq;
    const expiresWhenFixed = chosenDeferralReq?.expiresWhenFixed || false;
    const expiresOn = chosenDeferralReq?.expiresOn || null;

    if (expiresWhenFixed) {
        action = '(until fixed)';
    } else if (expiresOn) {
        const expiresOnDistance = getDistanceStrict(expiresOn, currentDate, {
            partialMethod: 'ceil',
            unit: 'd',
        });
        action = `(${expiresOnDistance})`;
    }

    return (
        <div>
            {type} {action}
        </div>
    );
}

export default VulnRequestedAction;
