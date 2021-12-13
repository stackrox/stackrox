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
};

function ApprovedDeferralsTable({ rows, updateTable }: ApprovedDeferralsTableProps): ReactElement {
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
    const selectedDeferralRequests = rows.filter((row) => selectedIds.includes(row.id));

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
                                        requests: selectedDeferralRequests,
                                    })
                                }
                                isDisabled={selectedDeferralRequests.length === 0}
                            >
                                Update deferrals ({selectedDeferralRequests.length})
                            </DropdownItem>
                            <DropdownItem
                                key="undo deferrals"
                                component="button"
                                onClick={() =>
                                    setRequestsToBeAssessed({
                                        type: 'DEFERRAL',
                                        action: 'UNDO',
                                        requests: selectedDeferralRequests,
                                    })
                                }
                                isDisabled={selectedDeferralRequests.length === 0}
                            >
                                Reobserve CVEs ({selectedDeferralRequests.length})
                            </DropdownItem>
                        </BulkActionsDropdown>
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
                numRequestsToBeAssessed={requestsToBeAssessed?.requests.length || 0}
                onSendRequest={undoVulnRequests}
                onCompleteRequest={completeAssessment}
                onCancel={cancelAssessment}
            />
            <UpdateDeferralModal
                isOpen={requestsToBeAssessed?.action === 'UPDATE'}
                numRequestsToBeAssessed={requestsToBeAssessed?.requests.length || 0}
                onSendRequest={updateVulnRequests}
                onCompleteRequest={completeAssessment}
                onCancel={cancelAssessment}
            />
        </>
    );
}

export default ApprovedDeferralsTable;
