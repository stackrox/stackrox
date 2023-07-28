import { ClipboardCopy, Popover } from '@patternfly/react-core';
import React, { ReactElement } from 'react';
import { InfoCircleIcon } from '@patternfly/react-icons';

import { getQueryString } from 'utils/queryStringUtils';

import { vulnManagementPendingApprovalsPath } from '../pathsForRiskAcceptance';

type PendingApprovalPopoverProps = {
    vulnRequestId: string;
};

function PendingApprovalPopover({ vulnRequestId }: PendingApprovalPopoverProps): ReactElement {
    const popoverRef = React.useRef<HTMLButtonElement>(null);

    const queryString = getQueryString({
        search: {
            'Request ID': vulnRequestId,
        },
    });
    const pendingRequestURL = `${window.location.origin}${vulnManagementPendingApprovalsPath}${queryString}`;

    return (
        <span>
            <Popover
                aria-label="Pending approval popover"
                headerContent={<div>Pending Approval</div>}
                bodyContent={<div>Use the link to share your request with your approver</div>}
                footerContent={
                    <ClipboardCopy isReadOnly hoverTip="Copy" clickTip="Copied">
                        {pendingRequestURL}
                    </ClipboardCopy>
                }
                reference={popoverRef}
            />
            <button type="button" ref={popoverRef}>
                <InfoCircleIcon
                    className="pf-u-info-color-100"
                    aria-label="Pending approval icon"
                />
            </button>
        </span>
    );
}

export default PendingApprovalPopover;
