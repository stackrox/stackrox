import React from 'react';
import { Link } from 'react-router-dom';
import { Td } from '@patternfly/react-table';

import { exceptionManagementPath } from 'routePaths';
import { getUrlQueryStringForSearchFilter } from 'utils/searchUtils';
import { VulnerabilityState } from 'types/cve.proto';
import { ensureExhaustive } from 'utils/type.utils';

function getExceptionManagementURL(
    cve: string,
    vulnerabilityState: Exclude<VulnerabilityState, 'OBSERVED'>
): string {
    const query = getUrlQueryStringForSearchFilter({ CVE: cve });

    switch (vulnerabilityState) {
        case 'DEFERRED':
            return `${exceptionManagementPath}/approved-deferrals?${query}`;
        case 'FALSE_POSITIVE':
            return `${exceptionManagementPath}/approved-false-positives?${query}`;
        default:
            return ensureExhaustive(vulnerabilityState);
    }
}

export type ExceptionDetailsCellProps = {
    cve: string;
    vulnerabilityState: Exclude<VulnerabilityState, 'OBSERVED'>;
};

function ExceptionDetailsCell({ cve, vulnerabilityState }: ExceptionDetailsCellProps) {
    return (
        <Td dataLabel="Request details">
            <Link to={getExceptionManagementURL(cve, vulnerabilityState)}>View</Link>
        </Td>
    );
}

export default ExceptionDetailsCell;
