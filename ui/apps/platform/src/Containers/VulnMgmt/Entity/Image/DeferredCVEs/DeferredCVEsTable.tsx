/* eslint-disable no-nested-ternary */
/* eslint-disable react/no-array-index-key */
import React, { ReactElement } from 'react';
import { TableComposable, Thead, Tbody, Tr, Th, Td } from '@patternfly/react-table';

import { VulnerabilitySeverity } from 'messages/common';

import VulnerabilitySeverityLabel from 'Components/PatternFly/VulnerabilitySeverityLabel';
import { ComponentWhereCVEOccurs, VulnerabilityRequestComments } from '../types';

export type DeferredCVERow = {
    id: string;
    cve: string;
    severity: VulnerabilitySeverity;
    components: ComponentWhereCVEOccurs[];
    comments: VulnerabilityRequestComments[];
    expiresAt: string;
    applyTo: string;
    approver: string;
};

export type DeferredCVEsTableProps = {
    rows: DeferredCVERow[];
};

function DeferredCVEsTable({ rows }: DeferredCVEsTableProps): ReactElement {
    return (
        <TableComposable aria-label="Observed CVEs Table" variant="compact" borders>
            <Thead>
                <Tr>
                    <Th>CVE</Th>
                    <Th>Severity</Th>
                    <Th>Affected Components</Th>
                    <Th>Comments</Th>
                    <Th>Expiration</Th>
                    <Th>Apply to</Th>
                    <Th>Approver</Th>
                </Tr>
            </Thead>
            <Tbody>
                {rows.map((row, rowIndex) => {
                    const actions = [
                        {
                            title: 'Cancel deferral',
                            onClick: (event) => {
                                event.preventDefault();
                            },
                        },
                    ];

                    return (
                        <Tr key={rowIndex}>
                            <Td dataLabel="Cell">{row.cve}</Td>
                            <Td dataLabel="Severity">
                                <VulnerabilitySeverityLabel severity={row.severity} />
                            </Td>
                            <Td dataLabel="Affected components">
                                {row.components.length} components
                            </Td>
                            <Td dataLabel="Comments">{row.comments.length} comments</Td>
                            <Td dataLabel="Expiration">{row.expiresAt}</Td>
                            <Td dataLabel="Apply to">{row.applyTo}</Td>
                            <Td dataLabel="Approver">{row.approver}</Td>
                            <Td
                                className="pf-u-text-align-right"
                                actions={{
                                    items: actions,
                                }}
                            />
                        </Tr>
                    );
                })}
            </Tbody>
        </TableComposable>
    );
}

export default DeferredCVEsTable;
