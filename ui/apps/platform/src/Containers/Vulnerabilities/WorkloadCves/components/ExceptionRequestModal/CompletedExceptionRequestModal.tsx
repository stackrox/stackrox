import React from 'react';
import {
    Button,
    ClipboardCopy,
    DescriptionList,
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
    Flex,
    Modal,
    Text,
    pluralize,
} from '@patternfly/react-core';
import differenceInCalendarDays from 'date-fns/difference_in_calendar_days';

import {
    BaseVulnerabilityException,
    VulnerabilityDeferralException,
    isDeferralException,
    isFalsePositiveException,
} from 'services/VulnerabilityExceptionService';
import { getDate } from 'utils/dateUtils';
import { ensureExhaustive } from 'utils/type.utils';
import { exceptionManagementPath } from 'routePaths';

function scopeDisplay(scope: BaseVulnerabilityException['scope']): string {
    if (scope.imageScope.remote === '.*') {
        return 'All images';
    }
    if (scope.imageScope.tag === '.*') {
        return `${scope.imageScope.registry}/${scope.imageScope.remote}:*`;
    }
    return `${scope.imageScope.registry}/${scope.imageScope.remote}:${scope.imageScope.tag}`;
}

function expiryDisplay(
    expiry: VulnerabilityDeferralException['deferralRequest']['expiry']
): string {
    const { expiryType } = expiry;
    switch (expiryType) {
        case 'ALL_CVE_FIXABLE':
            return 'When all CVEs are fixable';
        case 'ANY_CVE_FIXABLE':
            return 'When any CVE is fixable';
        case 'TIME': {
            if (expiry.expiresOn) {
                // Since the expiry here will always be in the future, we don't need to check which date is earlier
                const daysUntilExpiration = differenceInCalendarDays(expiry.expiresOn, new Date());
                return `${getDate(expiry.expiresOn)} (${pluralize(daysUntilExpiration, 'day')})`;
            }
            return 'Never';
        }
        default:
            return ensureExhaustive(expiryType);
    }
}

export type CompletedExceptionRequestModalProps = {
    isUpdate?: boolean;
    exceptionRequest: BaseVulnerabilityException;
    onClose: () => void;
};

function CompletedExceptionRequestModal({
    isUpdate = false,
    exceptionRequest,
    onClose,
}: CompletedExceptionRequestModalProps) {
    let title = '';
    const titleAction = isUpdate ? 'Update' : 'Request';
    let requestedAction = '';

    if (isDeferralException(exceptionRequest)) {
        title = `${titleAction} for deferral has been submitted`;
        requestedAction = 'Deferral';
    }
    if (isFalsePositiveException(exceptionRequest)) {
        title = `${titleAction} for false positive has been submitted`;
        requestedAction = 'False positive';
    }

    const exceptionRequestURL = `${window.location.origin}${exceptionManagementPath}/requests/${exceptionRequest.id}`;

    return (
        <Modal
            onClose={onClose}
            title={title}
            isOpen
            variant="medium"
            actions={[
                <Button key="confirm" variant="primary" onClick={onClose}>
                    Close
                </Button>,
            ]}
        >
            <Flex direction={{ default: 'column' }}>
                <Text>Use this link to share and discuss your request with your approver.</Text>
                <ClipboardCopy isReadOnly hoverTip="Copy" clickTip="Copied">
                    {exceptionRequestURL}
                </ClipboardCopy>
                <DescriptionList columnModifier={{ default: '2Col' }} className="pf-u-pt-md">
                    <DescriptionListGroup>
                        <DescriptionListTerm>Requested action</DescriptionListTerm>
                        <DescriptionListDescription>{requestedAction}</DescriptionListDescription>
                    </DescriptionListGroup>
                    <DescriptionListGroup>
                        <DescriptionListTerm>Scope</DescriptionListTerm>
                        <DescriptionListDescription>
                            {scopeDisplay(exceptionRequest.scope)}
                        </DescriptionListDescription>
                    </DescriptionListGroup>
                    <DescriptionListGroup>
                        <DescriptionListTerm>Requested</DescriptionListTerm>
                        <DescriptionListDescription>
                            {getDate(exceptionRequest.createdAt)}
                        </DescriptionListDescription>
                    </DescriptionListGroup>
                    <DescriptionListGroup>
                        <DescriptionListTerm>CVEs</DescriptionListTerm>
                        <DescriptionListDescription>
                            {exceptionRequest.cves.length}
                        </DescriptionListDescription>
                    </DescriptionListGroup>
                    {isDeferralException(exceptionRequest) && (
                        <DescriptionListGroup>
                            <DescriptionListTerm>Expires</DescriptionListTerm>
                            <DescriptionListDescription>
                                {expiryDisplay(exceptionRequest.deferralRequest.expiry)}
                            </DescriptionListDescription>
                        </DescriptionListGroup>
                    )}
                </DescriptionList>
            </Flex>
        </Modal>
    );
}

export default CompletedExceptionRequestModal;
