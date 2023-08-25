import React, { ReactElement, useState } from 'react';
import { TableComposable, Thead, Tbody, Tr, Th, Td } from '@patternfly/react-table';
import {
    Bullseye,
    Button,
    ButtonVariant,
    Divider,
    DropdownItem,
    PageSection,
    PageSectionVariants,
    Pagination,
    Spinner,
    Toolbar,
    ToolbarContent,
    ToolbarItem,
} from '@patternfly/react-core';

import useTableSelection from 'hooks/useTableSelection';
import BulkActionsDropdown from 'Components/PatternFly/BulkActionsDropdown';
import VulnerabilitySeverityIconText from 'Components/PatternFly/IconText/VulnerabilitySeverityIconText';
import { UsePaginationResult } from 'hooks/patternfly/usePagination';
import usePermissions from 'hooks/usePermissions';
import useAuthStatus from 'hooks/useAuthStatus';
import { SearchFilter } from 'types/search';
import EmptyStateTemplate from 'Components/PatternFly/EmptyStateTemplate';
import { GetSortParams } from 'hooks/patternfly/useTableSort';
import { Vulnerability, EmbeddedImageScanComponent } from '../imageVulnerabilities.graphql';
import { DeferredCVEsToBeAssessed } from './types';
import DeferredCVEActionsColumn from './DeferredCVEActionsColumn';
import useRiskAcceptance from '../useRiskAcceptance';
import UndoVulnRequestModal from '../UndoVulnRequestModal';
import RequestCommentsButton from '../RequestComments/RequestCommentsButton';
import DeferralExpirationDate from '../DeferralExpirationDate';
import VulnerabilityRequestScope from '../PendingApprovals/VulnerabilityRequestScope';
import CVESummaryLink from '../CVESummaryLink';
import SearchFilterResults from '../SearchFilterResults';
import ImageVulnsSearchFilter from '../ImageVulnsSearchFilter';

export type DeferredCVEsTableProps = {
    rows: Vulnerability[];
    isLoading: boolean;
    itemCount: number;
    updateTable: () => void;
    searchFilter: SearchFilter;
    setSearchFilter: React.Dispatch<React.SetStateAction<SearchFilter>>;
    getSortParams: GetSortParams;
    showComponentDetails: (components: EmbeddedImageScanComponent[], cveName: string) => void;
} & UsePaginationResult;

