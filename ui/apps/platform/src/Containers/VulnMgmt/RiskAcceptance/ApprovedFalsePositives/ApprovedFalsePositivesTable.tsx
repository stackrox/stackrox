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
import { ApprovedFalsePositiveRequestsToBeAssessed } from './types';
import useRiskAcceptance from '../useRiskAcceptance';
import VulnerabilityRequestScope from '../PendingApprovals/VulnerabilityRequestScope';
import UndoVulnRequestModal from '../UndoVulnRequestModal';
import ApprovedFalsePositiveActionsColumn from './ApprovedFalsePositiveActionsColumn';
import ImpactedEntities from '../ImpactedEntities/ImpactedEntities';
import VulnRequestedAction from '../VulnRequestedAction';
import ApprovedFalsePositivesSearchFilter from './ApprovedFalsePositivesSearchFilter';
import SearchFilterResults from '../SearchFilterResults';

export type ApprovedFalsePositivesTableProps = {
    rows: VulnerabilityRequest[];
    updateTable: () => void;
    isLoading: boolean;
    itemCount: number;
    searchFilter: SearchFilter;
    setSearchFilter: (newFilter: SearchFilter) => void;
} & UsePaginationResult;

function ApprovedFalsePositivesTable({
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
    const selectedDeferralsToReobserve = rows
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
                        <ApprovedFalsePositivesSearchFilter
                            searchFilter={searchFilter}
                            setSearchFilter={setSearchFilter}
                        />
                    </ToolbarItem>
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
                <TableComposable
                    aria-label="Approved False Positives Table"
                    variant="compact"
                    borders
                >
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
                                        <ApprovedFalsePositiveActionsColumn
                                            row={row}
                                            setRequestsToBeAssessed={setRequestsToBeAssessed}
                                            canReobserveCVE={canReobserveCVE}
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
                                            title="No approved false positives found"
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
