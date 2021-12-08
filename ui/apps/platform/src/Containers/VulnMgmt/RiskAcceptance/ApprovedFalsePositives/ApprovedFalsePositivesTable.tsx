import React, { ReactElement, useState } from 'react';
import { TableComposable, Thead, Tbody, Tr, Th, Td } from '@patternfly/react-table';
import {
    Divider,
    DropdownItem,
    Toolbar,
    ToolbarContent,
    ToolbarItem,
} from '@patternfly/react-core';

import RequestCommentsButton from 'Containers/VulnMgmt/RiskAcceptance/RequestComments/RequestCommentsButton';
import BulkActionsDropdown from 'Components/PatternFly/BulkActionsDropdown';
import useTableSelection from 'hooks/useTableSelection';
import { VulnerabilityRequest } from '../vulnerabilityRequests.graphql';
import { ApprovedFalsePositiveRequestsToBeAssessed } from './types';
import useRiskAcceptance from '../useRiskAcceptance';
import VulnerabilityRequestScope from '../PendingApprovals/VulnerabilityRequestScope';
import UndoVulnRequestModal from '../UndoVulnRequestModal';
import ApprovedFalsePositiveActionsColumn from './ApprovedFalsePositiveActionsColumn';
import ImpactedEntities from '../ImpactedEntities';
import RequestedAction from '../RequestedAction';

export type ApprovedFalsePositivesTableProps = {
    rows: VulnerabilityRequest[];
    updateTable: () => void;
    isLoading: boolean;
};

function ApprovedFalsePositivesTable({
    rows,
    updateTable,
}: ApprovedFalsePositivesTableProps): ReactElement {
    const {
        selected,
        allRowsSelected,
        numSelected,
        onSelect,
        onSelectAll,
        getSelectedIds,
        onClearAll,
    } = useTableSelection<VulnerabilityRequest>(rows);
    const [requestsToBeAssessed, setRequestsToBeAssessed] =
        useState<ApprovedFalsePositiveRequestsToBeAssessed>(null);
    const { undoVulnRequests } = useRiskAcceptance({
        requests: requestsToBeAssessed?.requests || [],
    });

    function cancelAssessment() {
        setRequestsToBeAssessed(null);
    }

    async function completeAssessment() {
        onClearAll();
        setRequestsToBeAssessed(null);
        updateTable();
    }

    const selectedIds = getSelectedIds();
    const selectedFalsePositiveRequests = rows.filter((row) => selectedIds.includes(row.id));

    return (
        <>
            <Toolbar>
                <ToolbarContent>
                    <ToolbarItem variant="separator" />
                    <ToolbarItem>
                        <BulkActionsDropdown isDisabled={numSelected === 0}>
                            <DropdownItem
                                key="undo false positives"
                                component="button"
                                onClick={() =>
                                    setRequestsToBeAssessed({
                                        type: 'FALSE_POSITIVE',
                                        action: 'UNDO',
                                        requests: selectedFalsePositiveRequests,
                                    })
                                }
                                isDisabled={selectedFalsePositiveRequests.length === 0}
                            >
                                Reobserve CVEs ({selectedFalsePositiveRequests.length})
                            </DropdownItem>
                        </BulkActionsDropdown>
                    </ToolbarItem>
                </ToolbarContent>
            </Toolbar>
            <Divider component="div" />
            <TableComposable aria-label="Approved False Positives Table" variant="compact" borders>
                <Thead>
                    <Tr>
                        <Th
                            select={{
                                onSelect: onSelectAll,
                                isSelected: allRowsSelected,
                            }}
                        />
                        <Th>Requested Entity</Th>
                        <Th>Requested Action</Th>
                        <Th>Scope</Th>
                        <Th>Impacted Entities</Th>
                        <Th>Apply to</Th>
                        <Th>Comments</Th>
                        <Th>Requestor</Th>
                    </Tr>
                </Thead>
                <Tbody>
                    {rows.map((row, rowIndex) => {
                        return (
                            <Tr key={row.id}>
                                <Td
                                    select={{
                                        rowIndex,
                                        onSelect,
                                        isSelected: selected[rowIndex],
                                    }}
                                />
                                <Td dataLabel="Requested Entity">{row.cves.ids[0]}</Td>
                                <Td dataLabel="Requested Action">
                                    <RequestedAction
                                        targetState={row.targetState}
                                        requestStatus={row.status}
                                        deferralReq={row.deferralReq}
                                        updatedDeferralReq={row.updatedDeferralReq}
                                    />
                                </Td>
                                <Td dataLabel="Scope">
                                    {row.scope.imageScope ? 'image' : 'global'}
                                </Td>
                                <Td dataLabel="Impacted entities">
                                    <ImpactedEntities
                                        deploymentCount={row.deploymentCount}
                                        imageCount={row.imageCount}
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
                                <Td className="pf-u-text-align-right">
                                    <ApprovedFalsePositiveActionsColumn
                                        row={row}
                                        setRequestsToBeAssessed={setRequestsToBeAssessed}
                                    />
                                </Td>
                            </Tr>
                        );
                    })}
                </Tbody>
            </TableComposable>
            <UndoVulnRequestModal
                type="FALSE_POSITIVE"
                isOpen={requestsToBeAssessed?.action === 'UNDO'}
                numRequestsToBeAssessed={requestsToBeAssessed?.requests.length || 0}
                onSendRequest={undoVulnRequests}
                onCompleteRequest={completeAssessment}
                onCancel={cancelAssessment}
            />
        </>
    );
}

export default ApprovedFalsePositivesTable;