function DeferredCVEsTable({
    rows,
    itemCount,
    page,
    perPage,
    onSetPage,
    onPerPageSelect,
    updateTable,
    searchFilter,
    setSearchFilter,
    isLoading,
    getSortParams,
    showComponentDetails,
}: DeferredCVEsTableProps): ReactElement {
    const {
        selected,
        allRowsSelected,
        numSelected,
        onSelect,
        onSelectAll,
        onClearAll,
        getSelectedIds,
    } = useTableSelection<Vulnerability>(rows);
    const [vulnsToBeAssessed, setVulnsToBeAssessed] = useState<DeferredCVEsToBeAssessed>(null);
    const { undoVulnRequests } = useRiskAcceptance({
        requestIDs: vulnsToBeAssessed?.requestIDs || [],
    });
    const { hasReadWriteAccess } = usePermissions();
    const { currentUser } = useAuthStatus();

    function cancelAssessment() {
        setVulnsToBeAssessed(null);
    }

    async function completeAssessment() {
        onClearAll();
        setVulnsToBeAssessed(null);
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
                    (canCreateRequests &&
                        row.vulnerabilityRequest.requestor.id === currentUser.userId))
            );
        })
        .map((row) => {
            return row.vulnerabilityRequest.id;
        });

    return (
        <>
            <Toolbar id="toolbar">
                <ToolbarContent>
                    <ToolbarItem>
                        <ImageVulnsSearchFilter
                            searchFilter={searchFilter}
                            setSearchFilter={setSearchFilter}
                        />
                    </ToolbarItem>
                    <ToolbarItem variant="separator" />
                    <ToolbarItem>
                        <BulkActionsDropdown isDisabled={numSelected === 0}>
                            <DropdownItem
                                key="undo deferrals"
                                component="button"
                                onClick={() =>
                                    setVulnsToBeAssessed({
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
            <TableComposable aria-label="Observed CVEs Table" variant="compact" borders>
                <Thead>
                    <Tr>
                        <Th
                            select={{
                                onSelect: onSelectAll,
                                isSelected: allRowsSelected,
                            }}
                        />
                        <Th>CVE</Th>
                        <Th>Fixable</Th>
                        <Th sort={getSortParams('Severity')}>Severity</Th>
                        <Th>Expires</Th>
                        <Th modifier="fitContent">Scope</Th>
                        <Th>Affected Components</Th>
                        <Th>Comments</Th>
                        <Th>Approver</Th>
                    </Tr>
                </Thead>
                <Tbody>
                    {isLoading && (
                        <Tr>
                            <Td colSpan={9}>
                                <Bullseye>
                                    <Spinner isSVG size="sm" />
                                </Bullseye>
                            </Td>
                        </Tr>
                    )}
                    {!isLoading && rows && rows.length === 0 ? (
                        <Tr>
                            <Td colSpan={9}>
                                <PageSection variant={PageSectionVariants.light} isFilled>
                                    <EmptyStateTemplate
                                        title="No deferral requests were approved."
                                        headingLevel="h3"
                                    />
                                </PageSection>
                            </Td>
                        </Tr>
                    ) : (
                        rows.map((row, rowIndex) => {
                            const canReobserveCVE =
                                canApproveRequests ||
                                (canCreateRequests &&
                                    row.vulnerabilityRequest?.requestor.id === currentUser.userId);

                            return (
                                <Tr key={row.cve}>
                                    <Td
                                        select={{
                                            rowIndex,
                                            onSelect,
                                            isSelected: selected[rowIndex],
                                        }}
                                    />
                                    <Td dataLabel="CVE">
                                        <CVESummaryLink cve={row.cve} id={row.id} />
                                    </Td>
                                    <Td dataLabel="Fixable">{row.isFixable ? 'Yes' : 'No'}</Td>
                                    <Td dataLabel="Severity">
                                        <VulnerabilitySeverityIconText severity={row.severity} />
                                    </Td>
                                    <Td dataLabel="Expires">
                                        {row.vulnerabilityRequest ? (
                                            <DeferralExpirationDate
                                                targetState={row.vulnerabilityRequest.targetState}
                                                requestStatus={row.vulnerabilityRequest.status}
                                                deferralReq={row.vulnerabilityRequest.deferralReq}
                                            />
                                        ) : (
                                            'N/A'
                                        )}
                                    </Td>
                                    <Td dataLabel="Scope">
                                        {row.vulnerabilityRequest ? (
                                            <VulnerabilityRequestScope
                                                scope={row.vulnerabilityRequest.scope}
                                            />
                                        ) : (
                                            'N/A'
                                        )}
                                    </Td>
                                    <Td dataLabel="Affected components">
                                        <Button
                                            variant={ButtonVariant.link}
                                            isInline
                                            onClick={() => {
                                                showComponentDetails(row.components, row.cve);
                                            }}
                                        >
                                            {row.components.length} components
                                        </Button>
                                    </Td>
                                    <Td dataLabel="Comments">
                                        {row.vulnerabilityRequest ? (
                                            <RequestCommentsButton
                                                comments={row.vulnerabilityRequest.comments}
                                                cve={row.vulnerabilityRequest.cves.cves[0]}
                                            />
                                        ) : (
                                            'N/A'
                                        )}
                                    </Td>
                                    <Td dataLabel="Approver">
                                        {row.vulnerabilityRequest
                                            ? row.vulnerabilityRequest.approvers
                                                  .map((user) => user.name)
                                                  .join(',')
                                            : 'N/A'}
                                    </Td>
                                    <Td className="pf-u-text-align-right">
                                        <DeferredCVEActionsColumn
                                            row={row}
                                            setVulnsToBeAssessed={setVulnsToBeAssessed}
                                            canReobserveCVE={canReobserveCVE}
                                        />
                                    </Td>
                                </Tr>
                            );
                        })
                    )}
                </Tbody>
            </TableComposable>
            <UndoVulnRequestModal
                type="DEFERRAL"
                isOpen={vulnsToBeAssessed?.action === 'UNDO'}
                numRequestsToBeAssessed={vulnsToBeAssessed?.requestIDs.length || 0}
                onSendRequest={undoVulnRequests}
                onCompleteRequest={completeAssessment}
                onCancel={cancelAssessment}
            />
        </>
    );
}

export default DeferredCVEsTable;
