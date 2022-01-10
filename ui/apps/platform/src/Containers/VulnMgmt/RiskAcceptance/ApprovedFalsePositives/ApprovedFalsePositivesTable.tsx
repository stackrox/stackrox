import React, { ReactElement, useState } from 'react';
import { TableComposable, Thead, Tbody, Tr, Th, Td } from '@patternfly/react-table';
import {
    Divider,
    DropdownItem,
    Pagination,
    Toolbar,
    ToolbarContent,
    ToolbarItem,
} from '@patternfly/react-core';

import RequestCommentsButton from 'Containers/VulnMgmt/RiskAcceptance/RequestComments/RequestCommentsButton';
import BulkActionsDropdown from 'Components/PatternFly/BulkActionsDropdown';
import useTableSelection from 'hooks/useTableSelection';
import { UsePaginationResult } from 'hooks/patternfly/usePagination';
import usePermissions from 'hooks/usePermissions';
import { VulnerabilityRequest } from '../vulnerabilityRequests.graphql';
import { ApprovedFalsePositiveRequestsToBeAssessed } from './types';
import useRiskAcceptance from '../useRiskAcceptance';
import VulnerabilityRequestScope from '../PendingApprovals/VulnerabilityRequestScope';
import UndoVulnRequestModal from '../UndoVulnRequestModal';
import ApprovedFalsePositiveActionsColumn from './ApprovedFalsePositiveActionsColumn';
import ImpactedEntities from '../ImpactedEntities';
import VulnRequestedAction from '../VulnRequestedAction';

export type ApprovedFalsePositivesTableProps = {
    rows: VulnerabilityRequest[];
    updateTable: () => void;
    isLoading: boolean;
    itemCount: number;
} & UsePaginationResult;

function ApprovedFalsePositivesTable({
    rows,
    updateTable,
    itemCount,
    page,
    perPage,
    onSetPage,
    onPerPageSelect,
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
        requestIDs: requestsToBeAssessed?.requestIDs || [],
    });
    const { currentUserName, hasReadWriteAccess } = usePermissions();

    function cancelAssessment() {
        setRequestsToBeAssessed(null);
    }

    async function completeAssessment() {
        onClearAll();
        setRequestsToBeAssessed(null);
        updateTable();
    }

    const canApproveRequests = hasReadWriteAccess('VulnerabilityManagementApprovals');
    const canCreateRequests = hasReadWriteAccess('VulnerabilityManagementRequests');

    const selectedIds = getSelectedIds();
    const selectedDeferralsToReobserve = rows
        .filter((row) => {
            return (
                selectedIds.includes(row.id) &&
                (canApproveRequests ||
                    (canCreateRequests && row.requestor.name === currentUserName))
            );
        })
        .map((row) => row.id);

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
                                        requestIDs: selectedDeferralsToReobserve,
                                    })
                                }
                                isDisabled={selectedDeferralsToReobserve.length === 0}
                            >
                                Reobserve CVEs ({selectedDeferralsToReobserve.length})
                            </DropdownItem>
                        </BulkActionsDropdown>
                    </ToolbarItem>
                    <ToolbarItem variant="pagination" alignment={{ default: 'alignRight' }}>
                        <Pagination
                            itemCount={itemCount}
                            page={page}
                            onSetPage={onSetPage}
                            perPage={perPage}
                            onPerPageSelect={onPerPageSelect}
                        />
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
                        const canReobserveCVE =
                            canApproveRequests ||
                            (canCreateRequests && row.requestor.name === currentUserName);

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
                                    <VulnRequestedAction
                                        targetState={row.targetState}
                                        requestStatus={row.status}
                                        deferralReq={row.deferralReq}
                                        updatedDeferralReq={row.updatedDeferralReq}
                                        currentDate={new Date()}
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
                                        canReobserveCVE={canReobserveCVE}
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
                numRequestsToBeAssessed={requestsToBeAssessed?.requestIDs.length || 0}
                onSendRequest={undoVulnRequests}
                onCompleteRequest={completeAssessment}
                onCancel={cancelAssessment}
            />
        </>
    );
}

export default ApprovedFalsePositivesTable;
