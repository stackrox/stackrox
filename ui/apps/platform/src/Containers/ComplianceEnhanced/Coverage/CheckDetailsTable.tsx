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
import { getDistanceStrictAsPhrase } from 'utils/dateUtils';
import { TableUIState } from 'utils/getTableUIState';

import CompoundSearchFilter from 'Components/CompoundSearchFilter/components/CompoundSearchFilter';
import SearchFilterChips from 'Components/PatternFly/SearchFilterChips';
import {
    OnSearchPayload,
    PartialCompoundSearchFilterConfig,
} from 'Components/CompoundSearchFilter/types';
import { SearchFilter } from 'types/search';

import { coverageClusterDetailsPath } from './compliance.coverage.routes';
import { getClusterResultsStatusObject } from './compliance.coverage.utils';
import { CHECK_STATUS_QUERY, CLUSTER_QUERY } from './compliance.coverage.constants';
import CheckStatusDropdown from './components/CheckStatusDropdown';
import StatusIcon from './components/StatusIcon';
import useScanConfigRouter from './hooks/useScanConfigRouter';

export const tabContentIdForResults = 'check-details-Results-tab-section';

export type CheckDetailsTableProps = {
    checkResultsCount: number;
    currentDatetime: Date;
    pagination: UseURLPaginationResult;
    profileName: string;
    tableState: TableUIState<ClusterCheckStatus>;
    getSortParams: UseURLSortResult['getSortParams'];
    searchFilterConfig: PartialCompoundSearchFilterConfig;
    searchFilter: SearchFilter;
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
    onSearch,
    onCheckStatusSelect,
    onClearFilters,
}: CheckDetailsTableProps) {
    const { generatePathWithScanConfig } = useScanConfigRouter();
    const { page, perPage, setPage, setPerPage } = pagination;

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
                            filterChipGroupDescriptors={[
                                {
                                    displayName: 'Cluster',
                                    searchFilterName: CLUSTER_QUERY,
                                },
                                {
                                    displayName: 'Compliance Status',
                                    searchFilterName: CHECK_STATUS_QUERY,
                                },
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
                        message: 'No results found for this check',
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
                                const firstDiscoveredAsPhrase = getDistanceStrictAsPhrase(
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
                                        <Td dataLabel="Last scanned">{firstDiscoveredAsPhrase}</Td>
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
