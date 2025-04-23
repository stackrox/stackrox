import React, { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import {
    Pagination,
    Text,
    TextVariants,
    Toolbar,
    ToolbarContent,
    ToolbarGroup,
    ToolbarItem,
} from '@patternfly/react-core';
import { ExpandableRowContent, Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import CompoundSearchFilter from 'Components/CompoundSearchFilter/components/CompoundSearchFilter';
import { makeFilterChipDescriptors } from 'Components/CompoundSearchFilter/utils/utils';
import { CompoundSearchFilterConfig, OnSearchPayload } from 'Components/CompoundSearchFilter/types';
import SearchFilterChips from 'Components/PatternFly/SearchFilterChips';
import TbodyUnified from 'Components/TableStateTemplates/TbodyUnified';
import { UseURLPaginationResult } from 'hooks/useURLPagination';
import { UseURLSortResult } from 'hooks/useURLSort';
import { ComplianceCheckResult } from 'services/ComplianceResultsService';
import { TableUIState } from 'utils/getTableUIState';
import { SearchFilter } from 'types/search';

import { DETAILS_TAB, TAB_NAV_QUERY } from './CheckDetailsPage';
import { CHECK_NAME_QUERY } from './compliance.coverage.constants';
import { coverageCheckDetailsPath } from './compliance.coverage.routes';
import { getClusterResultsStatusObject } from './compliance.coverage.utils';
import CheckStatusDropdown from './components/CheckStatusDropdown';
import ControlLabels from './components/ControlLabels';
import StatusIcon from './components/StatusIcon';
import useScanConfigRouter from './hooks/useScanConfigRouter';
import { complianceStatusFilterChipDescriptors } from '../searchFilterConfig';

export type ClusterDetailsTableProps = {
    checkResultsCount: number;
    profileName: string;
    tableState: TableUIState<ComplianceCheckResult>;
    pagination: UseURLPaginationResult;
    getSortParams: UseURLSortResult['getSortParams'];
    searchFilterConfig: CompoundSearchFilterConfig;
    searchFilter: SearchFilter;
    onFilterChange: (newFilter: SearchFilter) => void;
    onSearch: (payload: OnSearchPayload) => void;
    onCheckStatusSelect: (
        filterType: 'Compliance Check Status',
        checked: boolean,
        selection: string
    ) => void;
    onClearFilters: () => void;
};

function ClusterDetailsTable({
    checkResultsCount,
    profileName,
    tableState,
    pagination,
    getSortParams,
    searchFilterConfig,
    searchFilter,
    onFilterChange,
    onSearch,
    onCheckStatusSelect,
    onClearFilters,
}: ClusterDetailsTableProps) {
    /* eslint-disable no-nested-ternary */
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

    const filterChipGroupDescriptors = makeFilterChipDescriptors(searchFilterConfig);

    return (
        <>
            <Toolbar>
                <ToolbarContent>
                    <ToolbarGroup className="pf-v5-u-w-100">
                        <ToolbarItem className="pf-v5-u-flex-1">
                            <CompoundSearchFilter
                                config={searchFilterConfig}
                                searchFilter={searchFilter}
                                onSearch={onSearch}
                            />
                        </ToolbarItem>
                        <ToolbarItem>
                            <CheckStatusDropdown
                                searchFilter={searchFilter}
                                onSelect={onCheckStatusSelect}
                            />
                        </ToolbarItem>
                        <ToolbarItem variant="pagination" align={{ default: 'alignRight' }}>
                            <Pagination
                                itemCount={checkResultsCount}
                                page={page}
                                perPage={perPage}
                                onSetPage={(_, newPage) => setPage(newPage)}
                                onPerPageSelect={(_, newPerPage) => setPerPage(newPerPage)}
                            />
                        </ToolbarItem>
                    </ToolbarGroup>
                    <ToolbarGroup className="pf-v5-u-w-100">
                        <SearchFilterChips
                            searchFilter={searchFilter}
                            onFilterChange={onFilterChange}
                            filterChipGroupDescriptors={[
                                ...filterChipGroupDescriptors,
                                complianceStatusFilterChipDescriptors,
                            ]}
                        />
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
                                                    <Text
                                                        component={TextVariants.small}
                                                        className="pf-v5-u-color-200 pf-v5-u-text-truncate"
                                                    >
                                                        {rationale}
                                                    </Text>
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
