import React from 'react';

import { Link } from 'react-router-dom-v5-compat';
import { Label } from '@patternfly/react-core';
import { OutlinedClockIcon } from '@patternfly/react-icons';

import type { VulnerabilityState } from 'types/cve.proto';
import { getUrlQueryStringForSearchFilter } from 'utils/searchUtils';
import { exceptionManagementPath } from 'routePaths';

export type PendingExceptionLabelProps = {
    cve: string;
    isCompact: boolean; // true for table
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
function PendingExceptionLabel({ cve, isCompact, vulnerabilityState }: PendingExceptionLabelProps) {
    const query = getUrlQueryStringForSearchFilter({ CVE: [cve] });
    const url = `${exceptionManagementPath}/pending-requests?${query}`;

    return (
        <Link to={url}>
            <Label
                color="blue"
                isCompact={isCompact}
                icon={<OutlinedClockIcon />}
                variant="outline"
            >
                {vulnerabilityState === 'OBSERVED' ? 'Pending exception' : 'Pending update'}
            </Label>
        </Link>
    );
}

export default PendingExceptionLabel;
