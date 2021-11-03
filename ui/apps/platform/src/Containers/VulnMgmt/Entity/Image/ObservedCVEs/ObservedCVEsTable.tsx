/* eslint-disable no-nested-ternary */
/* eslint-disable react/no-array-index-key */
import React, { ReactElement } from 'react';
import { TableComposable, Thead, Tbody, Tr, Th, Td, IActions } from '@patternfly/react-table';
import VulnerabilitySeverityLabel from 'Components/PatternFly/VulnerabilitySeverityLabel';
import CVSSScoreLabel from 'Components/PatternFly/CVSSScoreLabel';
import { VulnerabilitySeverity } from 'messages/common';

export type ObservedCVERow = {
    cve: string;
    isFixable: boolean;
    severity: VulnerabilitySeverity;
    cvssScore: string;
    components: { name: string }[];
    discoveredAt: string;
};

export type ObservedCVEsTableProps = {
    rows: ObservedCVERow[];
    actions: IActions;
};

function ObservedCVEsTable({ rows, actions }: ObservedCVEsTableProps): ReactElement {
    return (
        <TableComposable aria-label="Observed CVEs Table" variant="compact" borders>
            <Thead>
                <Tr>
                    <Th>CVE</Th>
                    <Th>Fixable</Th>
                    <Th>Severity</Th>
                    <Th>CVSS score</Th>
                    <Th>Affected components</Th>
                    <Th>Discovered</Th>
                </Tr>
            </Thead>
            <Tbody>
                {rows.map((row, rowIndex) => (
                    <Tr key={rowIndex}>
                        <Td dataLabel="Cell">{row.cve}</Td>
                        <Td dataLabel="Fixable">{row.isFixable ? 'Yes' : 'No'}</Td>
                        <Td dataLabel="Severity">
                            <VulnerabilitySeverityLabel severity={row.severity} />
                        </Td>
                        <Td dataLabel="CVSS score">
                            <CVSSScoreLabel cvss={row.cvssScore} />
                        </Td>
                        <Td dataLabel="Affected components">{row.components.length}</Td>
                        <Td dataLabel="Discovered">{row.discoveredAt}</Td>
                        <Td
                            className="pf-u-text-align-right"
                            actions={{
                                items: actions,
                            }}
                        />
                    </Tr>
                ))}
            </Tbody>
        </TableComposable>
    );
}

export default ObservedCVEsTable;
