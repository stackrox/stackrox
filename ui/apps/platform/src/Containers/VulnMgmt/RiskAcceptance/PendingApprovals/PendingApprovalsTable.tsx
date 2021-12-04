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
import RequestedAction from './RequestedAction';
import VulnerabilityRequestScope from './VulnerabilityRequestScope';
import ApproveDeferralModal from './ApproveDeferralModal';
import useRiskAcceptance from '../useRiskAcceptance';
import DeferralRequestActionsColumn from './DeferralRequestActionsColumn';
import FalsePositiveRequestActionsColumn from './FalsePositiveRequestActionsColumn';
import { RequestsToBeAssessed } from './types';
import ApproveFalsePositiveModal from './ApproveFalsePositiveModal';
import DenyDeferralModal from './DenyDeferralModal';
import DenyFalsePositiveModal from './DenyFalsePositiveModal';
import CancelVulnRequestModal from './CancelVulnRequestModal';
import VulnRequestType from '../VulnRequestType';

export type PendingApprovalsTableProps = {
    rows: VulnerabilityRequest[];
    updateTable: () => void;
    isLoading: boolean;
};

function PendingApprovalsTable({ rows, updateTable }: PendingApprovalsTableProps): ReactElement {
    const {
        selected,
        allRowsSelected,
        numSelected,
        onSelect,
        onSelectAll,
        getSelectedIds,
        onClearAll,
    } = useTableSelection<VulnerabilityRequest>(rows);
    const [requestsToBeAssessed, setRequestsToBeAssessed] = useState<RequestsToBeAssessed>(null);
    const { approveVulnRequests, denyVulnRequests, deleteVulnRequests } = useRiskAcceptance({
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
    const selectedDeferralRequests = rows.filter(
        (row) => selectedIds.includes(row.id) && row.targetState === 'DEFERRED'
    );
    const selectedFalsePositiveRequests = rows.filter(
        (row) => selectedIds.includes(row.id) && row.targetState === 'FALSE_POSITIVE'
    );

    return (
        <>
            <Toolbar>
                <ToolbarContent>
                    <ToolbarItem variant="separator" />
                    <ToolbarItem>
                        <BulkActionsDropdown isDisabled={numSelected === 0}>
                            <DropdownItem
                                key="approve deferrals"
                                component="button"
                                onClick={() =>
                                    setRequestsToBeAssessed({
                                        type: 'DEFERRAL',
                                        action: 'APPROVE',
                                        requests: selectedDeferralRequests,
                                    })
                                }
                                isDisabled={selectedDeferralRequests.length === 0}
                            >
                                Approve deferrals ({selectedDeferralRequests.length})
                            </DropdownItem>
                            <DropdownItem
                                key="deny deferrals"
                                component="button"
                                onClick={() =>
                                    setRequestsToBeAssessed({
                                        type: 'DEFERRAL',
                                        action: 'DENY',
                                        requests: selectedDeferralRequests,
                                    })
                                }
                                isDisabled={selectedDeferralRequests.length === 0}
                            >
                                Deny deferrals ({selectedDeferralRequests.length})
                            </DropdownItem>
                            <DropdownItem
                                key="cancel deferrals"
                                component="button"
                                onClick={() =>
                                    setRequestsToBeAssessed({
                                        type: 'DEFERRAL',
                                        action: 'CANCEL',
                                        requests: selectedDeferralRequests,
                                    })
                                }
                                isDisabled={selectedDeferralRequests.length === 0}
                            >
                                Cancel deferrals ({selectedDeferralRequests.length})
                            </DropdownItem>
                            <DropdownItem
                                key="approve false positives"
                                component="button"
                                onClick={() =>
                                    setRequestsToBeAssessed({
                                        type: 'FALSE_POSITIVE',
                                        action: 'APPROVE',
                                        requests: selectedFalsePositiveRequests,
                                    })
                                }
                                isDisabled={selectedFalsePositiveRequests.length === 0}
                            >
                                Approve false positives ({selectedFalsePositiveRequests.length})
                            </DropdownItem>
                            <DropdownItem
                                key="deny false positives"
                                component="button"
                                onClick={() =>
                                    setRequestsToBeAssessed({
                                        type: 'FALSE_POSITIVE',
                                        action: 'DENY',
                                        requests: selectedFalsePositiveRequests,
                                    })
                                }
                                isDisabled={selectedFalsePositiveRequests.length === 0}
                            >
                                Deny false positives ({selectedFalsePositiveRequests.length})
                            </DropdownItem>
                            <DropdownItem
                                key="cancel false positives"
                                component="button"
                                onClick={() =>
                                    setRequestsToBeAssessed({
                                        type: 'FALSE_POSITIVE',
                                        action: 'CANCEL',
                                        requests: selectedFalsePositiveRequests,
                                    })
                                }
                                isDisabled={selectedFalsePositiveRequests.length === 0}
                            >
                                Cancel false positives ({selectedFalsePositiveRequests.length})
                            </DropdownItem>
                        </BulkActionsDropdown>
                    </ToolbarItem>
                </ToolbarContent>
            </Toolbar>
            <Divider component="div" />
            <TableComposable aria-label="Pending Approvals Table" variant="compact" borders>
                <Thead>
                    <Tr>
                        <Th
                            select={{
                                onSelect: onSelectAll,
                                isSelected: allRowsSelected,
                            }}
                        />
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
                                <Td dataLabel="Type">
                                    <VulnRequestType
                                        targetState={row.targetState}
                                        requestStatus={row.status}
                                    />
                                </Td>
                                <Td dataLabel="Scope">
                                    {row.scope.imageScope ? 'image' : 'global'}
                                </Td>
                                <Td dataLabel="Impacted entities">-</Td>
                                <Td dataLabel="Requested Action">
                                    <RequestedAction
                                        targetState={row.targetState}
                                        requestStatus={row.status}
                                        deferralReq={row.deferralReq}
                                        updatedDeferralReq={row.updatedDeferralReq}
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
                                    {row.targetState === 'DEFERRED' && (
                                        <DeferralRequestActionsColumn
                                            row={row}
                                            setRequestsToBeAssessed={setRequestsToBeAssessed}
                                        />
                                    )}
                                    {row.targetState === 'FALSE_POSITIVE' && (
                                        <FalsePositiveRequestActionsColumn
                                            row={row}
                                            setRequestsToBeAssessed={setRequestsToBeAssessed}
                                        />
                                    )}
                                </Td>
                            </Tr>
                        );
                    })}
                </Tbody>
            </TableComposable>
            {/* @TODO: The modals are very similiar and probably could be abstracted out more */}
            <ApproveDeferralModal
                isOpen={
                    requestsToBeAssessed?.type === 'DEFERRAL' &&
                    requestsToBeAssessed.action === 'APPROVE'
                }
                numRequestsToBeAssessed={requestsToBeAssessed?.requests.length || 0}
                numImpactedDeployments={0} // Add this when the data is available from backend
                numImpactedImages={0} // Add this when the data is available from backend
                onSendRequest={approveVulnRequests}
                onCompleteRequest={completeAssessment}
                onCancel={cancelAssessment}
            />
            <ApproveFalsePositiveModal
                isOpen={
                    requestsToBeAssessed?.type === 'FALSE_POSITIVE' &&
                    requestsToBeAssessed.action === 'APPROVE'
                }
                numRequestsToBeAssessed={requestsToBeAssessed?.requests.length || 0}
                numImpactedDeployments={0} // Add this when the data is available from backend
                numImpactedImages={0} // Add this when the data is available from backend
                onSendRequest={approveVulnRequests}
                onCompleteRequest={completeAssessment}
                onCancel={cancelAssessment}
            />
            <DenyDeferralModal
                isOpen={
                    requestsToBeAssessed?.type === 'DEFERRAL' &&
                    requestsToBeAssessed.action === 'DENY'
                }
                numRequestsToBeAssessed={requestsToBeAssessed?.requests.length || 0}
                onSendRequest={denyVulnRequests}
                onCompleteRequest={completeAssessment}
                onCancel={cancelAssessment}
            />
            <DenyFalsePositiveModal
                isOpen={
                    requestsToBeAssessed?.type === 'FALSE_POSITIVE' &&
                    requestsToBeAssessed.action === 'DENY'
                }
                numRequestsToBeAssessed={requestsToBeAssessed?.requests.length || 0}
                onSendRequest={denyVulnRequests}
                onCompleteRequest={completeAssessment}
                onCancel={cancelAssessment}
            />
            {requestsToBeAssessed?.action === 'CANCEL' && (
                <CancelVulnRequestModal
                    type={requestsToBeAssessed?.type}
                    numRequestsToBeAssessed={requestsToBeAssessed?.requests.length || 0}
                    onSendRequest={deleteVulnRequests}
                    onCompleteRequest={completeAssessment}
                    onCancel={cancelAssessment}
                />
            )}
        </>
    );
}

export default PendingApprovalsTable;
