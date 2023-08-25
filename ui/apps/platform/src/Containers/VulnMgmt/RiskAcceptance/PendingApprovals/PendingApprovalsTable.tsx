import React, { ReactElement, useState } from 'react';
import { TableComposable, Thead, Tbody, Tr, Th, Td } from '@patternfly/react-table';
import {
    Bullseye,
    Divider,
    DropdownItem,
    Pagination,
    Spinner,
    Toolbar,
    ToolbarContent,
    ToolbarItem,
} from '@patternfly/react-core';
import { SearchIcon } from '@patternfly/react-icons';

import RequestCommentsButton from 'Containers/VulnMgmt/RiskAcceptance/RequestComments/RequestCommentsButton';
import BulkActionsDropdown from 'Components/PatternFly/BulkActionsDropdown';
import EmptyStateTemplate from 'Components/PatternFly/EmptyStateTemplate';
import useTableSelection from 'hooks/useTableSelection';
import { UsePaginationResult } from 'hooks/patternfly/usePagination';
import usePermissions from 'hooks/usePermissions';
import { SearchFilter } from 'types/search';
import useAuthStatus from 'hooks/useAuthStatus';
import { VulnerabilityRequest } from '../vulnerabilityRequests.graphql';
import VulnRequestedAction from '../VulnRequestedAction';
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
import DeferralExpirationDate from '../DeferralExpirationDate';
import ImpactedEntities from '../ImpactedEntities/ImpactedEntities';
import PendingApprovalsSearchFilter from './PendingApprovalsSearchFilter';
import SearchFilterResults from '../SearchFilterResults';

export type PendingApprovalsTableProps = {
    rows: VulnerabilityRequest[];
    updateTable: () => void;
    isLoading: boolean;
    itemCount: number;
    searchFilter: SearchFilter;
    setSearchFilter: (newFilter: SearchFilter) => void;
} & UsePaginationResult;

