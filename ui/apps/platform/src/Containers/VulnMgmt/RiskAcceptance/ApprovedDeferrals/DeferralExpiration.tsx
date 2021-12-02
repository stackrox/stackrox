import React, { ReactElement } from 'react';

import { DeferralRequest } from 'types/vuln_request.proto';
import { getDistanceStrictAsPhrase } from 'utils/dateUtils';

export type DeferralExpirationProps = {
    deferralReq: DeferralRequest;
};

function DeferralExpiration({ deferralReq }: DeferralExpirationProps): ReactElement {
    let expiration = '';
    if (deferralReq?.expiresWhenFixed) {
        expiration = 'when fixed';
    } else {
        expiration = getDistanceStrictAsPhrase(deferralReq.expiresOn);
    }

    return <span>{expiration}</span>;
}

export default DeferralExpiration;
