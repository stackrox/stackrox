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
import differenceInDays from 'date-fns/difference_in_days';

import {
    BaseVulnerabilityException,
    VulnerabilityDeferralException,
    isDeferralException,
    isFalsePositiveException,
} from 'services/VulnerabilityExceptionService';
import { getDate } from 'utils/dateUtils';
import { ensureExhaustive } from 'utils/type.utils';

function scopeDisplay(scope: BaseVulnerabilityException['scope']): string {
    if (scope.imageScope.remote === '.*') {
        return 'All images';
    }
    if (scope.imageScope.tag === '.*') {
        return `${scope.imageScope.registry}/${scope.imageScope.remote}:*`;
    }
    return `${scope.imageScope.registry}/${scope.imageScope.remote}:${scope.imageScope.tag}`;
}

function expiryDisplay(expiry: VulnerabilityDeferralException['deferralReq']['expiry']): string {
    const { expiryType } = expiry;
    switch (expiryType) {
        case 'ALL_CVE_FIXABLE':
            return 'When all CVEs are fixable';
        case 'ANY_CVE_FIXABLE':
            return 'When any CVE is fixable';
        case 'TIME': {
            if (expiry.expiresOn) {
                // Since the expiry here will always be in the future, we don't need to check which date is earlier
                const daysUntilExpiration = differenceInDays(expiry.expiresOn, new Date());
                return `${getDate(expiry.expiresOn)} (${pluralize(daysUntilExpiration, 'day')})`;
            }
            return 'Never';
        }
        default:
            return ensureExhaustive(expiryType);
    }
}

export type CompletedExceptionRequestModalProps = {
    exceptionRequest: BaseVulnerabilityException;
    onClose: () => void;
};

function CompletedExceptionRequestModal({
    exceptionRequest,
    onClose,
}: CompletedExceptionRequestModalProps) {
    let title = '';
    let requestedAction = '';

    if (isDeferralException(exceptionRequest)) {
        title = 'Request for deferral has been submitted';
        requestedAction = 'Deferral';
    }
    if (isFalsePositiveException(exceptionRequest)) {
        title = 'Request for false positive has been submitted';
        requestedAction = 'False positive';
    }

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
                <ClipboardCopy
                    isReadOnly
                    hoverTip="Copy"
                    clickTip="Copied"
                >{`todo.path.to.requests.page/${exceptionRequest.name}`}</ClipboardCopy>
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
                                {expiryDisplay(exceptionRequest.deferralReq.expiry)}
                            </DescriptionListDescription>
                        </DescriptionListGroup>
                    )}
                </DescriptionList>
            </Flex>
        </Modal>
    );
}

export default CompletedExceptionRequestModal;