function PendingApprovalsTable({
    rows,
    updateTable,
    isLoading,
    itemCount,
    searchFilter,
    setSearchFilter,
    page,
    perPage,
    onSetPage,
    onPerPageSelect,
}: PendingApprovalsTableProps): ReactElement {
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
    const requestIDs = requestsToBeAssessed?.requests.map((request) => request.id) || [];
    const { approveVulnRequests, denyVulnRequests, deleteVulnRequests } = useRiskAcceptance({
        requestIDs,
    });
    const { hasReadWriteAccess } = usePermissions();
    const { currentUser } = useAuthStatus();

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
    const selectedDeferralsToApproveDeny = rows.filter((row) => {
        return canApproveRequests && row.targetState === 'DEFERRED' && selectedIds.includes(row.id);
    });
    const selectedFalsePositivesToApproveDeny = rows.filter((row) => {
        return (
            canApproveRequests &&
            row.targetState === 'FALSE_POSITIVE' &&
            selectedIds.includes(row.id)
        );
    });
    const selectedDeferralsToCancel = rows.filter((row) => {
        return (
            (canApproveRequests ||
                (canCreateRequests && row.requestor.id === currentUser.userId)) &&
            row.targetState === 'DEFERRED' &&
            selectedIds.includes(row.id)
        );
    });
    const selectedFalsePositivesToCancel = rows.filter((row) => {
        return (
            (canApproveRequests ||
                (canCreateRequests && row.requestor.id === currentUser.userId)) &&
            row.targetState === 'FALSE_POSITIVE' &&
            selectedIds.includes(row.id)
        );
    });

    return (
        <>
            <Toolbar>
                <ToolbarContent>
                    <ToolbarItem>
                        <PendingApprovalsSearchFilter
                            searchFilter={searchFilter}
                            setSearchFilter={setSearchFilter}
                        />
                    </ToolbarItem>
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
                                        requests: selectedDeferralsToApproveDeny,
                                    })
                                }
                                isDisabled={selectedDeferralsToApproveDeny.length === 0}
                            >
                                Approve deferrals ({selectedDeferralsToApproveDeny.length})
                            </DropdownItem>
                            <DropdownItem
                                key="approve false positives"
                                component="button"
                                onClick={() =>
                                    setRequestsToBeAssessed({
                                        type: 'FALSE_POSITIVE',
                                        action: 'APPROVE',
                                        requests: selectedFalsePositivesToApproveDeny,
                                    })
                                }
                                isDisabled={selectedFalsePositivesToApproveDeny.length === 0}
                            >
                                Approve false positives (
                                {selectedFalsePositivesToApproveDeny.length})
                            </DropdownItem>
                            <DropdownItem
                                key="deny deferrals"
                                component="button"
                                onClick={() =>
                                    setRequestsToBeAssessed({
                                        type: 'DEFERRAL',
                                        action: 'DENY',
                                        requests: selectedDeferralsToApproveDeny,
                                    })
                                }
                                isDisabled={selectedDeferralsToApproveDeny.length === 0}
                            >
                                Deny deferrals ({selectedDeferralsToApproveDeny.length})
                            </DropdownItem>
                            <DropdownItem
                                key="deny false positives"
                                component="button"
                                onClick={() =>
                                    setRequestsToBeAssessed({
                                        type: 'FALSE_POSITIVE',
                                        action: 'DENY',
                                        requests: selectedFalsePositivesToApproveDeny,
                                    })
                                }
                                isDisabled={selectedFalsePositivesToApproveDeny.length === 0}
                            >
                                Deny false positives ({selectedFalsePositivesToApproveDeny.length})
                            </DropdownItem>
                            <DropdownItem
                                key="cancel deferrals"
                                component="button"
                                onClick={() =>
                                    setRequestsToBeAssessed({
                                        type: 'DEFERRAL',
                                        action: 'CANCEL',
                                        requests: selectedDeferralsToCancel,
                                    })
                                }
                                isDisabled={selectedDeferralsToCancel.length === 0}
                            >
                                Cancel deferrals ({selectedDeferralsToCancel.length})
                            </DropdownItem>
                            <DropdownItem
                                key="cancel false positives"
                                component="button"
                                onClick={() =>
                                    setRequestsToBeAssessed({
                                        type: 'FALSE_POSITIVE',
                                        action: 'CANCEL',
                                        requests: selectedFalsePositivesToCancel,
                                    })
                                }
                                isDisabled={selectedFalsePositivesToCancel.length === 0}
                            >
                                Cancel false positives ({selectedFalsePositivesToCancel.length})
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
            {Object.keys(searchFilter).length !== 0 && (
                <Toolbar>
                    <ToolbarContent>
                        <ToolbarItem>
                            <SearchFilterResults
                                searchFilter={searchFilter}
                                setSearchFilter={setSearchFilter}
                            />
                        </ToolbarItem>
                    </ToolbarContent>
                </Toolbar>
            )}
            <Divider component="div" />
            {isLoading ? (
                <Bullseye>
                    <Spinner isSVG size="xl" />
                </Bullseye>
            ) : (
                <TableComposable aria-label="Pending Approvals Table" variant="compact" borders>
                    <Thead>
                        <Tr>
                            <Th
                                select={{
                                    onSelect: onSelectAll,
                                    isSelected: allRowsSelected,
                                }}
                            />
                            <Th>Requested entity</Th>
                            <Th>Requested action</Th>
                            <Th>Expires</Th>
                            <Th modifier="fitContent">Scope</Th>
                            <Th>Impacted entities</Th>
                            <Th>Comments</Th>
                            <Th>Requestor</Th>
                        </Tr>
                    </Thead>
                    <Tbody>
                        {rows.map((row, rowIndex) => {
                            const canCancelRequest =
                                canApproveRequests ||
                                (canCreateRequests && row.requestor.id === currentUser.userId);

                            return (
                                <Tr key={row.id}>
                                    <Td
                                        select={{
                                            rowIndex,
                                            onSelect,
                                            isSelected: selected[rowIndex],
                                        }}
                                    />
                                    <Td dataLabel="Requested entity">{row.cves.cves[0]}</Td>
                                    <Td dataLabel="Requested action">
                                        <VulnRequestedAction
                                            targetState={row.targetState}
                                            requestStatus={row.status}
                                            deferralReq={row.deferralReq}
                                            updatedDeferralReq={row.updatedDeferralReq}
                                            currentDate={new Date()}
                                        />
                                    </Td>
                                    <Td dataLabel="Expires">
                                        <DeferralExpirationDate
                                            targetState={row.targetState}
                                            requestStatus={row.status}
                                            deferralReq={row.deferralReq}
                                            updatedDeferralReq={row.updatedDeferralReq}
                                        />
                                    </Td>
                                    <Td dataLabel="Scope">
                                        <VulnerabilityRequestScope scope={row.scope} />
                                    </Td>
                                    <Td dataLabel="Impacted entities">
                                        <ImpactedEntities
                                            deployments={row.deployments}
                                            deploymentCount={row.deploymentCount}
                                            images={row.images}
                                            imageCount={row.imageCount}
                                        />
                                    </Td>
                                    <Td dataLabel="Comments">
                                        <RequestCommentsButton
                                            comments={row.comments}
                                            cve={row.cves.cves[0]}
                                        />
                                    </Td>
                                    <Td dataLabel="Requestor">{row.requestor.name}</Td>
                                    <Td className="pf-u-text-align-right">
                                        {row.targetState === 'DEFERRED' && (
                                            <DeferralRequestActionsColumn
                                                row={row}
                                                setRequestsToBeAssessed={setRequestsToBeAssessed}
                                                canApproveRequest={canApproveRequests}
                                                canCancelRequest={canCancelRequest}
                                            />
                                        )}
                                        {row.targetState === 'FALSE_POSITIVE' && (
                                            <FalsePositiveRequestActionsColumn
                                                row={row}
                                                setRequestsToBeAssessed={setRequestsToBeAssessed}
                                                canApproveRequest={canApproveRequests}
                                                canCancelRequest={canCancelRequest}
                                            />
                                        )}
                                    </Td>
                                </Tr>
                            );
                        })}
                        {!rows.length && (
                            <Tr>
                                <Td colSpan={8}>
                                    <Bullseye>
                                        <EmptyStateTemplate
                                            title="No pending approvals found"
                                            headingLevel="h2"
                                            icon={SearchIcon}
                                        >
                                            To continue, edit your filter settings and search again.
                                        </EmptyStateTemplate>
                                    </Bullseye>
                                </Td>
                            </Tr>
                        )}
                    </Tbody>
                </TableComposable>
            )}

            {/* @TODO: The modals are very similiar and probably could be abstracted out more */}
            <ApproveDeferralModal
                isOpen={
                    requestsToBeAssessed?.type === 'DEFERRAL' &&
                    requestsToBeAssessed.action === 'APPROVE'
                }
                vulnerabilityRequests={requestsToBeAssessed?.requests || []}
                onSendRequest={approveVulnRequests}
                onCompleteRequest={completeAssessment}
                onCancel={cancelAssessment}
            />
            <ApproveFalsePositiveModal
                isOpen={
                    requestsToBeAssessed?.type === 'FALSE_POSITIVE' &&
                    requestsToBeAssessed.action === 'APPROVE'
                }
                vulnerabilityRequests={requestsToBeAssessed?.requests || []}
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
