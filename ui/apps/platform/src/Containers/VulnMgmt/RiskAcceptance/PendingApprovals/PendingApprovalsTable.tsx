// @TODO: We might be able to reuse the same logic for each table
import React, { ReactElement } from 'react';
import { TableComposable, Thead, Tbody, Tr, Th, Td } from '@patternfly/react-table';

import RequestCommentsButton from 'Containers/VulnMgmt/RiskAcceptance/RequestComments/RequestCommentsButton';
import { vulnerabilityStateLabels } from 'messages/vulnerability';
import { VulnerabilityRequest } from './pendingApprovals.graphql';
import RequestedAction from './RequestedAction';
import VulnerabilityRequestScope from './VulnerabilityRequestScope';

export type PendingApprovalsTableProps = {
    rows: VulnerabilityRequest[];
    isLoading: boolean;
};

function PendingApprovalsTable({ rows }: PendingApprovalsTableProps): ReactElement {
    return (
        <TableComposable aria-label="Pending Approvals Table" variant="compact" borders>
            <Thead>
                <Tr>
                    <Th>Requested Entity</Th>
                    <Th>Type</Th>
                    <Th>Scope</Th>
                    <Th>Impacted Entities</Th>
                    <Th>Requested Action</Th>
                    <Th>Apply to</Th>
                    <Th>Comments</Th>
                    <Th>Requestor</Th>
                </Tr>
            </Thead>
            <Tbody>
                {rows.map((row) => {
                    // @TODO: Remove actions not appropriate for the row
                    const actions = [
                        {
                            title: 'Approve Deferral',
                            onClick: (event) => {
                                event.preventDefault();
                            },
                        },
                        {
                            title: 'Approve False Positive',
                            onClick: (event) => {
                                event.preventDefault();
                            },
                        },
                        {
                            isSeparator: true,
                        },
                        {
                            title: 'Deny Deferral',
                            onClick: (event) => {
                                event.preventDefault();
                            },
                        },
                        {
                            title: 'Deny False Positive',
                            onClick: (event) => {
                                event.preventDefault();
                            },
                        },
                        {
                            isSeparator: true,
                        },
                        {
                            title: 'Cancel Deferral',
                            onClick: (event) => {
                                event.preventDefault();
                            },
                        },
                        {
                            title: 'Cancel False Positive',
                            onClick: (event) => {
                                event.preventDefault();
                            },
                        },
                    ];

                    return (
                        <Tr key={row.id}>
                            <Td dataLabel="Requested Entity">{row.cves.ids[0]}</Td>
                            <Td dataLabel="Type">{vulnerabilityStateLabels[row.targetState]}</Td>
                            <Td dataLabel="Scope">{row.scope.imageScope ? 'image' : 'global'}</Td>
                            <Td dataLabel="Impacted entities">-</Td>
                            <Td dataLabel="Requested Action">
                                <RequestedAction
                                    targetState={row.targetState}
                                    deferralReq={row.deferralReq}
                                />
                            </Td>
                            <Td dataLabel="Apply to">
                                <VulnerabilityRequestScope scope={row.scope} />
                            </Td>
                            <Td dataLabel="Comments">
                                <RequestCommentsButton
                                    comments={row.comments}
                                    cve={row.cves.ids[0]}
                                />
                            </Td>
                            <Td dataLabel="Requestor">{row.requestor.name}</Td>
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

export default PendingApprovalsTable;
