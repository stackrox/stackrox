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
        expirationDate = 'Never';
    } else {
        // if "updatedDeferralReq" is not passed then default to "deferralReq"
        const { expiresWhenFixed, expiresOn } =
            requestStatus === 'APPROVED_PENDING_UPDATE' && updatedDeferralReq
                ? updatedDeferralReq
                : deferralReq;
        if (expiresWhenFixed) {
            expirationDate = 'When fixed';
            // The backend returns the following date when the deferral request is indefinitely
            // We should ideally return a null
        } else if (expiresOn && expiresOn !== '1970-01-01T00:00:00Z') {
            expirationDate = getDate(expiresOn);
        } else {
            expirationDate = 'Never';
        }
    }
    return <div>{expirationDate}</div>;
}

export default DeferralExpirationDate;
