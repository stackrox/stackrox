/* eslint-disable no-nested-ternary */
/* eslint-disable react/no-array-index-key */
import React, { ReactElement, useState } from 'react';
import { TableComposable, Thead, Tbody, Tr, Th, Td, IActions } from '@patternfly/react-table';
import {
    Bullseye,
    Button,
    ButtonVariant,
    Divider,
    DropdownItem,
    Flex,
    FlexItem,
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
import { UsePaginationResult } from 'hooks/patternfly/usePagination';
import VulnerabilitySeverityIconText from 'Components/PatternFly/IconText/VulnerabilitySeverityIconText';
import DateTimeFormat from 'Components/PatternFly/DateTimeFormat';
import usePermissions from 'hooks/usePermissions';
import { SearchFilter } from 'types/search';
import EmptyStateTemplate from 'Components/PatternFly/EmptyStateTemplate';
import { GetSortParams } from 'hooks/patternfly/useTableSort';
import DeferralFormModal from './DeferralFormModal';
import FalsePositiveRequestModal from './FalsePositiveFormModal';
import { Vulnerability, EmbeddedImageScanComponent } from '../imageVulnerabilities.graphql';
import useDeferVulnerability from './useDeferVulnerability';
import useMarkFalsePositive from './useMarkFalsePositive';
import PendingApprovalPopover from './PendingApprovalPopover';
import CVESummaryLink from '../CVESummaryLink';
import ImageVulnsSearchFilter from '../ImageVulnsSearchFilter';
import SearchFilterResults from '../SearchFilterResults';

export type CVEsToBeAssessed = {
    type: 'DEFERRAL' | 'FALSE_POSITIVE';
    cves: string[];
} | null;

export type ObservedCVEsTableProps = {
    rows: Vulnerability[];
    isLoading: boolean;
    registry: string;
    remote: string;
    tag: string;
    itemCount: number;
    updateTable: () => void;
    searchFilter: SearchFilter;
    setSearchFilter: React.Dispatch<React.SetStateAction<SearchFilter>>;
    getSortParams: GetSortParams;
    showComponentDetails: (components: EmbeddedImageScanComponent[], cveName: string) => void;
} & UsePaginationResult;

function ObservedCVEsTable({
    rows,
    registry,
    remote,
    tag,
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
}: ObservedCVEsTableProps): ReactElement {
    const {
        selected,
        allRowsSelected,
        numSelected,
        onSelect,
        onSelectAll,
        getSelectedIds,
        onClearAll,
    } = useTableSelection<Vulnerability>(rows);
    const [cvesToBeAssessed, setCVEsToBeAssessed] = useState<CVEsToBeAssessed>(null);
    const requestDeferral = useDeferVulnerability({
        cves: cvesToBeAssessed?.cves || [],
        registry,
        remote,
        tag,
    });
    const requestFalsePositive = useMarkFalsePositive({
        cves: cvesToBeAssessed?.cves || [],
        registry,
        remote,
        tag,
    });
    const { hasReadWriteAccess } = usePermissions();

    function cancelAssessment() {
        setCVEsToBeAssessed(null);
    }

    function completeAssessment() {
        onClearAll();
        setCVEsToBeAssessed(null);
        updateTable();
    }

    const canCreateRequests = hasReadWriteAccess('VulnerabilityManagementRequests');

    const selectedIds = getSelectedIds();
    const selectedVulnsToDeferOrMarkFalsePositive = canCreateRequests
        ? rows
              .filter((row) => {
                  return selectedIds.includes(row.id);
              })
              .map((row) => row.cve)
        : [];

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
                                key="upgrade"
                                component="button"
                                onClick={() =>
                                    setCVEsToBeAssessed({
                                        type: 'DEFERRAL',
                                        cves: selectedVulnsToDeferOrMarkFalsePositive,
                                    })
                                }
                                isDisabled={selectedVulnsToDeferOrMarkFalsePositive.length === 0}
                            >
                                Defer CVE ({selectedVulnsToDeferOrMarkFalsePositive.length})
                            </DropdownItem>
                            <DropdownItem
                                key="delete"
                                component="button"
                                onClick={() =>
                                    setCVEsToBeAssessed({
                                        type: 'FALSE_POSITIVE',
                                        cves: selectedVulnsToDeferOrMarkFalsePositive,
                                    })
                                }
                                isDisabled={selectedVulnsToDeferOrMarkFalsePositive.length === 0}
                            >
                                Mark false positive (
                                {selectedVulnsToDeferOrMarkFalsePositive.length})
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
                        <Th sort={getSortParams('CVSS')}>CVSS score</Th>
                        <Th>Affected components</Th>
                        <Th sort={getSortParams('Discovered')}>Discovered</Th>
                    </Tr>
                </Thead>
                <Tbody>
                    {isLoading && (
                        <Tr>
                            <Td colSpan={7}>
                                <Bullseye>
                                    <Spinner isSVG size="sm" />
                                </Bullseye>
                            </Td>
                        </Tr>
                    )}
                    {!isLoading && rows && rows.length === 0 ? (
                        <Tr>
                            <Td colSpan={7}>
                                <PageSection variant={PageSectionVariants.light} isFilled>
                                    <EmptyStateTemplate
                                        title="No CVEs available"
                                        headingLevel="h3"
                                    />
                                </PageSection>
                            </Td>
                        </Tr>
                    ) : (
                        rows.map((row, rowIndex) => {
                            const actions: IActions = [
                                {
                                    title: 'Defer CVE',
                                    onClick: (event) => {
                                        event.preventDefault();
                                        setCVEsToBeAssessed({ type: 'DEFERRAL', cves: [row.cve] });
                                    },
                                    isDisabled: !canCreateRequests,
                                },
                                {
                                    title: 'Mark as False Positive',
                                    onClick: (event) => {
                                        event.preventDefault();
                                        setCVEsToBeAssessed({
                                            type: 'FALSE_POSITIVE',
                                            cves: [row.cve],
                                        });
                                    },
                                    isDisabled: !canCreateRequests,
                                },
                            ];
                            return (
                                <Tr key={rowIndex}>
                                    <Td
                                        select={{
                                            rowIndex,
                                            onSelect,
                                            isSelected: selected[rowIndex],
                                        }}
                                    />
                                    <Td dataLabel="CVE">
                                        <Flex alignItems={{ default: 'alignItemsCenter' }}>
                                            <FlexItem>
                                                <CVESummaryLink cve={row.cve} id={row.id} />
                                            </FlexItem>
                                            {row.vulnerabilityRequest?.id &&
                                                !row.vulnerabilityRequest.expired && (
                                                    <FlexItem>
                                                        <PendingApprovalPopover
                                                            vulnRequestId={
                                                                row.vulnerabilityRequest.id
                                                            }
                                                        />
                                                    </FlexItem>
                                                )}
                                        </Flex>
                                    </Td>
                                    <Td dataLabel="Fixable">{row.isFixable ? 'Yes' : 'No'}</Td>
                                    <Td dataLabel="Severity">
                                        <VulnerabilitySeverityIconText severity={row.severity} />
                                    </Td>
                                    <Td dataLabel="CVSS score">{Number(row.cvss).toFixed(1)}</Td>
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
                                    <Td dataLabel="Discovered">
                                        <DateTimeFormat time={row.discoveredAtImage} />
                                    </Td>
                                    <Td
                                        className="pf-u-text-align-right"
                                        actions={{
                                            items: actions,
                                        }}
                                    />
                                </Tr>
                            );
                        })
                    )}
                </Tbody>
            </TableComposable>
            <DeferralFormModal
                isOpen={cvesToBeAssessed?.type === 'DEFERRAL'}
                numCVEsToBeAssessed={cvesToBeAssessed?.cves.length || 0}
                onSendRequest={requestDeferral}
                onCompleteRequest={completeAssessment}
                onCancelDeferral={cancelAssessment}
            />
            <FalsePositiveRequestModal
                isOpen={cvesToBeAssessed?.type === 'FALSE_POSITIVE'}
                numCVEsToBeAssessed={cvesToBeAssessed?.cves.length || 0}
                onSendRequest={requestFalsePositive}
                onCompleteRequest={completeAssessment}
                onCancelFalsePositive={cancelAssessment}
            />
        </>
    );
}

export default ObservedCVEsTable;
