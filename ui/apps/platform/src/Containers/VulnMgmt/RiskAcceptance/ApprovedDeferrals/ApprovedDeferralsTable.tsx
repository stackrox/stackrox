import React, { ReactElement, useState } from 'react';
import { TableComposable, Thead, Tbody, Tr, Th, Td } from '@patternfly/react-table';
import {
    Divider,
    DropdownItem,
    Pagination,
    Toolbar,
    ToolbarContent,
    ToolbarItem,
    Bullseye,
    Spinner,
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
import { ApprovedDeferralRequestsToBeAssessed } from './types';
import useRiskAcceptance from '../useRiskAcceptance';
import VulnerabilityRequestScope from '../PendingApprovals/VulnerabilityRequestScope';
import UndoVulnRequestModal from '../UndoVulnRequestModal';
import UpdateDeferralModal from './UpdateDeferralModal';
import ApprovedDeferralActionsColumn from './ApprovedDeferralActionsColumn';
import ImpactedEntities from '../ImpactedEntities/ImpactedEntities';
import VulnRequestedAction from '../VulnRequestedAction';
import DeferralExpirationDate from '../DeferralExpirationDate';
import ApprovedDeferralsSearchFilter from './ApprovedDeferralsSearchFilter';
import SearchFilterResults from '../SearchFilterResults';

export type ApprovedDeferralsTableProps = {
    rows: VulnerabilityRequest[];
    updateTable: () => void;
    isLoading: boolean;
    itemCount: number;
    searchFilter: SearchFilter;
    setSearchFilter: (newFilter: SearchFilter) => void;
} & UsePaginationResult;

function ApprovedDeferralsTable({
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
    const selectedDeferralsAssess = rows
        .filter((row) => {
            return (
                selectedIds.includes(row.id) &&
                (canApproveRequests ||
                    (canCreateRequests && row.requestor.id === currentUser.userId))
            );
        })
        .map((row) => row.id);

    return (
        <>
            <Toolbar>
                <ToolbarContent>
                    <ToolbarItem>
                        <ApprovedDeferralsSearchFilter
                            searchFilter={searchFilter}
                            setSearchFilter={setSearchFilter}
                        />
                    </ToolbarItem>
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
                                        requestIDs: selectedDeferralsAssess,
                                    })
                                }
                                isDisabled={selectedDeferralsAssess.length === 0}
                            >
                                Update deferrals ({selectedDeferralsAssess.length})
                            </DropdownItem>
                            <DropdownItem
                                key="undo deferrals"
                                component="button"
                                onClick={() =>
                                    setRequestsToBeAssessed({
                                        type: 'DEFERRAL',
                                        action: 'UNDO',
                                        requestIDs: selectedDeferralsAssess,
                                    })
                                }
                                isDisabled={selectedDeferralsAssess.length === 0}
                            >
                                Reobserve CVEs ({selectedDeferralsAssess.length})
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
                <TableComposable aria-label="Approved Deferrals Table" variant="compact" borders>
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
                            const canReobserveCVE =
                                canApproveRequests ||
                                (canCreateRequests && row.requestor.id === currentUser.userId);
                            const canUpdateDeferral = canReobserveCVE;

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
                        {!rows.length && (
                            <Tr>
                                <Td colSpan={8}>
                                    <Bullseye>
                                        <EmptyStateTemplate
                                            title="No approved deferrals found"
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
