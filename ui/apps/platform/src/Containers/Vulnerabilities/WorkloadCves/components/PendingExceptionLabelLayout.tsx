import React from 'react';
import { Link } from 'react-router-dom-v5-compat';
import { Flex, FlexItem, Label } from '@patternfly/react-core';
import { OutlinedClockIcon } from '@patternfly/react-icons';

import { VulnerabilityState } from 'types/cve.proto';
import { exceptionManagementPath } from 'routePaths';
import { getUrlQueryStringForSearchFilter } from 'utils/searchUtils';

export type PendingExceptionLabelLayoutProps = {
    children: React.ReactNode;
    hasPendingException: boolean;
    cve: string;
    vulnerabilityState: VulnerabilityState;
};

/**
 * ‘Pending exception’ label layout for use in tables. Conditionally renders a label
 * with a link to the exception request page if the vulnerability has a pending exception.
 *
 * @param children - The table cell contents to render before the label
 * @param hasPendingException - Whether the vulnerability has a pending exception
 * @param cve - The CVE ID of the vulnerability
 * @param vulnerabilityState - The vulnerability state
 */
function PendingExceptionLabelLayout({
    children,
    hasPendingException,
    cve,
    vulnerabilityState,
}: PendingExceptionLabelLayoutProps) {
    const query = getUrlQueryStringForSearchFilter({ CVE: [cve] });
    const url = `${exceptionManagementPath}/pending-requests?${query}`;
    return (
        <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsXs' }}>
            {children}
            {hasPendingException && (
                <FlexItem>
                    <Link to={url}>
                        <Label
                            color="blue"
                            isCompact
                            icon={<OutlinedClockIcon />}
                            variant="outline"
                        >
                            {vulnerabilityState === 'OBSERVED'
                                ? 'Pending exception'
                                : 'Pending update'}
                        </Label>
                    </Link>
                </FlexItem>
            )}
        </Flex>
    );
}

export default PendingExceptionLabelLayout;
