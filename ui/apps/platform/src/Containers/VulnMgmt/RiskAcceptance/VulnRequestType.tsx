import React, { ReactElement } from 'react';

import { vulnerabilityStateLabels } from 'messages/vulnerability';
import { VulnerabilityState } from 'types/cve.proto';
import { RequestStatus } from 'types/vuln_request.proto';

export type VulnRequestTypeProps = {
    targetState: VulnerabilityState;
    requestStatus: RequestStatus;
};

function VulnRequestType({ targetState, requestStatus }: VulnRequestTypeProps): ReactElement {
    const type = vulnerabilityStateLabels[targetState];
    const isDeferralUpdated =
        targetState === 'DEFERRED' && requestStatus === 'APPROVED_PENDING_UPDATE';
    const text = isDeferralUpdated ? `${type} (pending update)` : type;
    return <span>{text}</span>;
}

export default VulnRequestType;
