import React, { ReactElement } from 'react';

import { VulnerabilityState } from 'types/cve.proto';
import { DeferralRequest, RequestStatus } from 'types/vuln_request.proto';
import { getDate } from 'utils/dateUtils';

export type DeferralExpirationDateProps = {
    targetState: VulnerabilityState;
    requestStatus: RequestStatus;
    deferralReq: DeferralRequest;
    updatedDeferralReq?: DeferralRequest;
};

function DeferralExpirationDate({
    targetState,
    requestStatus,
    deferralReq,
    updatedDeferralReq,
}: DeferralExpirationDateProps): ReactElement {
    let expirationDate = '';
    if (targetState === 'FALSE_POSITIVE') {
        expirationDate = 'never';
    } else {
        // if "updatedDeferralReq" is not passed then default to "deferralReq"
        const { expiresWhenFixed, expiresOn } =
            requestStatus === 'APPROVED_PENDING_UPDATE' && updatedDeferralReq
                ? updatedDeferralReq
                : deferralReq;
        if (expiresWhenFixed) {
            expirationDate = 'when fixed';
        } else if (expiresOn) {
            expirationDate = getDate(expiresOn);
        } else {
            expirationDate = 'never';
        }
    }
    return <div>{expirationDate}</div>;
}

export default DeferralExpirationDate;
