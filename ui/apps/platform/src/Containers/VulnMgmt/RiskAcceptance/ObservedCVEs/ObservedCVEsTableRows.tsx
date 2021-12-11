import { Spinner } from '@patternfly/react-core';
import { IActions, Td, Tr } from '@patternfly/react-table';
import { TdSelectType } from '@patternfly/react-table/dist/esm/components/Table/base';
import CVSSScoreLabel from 'Components/PatternFly/CVSSScoreLabel';
import DateTimeFormat from 'Components/PatternFly/DateTimeFormat';
import VulnerabilitySeverityLabel from 'Components/PatternFly/VulnerabilitySeverityLabel';
import React from 'react';
import AffectedComponentsButton from '../AffectedComponents/AffectedComponentsButton';
import { Vulnerability } from '../imageVulnerabilities.graphql';
import useVulnerabilityRequests from '../useVulnerabilityRequests';

export type ObservedCVEsTableRowsProps = {
    row: Vulnerability;
    rowIndex: number;
    onSelect: TdSelectType['onSelect'];
    selected: boolean[];
    actions: IActions;
    page: number;
    perPage: number;
};

/*
 * Backend mentioned that we need to fetch vuln requests for each vulnerability (row), so we need
 * to have this component in order to use the `useVulnerabilityRequests` hook. For now, we'll
 * have to do things this way until we get a better understanding of the performance implications
 * of adding a resolver under vulnerabilities to get the vulnerability requests
 */
function ObservedCVEsTableRows({
    row,
    rowIndex,
    onSelect,
    selected,
    actions,
    page,
    perPage,
}: ObservedCVEsTableRowsProps) {
    const { isLoading, data } = useVulnerabilityRequests({
        query: `CVE:${row.cve}`,
        pagination: {
            limit: perPage,
            offset: (page - 1) * perPage,
            sortOption: {
                field: 'id',
                reversed: false,
            },
        },
    });

    const vulnerabilityRequest = data?.results[0];

    let vulnerabilityRequestState = '-';
    if (vulnerabilityRequest?.targetState === 'DEFERRED') {
        vulnerabilityRequestState = 'Pending Deferral';
    } else if (vulnerabilityRequest?.targetState === 'FALSE_POSITIVE') {
        vulnerabilityRequestState = 'Pending False Positive';
    }

    return (
        <Tr key={rowIndex}>
            <Td
                select={{
                    rowIndex,
                    onSelect,
                    isSelected: selected[rowIndex],
                }}
            />
            <Td dataLabel="Cell">{row.cve}</Td>
            <Td dataLabel="Fixable">{row.isFixable ? 'Yes' : 'No'}</Td>
            <Td dataLabel="Severity">
                <VulnerabilitySeverityLabel severity={row.severity} />
            </Td>
            <Td dataLabel="CVSS score">
                <CVSSScoreLabel cvss={row.cvss} scoreVersion={row.scoreVersion} />
            </Td>
            <Td dataLabel="Affected components">
                <AffectedComponentsButton components={row.components} />
            </Td>
            <Td dataLabel="Discovered">
                <DateTimeFormat time={row.discoveredAtImage} />
            </Td>
            <Td dataLabel="Request State">
                {isLoading ? <Spinner size="sm" /> : vulnerabilityRequestState}
            </Td>
            <Td
                className="pf-u-text-align-right"
                actions={{
                    items: actions,
                }}
            />
        </Tr>
    );
}

export default ObservedCVEsTableRows;
