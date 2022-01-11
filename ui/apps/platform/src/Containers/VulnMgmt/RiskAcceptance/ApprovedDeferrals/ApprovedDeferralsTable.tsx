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
import usePermissions from 'hooks/patternfly/usePermissions';
import { VulnerabilityRequest } from '../vulnerabilityRequests.graphql';
import { ApprovedDeferralRequestsToBeAssessed } from './types';
import useRiskAcceptance from '../useRiskAcceptance';
import VulnerabilityRequestScope from '../PendingApprovals/VulnerabilityRequestScope';
import UndoVulnRequestModal from '../UndoVulnRequestModal';
import UpdateDeferralModal from './UpdateDeferralModal';
import ApprovedDeferralActionsColumn from './ApprovedDeferralActionsColumn';
import ImpactedEntities from '../ImpactedEntities';
import VulnRequestedAction from '../VulnRequestedAction';
import DeferralExpirationDate from '../DeferralExpirationDate';

export type ApprovedDeferralsTableProps = {
    rows: VulnerabilityRequest[];
    updateTable: () => void;
    isLoading: boolean;
    itemCount: number;
} & UsePaginationResult;

function ApprovedDeferralsTable({
    rows,
    updateTable,
    itemCount,
    page,
    perPage,
    onSetPage,
    onPerPageSelect,
}: ApprovedDeferralsTableProps): ReactElement {
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
        useState<ApprovedDeferralRequestsToBeAssessed>(null);
    const { updateVulnRequests, undoVulnRequests } = useRiskAcceptance({
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
    const selectedDeferralsToUpdate = rows
        .filter((row) => {
            return (
                selectedIds.includes(row.id) &&
                canCreateRequests &&
                row.requestor.name === currentUserName
            );
        })
        .map((row) => row.id);
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
                                key="update deferrals"
                                component="button"
                                onClick={() =>
                                    setRequestsToBeAssessed({
                                        type: 'DEFERRAL',
                                        action: 'UPDATE',
                                        requestIDs: selectedDeferralsToUpdate,
                                    })
                                }
                                isDisabled={selectedDeferralsToUpdate.length === 0}
                            >
                                Update deferrals ({selectedDeferralsToUpdate.length})
                            </DropdownItem>
                            <DropdownItem
                                key="undo deferrals"
                                component="button"
                                onClick={() =>
                                    setRequestsToBeAssessed({
                                        type: 'DEFERRAL',
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
            <TableComposable aria-label="Approved Deferrals Table" variant="compact" borders>
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
                        <Th>Expires</Th>
                        <Th>Scope</Th>
                        <Th>Impacted Entities</Th>
                        <Th>Apply to</Th>
                        <Th>Comments</Th>
                        <Th>Requestor</Th>
                    </Tr>
                </Thead>
                <Tbody>
                    {rows.map((row, rowIndex) => {
                        const canUpdateDeferral =
                            canCreateRequests && row.requestor.name === currentUserName;
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
                                        currentDate={new Date()}
                                    />
                                </Td>
                                <Td dataLabel="Expires">
                                    <DeferralExpirationDate
                                        targetState={row.targetState}
                                        requestStatus={row.status}
                                        deferralReq={row.deferralReq}
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
                                    <ApprovedDeferralActionsColumn
                                        row={row}
                                        setRequestsToBeAssessed={setRequestsToBeAssessed}
                                        canReobserveCVE={canReobserveCVE}
                                        canUpdateDeferral={canUpdateDeferral}
                                    />
                                </Td>
                            </Tr>
                        );
                    })}
                </Tbody>
            </TableComposable>
            <UndoVulnRequestModal
                type="DEFERRAL"
                isOpen={requestsToBeAssessed?.action === 'UNDO'}
                numRequestsToBeAssessed={requestsToBeAssessed?.requestIDs.length || 0}
                onSendRequest={undoVulnRequests}
                onCompleteRequest={completeAssessment}
                onCancel={cancelAssessment}
            />
            <UpdateDeferralModal
                isOpen={requestsToBeAssessed?.action === 'UPDATE'}
                numRequestsToBeAssessed={requestsToBeAssessed?.requestIDs.length || 0}
                onSendRequest={updateVulnRequests}
                onCompleteRequest={completeAssessment}
                onCancel={cancelAssessment}
            />
        </>
    );
}

export default ApprovedDeferralsTable;
