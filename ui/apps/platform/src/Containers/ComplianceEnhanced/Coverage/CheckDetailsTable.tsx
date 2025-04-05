import React from 'react';
import { Link } from 'react-router-dom';
import {
    Pagination,
    Toolbar,
    ToolbarContent,
    ToolbarGroup,
    ToolbarItem,
} from '@patternfly/react-core';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import TbodyUnified from 'Components/TableStateTemplates/TbodyUnified';
import { UseURLPaginationResult } from 'hooks/useURLPagination';
import { UseURLSortResult } from 'hooks/useURLSort';
import { ClusterCheckStatus } from 'services/ComplianceResultsService';
import { TableUIState } from 'utils/getTableUIState';

import CompoundSearchFilter from 'Components/CompoundSearchFilter/components/CompoundSearchFilter';
import { makeFilterChipDescriptors } from 'Components/CompoundSearchFilter/utils/utils';
import SearchFilterChips from 'Components/PatternFly/SearchFilterChips';
import { CompoundSearchFilterConfig, OnSearchPayload } from 'Components/CompoundSearchFilter/types';
import { SearchFilter } from 'types/search';

import { coverageClusterDetailsPath } from './compliance.coverage.routes';
import {
    getClusterResultsStatusObject,
    getTimeDifferenceAsPhrase,
} from './compliance.coverage.utils';
import CheckStatusDropdown from './components/CheckStatusDropdown';
import StatusIcon from './components/StatusIcon';
import useScanConfigRouter from './hooks/useScanConfigRouter';
import { complianceStatusFilterChipDescriptors } from '../searchFilterConfig';

export const tabContentIdForResults = 'check-details-Results-tab-section';

export type CheckDetailsTableProps = {
    checkResultsCount: number;
    currentDatetime: Date;
    pagination: UseURLPaginationResult;
    profileName: string;
    tableState: TableUIState<ClusterCheckStatus>;
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

function CheckDetailsTable({
    checkResultsCount,
    currentDatetime,
    pagination,
    profileName,
    tableState,
    getSortParams,
    searchFilterConfig,
    searchFilter,
    onFilterChange,
    onSearch,
    onCheckStatusSelect,
    onClearFilters,
}: CheckDetailsTableProps) {
    const { generatePathWithScanConfig } = useScanConfigRouter();
    const { page, perPage, setPage, setPerPage } = pagination;

    const filterChipGroupDescriptors = makeFilterChipDescriptors(searchFilterConfig);

    return (
        <div id={tabContentIdForResults}>
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
                        <Th sort={getSortParams('Cluster')}>Cluster</Th>
                        <Th>Last scanned</Th>
                        <Th>Compliance status</Th>
                    </Tr>
                </Thead>
                <TbodyUnified
                    tableState={tableState}
                    colSpan={3}
                    errorProps={{
                        title: 'There was an error loading results for this check',
                    }}
                    emptyProps={{
                        message:
                            'If you have recently created a scan schedule, please wait a few minutes for the results to become available.',
                    }}
                    filteredEmptyProps={{ onClearFilters }}
                    renderer={({ data }) => (
                        <Tbody>
                            {data.map((clusterInfo) => {
                                const {
                                    cluster: { clusterId, clusterName },
                                    lastScanTime,
                                    status,
                                } = clusterInfo;
                                const clusterStatusObject = getClusterResultsStatusObject(status);
                                const lastScanTimeAsPhrase = getTimeDifferenceAsPhrase(
                                    lastScanTime,
                                    currentDatetime
                                );

                                return (
                                    <Tr key={clusterId}>
                                        <Td dataLabel="Cluster">
                                            <Link
                                                to={generatePathWithScanConfig(
                                                    coverageClusterDetailsPath,
                                                    {
                                                        clusterId,
                                                        profileName,
                                                    }
                                                )}
                                            >
                                                {clusterName}
                                            </Link>
                                        </Td>
                                        <Td dataLabel="Last scanned">{lastScanTimeAsPhrase}</Td>
                                        <Td dataLabel="Compliance status">
                                            <StatusIcon clusterStatusObject={clusterStatusObject} />
                                        </Td>
                                    </Tr>
                                );
                            })}
                        </Tbody>
                    )}
                />
            </Table>
        </div>
    );
}

export default CheckDetailsTable;
