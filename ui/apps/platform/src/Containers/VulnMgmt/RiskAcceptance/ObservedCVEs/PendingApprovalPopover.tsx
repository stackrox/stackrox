import { ClipboardCopy, Popover } from '@patternfly/react-core';
import React, { ReactElement } from 'react';
import { InfoCircleIcon } from '@patternfly/react-icons';

import PopoverBodyContent from 'Components/PopoverBodyContent';
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
                bodyContent={
                    <PopoverBodyContent
                        headerContent="Pending Approval"
                        bodyContent="Use the link to share your request with your approver"
                        footerContent={
                            <ClipboardCopy isReadOnly hoverTip="Copy" clickTip="Copied">
                                {pendingRequestURL}
                            </ClipboardCopy>
                        }
                    />
                }
                triggerRef={popoverRef}
            />
            <button type="button" ref={popoverRef}>
                <InfoCircleIcon
                    className="pf-v5-u-info-color-100"
                    aria-label="Pending approval icon"
                />
            </button>
        </span>
    );
}

export default PendingApprovalPopover;
