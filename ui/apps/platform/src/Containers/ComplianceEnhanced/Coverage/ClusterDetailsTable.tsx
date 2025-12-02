import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom-v5-compat';
import {
    Content,
    ContentVariants,
    Pagination,
    Toolbar,
    ToolbarContent,
    ToolbarGroup,
    ToolbarItem,
} from '@patternfly/react-core';
import { ExpandableRowContent, Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import CompoundSearchFilter from 'Components/CompoundSearchFilter/components/CompoundSearchFilter';
import CompoundSearchFilterLabels from 'Components/CompoundSearchFilter/components/CompoundSearchFilterLabels';
import SearchFilterSelectInclusive from 'Components/CompoundSearchFilter/components/SearchFilterSelectInclusive';
import type { OnSearchCallback } from 'Components/CompoundSearchFilter/types';
import TbodyUnified from 'Components/TableStateTemplates/TbodyUnified';
import type { UseURLPaginationResult } from 'hooks/useURLPagination';
import type { UseURLSortResult } from 'hooks/useURLSort';
import type { ComplianceCheckResult } from 'services/ComplianceResultsService';
import type { TableUIState } from 'utils/getTableUIState';
import type { SearchFilter } from 'types/search';

import { DETAILS_TAB, TAB_NAV_QUERY } from './CheckDetailsPage';
import { CHECK_NAME_QUERY } from './compliance.coverage.constants';
import { coverageCheckDetailsPath } from './compliance.coverage.routes';
import { getClusterResultsStatusObject } from './compliance.coverage.utils';
import ControlLabels from './components/ControlLabels';
import StatusIcon from './components/StatusIcon';
import useScanConfigRouter from './hooks/useScanConfigRouter';
import {
    attributeForComplianceCheckStatus,
    profileCheckSearchFilterConfig,
} from '../searchFilterConfig';

const searchFilterConfig = [profileCheckSearchFilterConfig];

export type ClusterDetailsTableProps = {
    checkResultsCount: number;
    profileName: string;
    tableState: TableUIState<ComplianceCheckResult>;
    pagination: UseURLPaginationResult;
    getSortParams: UseURLSortResult['getSortParams'];
    searchFilter: SearchFilter;
    onFilterChange: (newFilter: SearchFilter) => void;
    onSearch: OnSearchCallback;
    onClearFilters: () => void;
};

function ClusterDetailsTable({
    checkResultsCount,
    profileName,
    tableState,
    pagination,
    getSortParams,
    searchFilter,
    onFilterChange,
    onSearch,
    onClearFilters,
}: ClusterDetailsTableProps) {
    const { page, perPage, setPage, setPerPage } = pagination;
    const { generatePathWithScanConfig } = useScanConfigRouter();
    const [expandedRows, setExpandedRows] = useState<number[]>([]);

    function toggleRow(selectedRowIndex: number) {
        const newExpandedRows = expandedRows.includes(selectedRowIndex)
            ? expandedRows.filter((index) => index !== selectedRowIndex)
            : [...expandedRows, selectedRowIndex];
        setExpandedRows(newExpandedRows);
    }

    useEffect(() => {
        setExpandedRows([]);
    }, [page, perPage, tableState]);

    return (
        <>
            <Toolbar>
                <ToolbarContent>
                    <ToolbarGroup className="pf-v6-u-w-100">
                        <ToolbarItem className="pf-v6-u-flex-1">
                            <CompoundSearchFilter
                                config={searchFilterConfig}
                                searchFilter={searchFilter}
                                onSearch={onSearch}
                            />
                        </ToolbarItem>
                        <ToolbarItem>
                            <SearchFilterSelectInclusive
                                attribute={attributeForComplianceCheckStatus}
                                isSeparate
                                onSearch={onSearch}
                                searchFilter={searchFilter}
                            />
                        </ToolbarItem>
                    </ToolbarGroup>
                    <ToolbarGroup className="pf-v6-u-w-100">
                        <CompoundSearchFilterLabels
                            attributesSeparateFromConfig={[attributeForComplianceCheckStatus]}
                            config={searchFilterConfig}
                            onFilterChange={onFilterChange}
                            searchFilter={searchFilter}
                        />
                    </ToolbarGroup>
                    <ToolbarGroup className="pf-v6-u-w-100">
                        <ToolbarItem variant="pagination" align={{ default: 'alignEnd' }}>
                            <Pagination
                                itemCount={checkResultsCount}
                                page={page}
                                perPage={perPage}
                                onSetPage={(_, newPage) => setPage(newPage)}
                                onPerPageSelect={(_, newPerPage) => setPerPage(newPerPage)}
                            />
                        </ToolbarItem>
                    </ToolbarGroup>
                </ToolbarContent>
            </Toolbar>
            <Table>
                <Thead>
                    <Tr>
                        <Th sort={getSortParams(CHECK_NAME_QUERY)}>Check</Th>
                        <Th modifier="fitContent" width={10}>
                            Controls
                        </Th>
                        <Th modifier="fitContent" width={10}>
                            Compliance status
                        </Th>
                    </Tr>
                </Thead>
                <TbodyUnified
                    tableState={tableState}
                    colSpan={3}
                    errorProps={{
                        title: 'There was an error loading results for this cluster',
                    }}
                    emptyProps={{
                        message:
                            'If you have recently created a scan schedule, please wait a few minutes for the results to become available.',
                    }}
                    filteredEmptyProps={{ onClearFilters }}
                    renderer={({ data }) => (
                        <>
                            {data.map((checkResult, rowIndex) => {
                                const { checkName, rationale, status, controls } = checkResult;
                                const clusterStatusObject = getClusterResultsStatusObject(status);
                                const isRowExpanded = expandedRows.includes(rowIndex);

                                return (
                                    <Tbody isExpanded={isRowExpanded} key={checkName}>
                                        <Tr>
                                            <Td dataLabel="Check">
                                                <Link
                                                    to={`${generatePathWithScanConfig(
                                                        coverageCheckDetailsPath,
                                                        {
                                                            checkName,
                                                            profileName,
                                                        },
                                                        {
                                                            customParams: {
                                                                [TAB_NAV_QUERY]: DETAILS_TAB,
                                                            },
                                                        }
                                                    )}`}
                                                >
                                                    {checkName}
                                                </Link>
                                                {/*
                                                    grid display is required to prevent the cell from
                                                    expanding to the text length. The Truncate PF component
                                                    is not used here because it displays a tooltip on hover
                                                */}
                                                <div style={{ display: 'grid' }}>
                                                    <Content
                                                        component={ContentVariants.small}
                                                        className="pf-v6-u-color-200 pf-v6-u-text-truncate"
                                                    >
                                                        {rationale}
                                                    </Content>
                                                </div>
                                            </Td>
                                            <Td
                                                dataLabel="Controls"
                                                modifier="fitContent"
                                                compoundExpand={
                                                    controls.length > 1
                                                        ? {
                                                              isExpanded: isRowExpanded,
                                                              onToggle: () => toggleRow(rowIndex),
                                                              rowIndex,
                                                              columnIndex: 1,
                                                          }
                                                        : undefined
                                                }
                                            >
                                                {controls.length > 1 ? (
                                                    `${controls.length} controls`
                                                ) : controls.length === 1 ? (
                                                    <ControlLabels controls={controls} />
                                                ) : (
                                                    '-'
                                                )}
                                            </Td>
                                            <Td dataLabel="Compliance status" modifier="fitContent">
                                                <StatusIcon
                                                    clusterStatusObject={clusterStatusObject}
                                                />
                                            </Td>
                                        </Tr>
                                        {isRowExpanded && (
                                            <Tr isExpanded={isRowExpanded}>
                                                <Td colSpan={6}>
                                                    <ExpandableRowContent>
                                                        <ControlLabels controls={controls} />
                                                    </ExpandableRowContent>
                                                </Td>
                                            </Tr>
                                        )}
                                    </Tbody>
                                );
                            })}
                        </>
                    )}
                />
            </Table>
        </>
    );
}

export default ClusterDetailsTable;
