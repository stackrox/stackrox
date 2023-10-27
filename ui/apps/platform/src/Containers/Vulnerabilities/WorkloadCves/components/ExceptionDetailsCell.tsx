import React from 'react';
import { Button } from '@patternfly/react-core';
import { Td } from '@patternfly/react-table';

import LinkShim from 'Components/PatternFly/LinkShim';
import { exceptionManagementPath } from 'routePaths';
import { getUrlQueryStringForSearchFilter } from 'utils/searchUtils';
import { VulnerabilityState } from 'types/cve.proto';
import { ensureExhaustive } from 'utils/type.utils';

function getExceptionManagementURL(
    cve: string,
    vulnerabilityState: Exclude<VulnerabilityState, 'OBSERVED'>
): string {
    const query = getUrlQueryStringForSearchFilter({ cve });

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
            <Button
                variant="link"
                isInline
                component={LinkShim}
                href={getExceptionManagementURL(cve, vulnerabilityState)}
            >
                View
            </Button>
        </Td>
    );
}

export default ExceptionDetailsCell;
