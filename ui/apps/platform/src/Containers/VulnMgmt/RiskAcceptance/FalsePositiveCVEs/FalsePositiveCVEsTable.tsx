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

import VulnerabilitySeverityIconText from 'Components/PatternFly/IconText/VulnerabilitySeverityIconText';
import BulkActionsDropdown from 'Components/PatternFly/BulkActionsDropdown';
import useTableSelection from 'hooks/useTableSelection';
import { UsePaginationResult } from 'hooks/patternfly/usePagination';
import usePermissions from 'hooks/usePermissions';
import useAuthStatus from 'hooks/useAuthStatus';
import { SearchFilter } from 'types/search';
import EmptyStateTemplate from 'Components/PatternFly/EmptyStateTemplate';
import { GetSortParams } from 'hooks/patternfly/useTableSort';
import { Vulnerability, EmbeddedImageScanComponent } from '../imageVulnerabilities.graphql';
import { FalsePositiveCVEsToBeAssessed } from './types';
import useRiskAcceptance from '../useRiskAcceptance';
import UndoVulnRequestModal from '../UndoVulnRequestModal';
import FalsePositiveCVEActionsColumn from './FalsePositiveCVEActionsColumns';
import RequestCommentsButton from '../RequestComments/RequestCommentsButton';
import VulnerabilityRequestScope from '../PendingApprovals/VulnerabilityRequestScope';
import CVESummaryLink from '../CVESummaryLink';
import ImageVulnsSearchFilter from '../ImageVulnsSearchFilter';
import SearchFilterResults from '../SearchFilterResults';

export type FalsePositiveCVEsTableProps = {
    rows: Vulnerability[];
    isLoading: boolean;
    itemCount: number;
    updateTable: () => void;
    searchFilter: SearchFilter;
    setSearchFilter: React.Dispatch<React.SetStateAction<SearchFilter>>;
    getSortParams: GetSortParams;
    showComponentDetails: (components: EmbeddedImageScanComponent[], cveName: string) => void;
} & UsePaginationResult;

function FalsePositiveCVEsTable({
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
}: FalsePositiveCVEsTableProps): ReactElement {
    const {
        selected,
        allRowsSelected,
        numSelected,
        onSelect,
        onSelectAll,
        onClearAll,
        getSelectedIds,
    } = useTableSelection<Vulnerability>(rows);
    const [vulnsToBeAssessed, setVulnsToBeAssessed] = useState<FalsePositiveCVEsToBeAssessed>(null);
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
    const selectedFalsePositivesToReobserve = rows
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
                                key="undo false positives"
                                component="button"
                                onClick={() =>
                                    setVulnsToBeAssessed({
                                        type: 'FALSE_POSITIVE',
                                        action: 'UNDO',
                                        requestIDs: selectedFalsePositivesToReobserve,
                                    })
                                }
                                isDisabled={selectedFalsePositivesToReobserve.length === 0}
                            >
                                Reobserve CVEs ({selectedFalsePositivesToReobserve.length})
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
                        <Th modifier="fitContent">Scope</Th>
                        <Th>Affected Components</Th>
                        <Th>Comments</Th>
                        <Th>Approver</Th>
                    </Tr>
                </Thead>
                <Tbody>
                    {isLoading && (
                        // @TODO: Consider accessibility in this approach (https://github.com/stackrox/stackrox/pull/671#discussion_r811295184)
                        <Tr>
                            <Td colSpan={9}>
                                <Bullseye>
                                    <Spinner isSVG size="sm" />
                                </Bullseye>
                            </Td>
                        </Tr>
                    )}
                    {!isLoading && rows && rows.length === 0 ? (
                        // @TODO: Consider accessibility in this approach (https://github.com/stackrox/stackrox/pull/671#discussion_r811295477)
                        <Tr>
                            <Td colSpan={9}>
                                <PageSection variant={PageSectionVariants.light} isFilled>
                                    <EmptyStateTemplate
                                        title="No false positive requests were approved."
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
                                    row.vulnerabilityRequest.requestor.id === currentUser.userId);

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
                                    <Td dataLabel="Scope">
                                        <VulnerabilityRequestScope
                                            scope={row.vulnerabilityRequest.scope}
                                        />
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
                                        <RequestCommentsButton
                                            comments={row.vulnerabilityRequest.comments}
                                            cve={row.vulnerabilityRequest.cves.cves[0]}
                                        />
                                    </Td>
                                    <Td dataLabel="Approver">
                                        {row.vulnerabilityRequest.approvers
                                            .map((user) => user.name)
                                            .join(',')}
                                    </Td>
                                    <Td className="pf-u-text-align-right">
                                        <FalsePositiveCVEActionsColumn
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
                type="FALSE_POSITIVE"
                isOpen={vulnsToBeAssessed?.action === 'UNDO'}
                numRequestsToBeAssessed={vulnsToBeAssessed?.requestIDs.length || 0}
                onSendRequest={undoVulnRequests}
                onCompleteRequest={completeAssessment}
                onCancel={cancelAssessment}
            />
        </>
    );
}

export default FalsePositiveCVEsTable;
