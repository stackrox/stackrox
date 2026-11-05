import { Link } from 'react-router-dom-v5-compat';
import {
    Pagination,
    Toolbar,
    ToolbarContent,
    ToolbarGroup,
    ToolbarItem,
} from '@patternfly/react-core';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import TbodyUnified from 'Components/TableStateTemplates/TbodyUnified';
import useFeatureFlags from 'hooks/useFeatureFlags';
import type { UseURLPaginationResult } from 'hooks/useURLPagination';
import type { UseURLSortResult } from 'hooks/useURLSort';
import type { ClusterCheckStatus } from 'services/ComplianceResultsService';
import type { TableUIState } from 'utils/getTableUIState';

import CompoundSearchFilter from 'Components/CompoundSearchFilter/components/CompoundSearchFilter';
import CompoundSearchFilterLabels from 'Components/CompoundSearchFilter/components/CompoundSearchFilterLabels';
import SearchFilterSelectInclusive from 'Components/CompoundSearchFilter/components/SearchFilterSelectInclusive';
import type { OnSearchCallback } from 'Components/CompoundSearchFilter/types';
import type { SearchFilter } from 'types/search';

import { coverageClusterDetailsPath } from './compliance.coverage.routes';
import {
    getCheckScanStatus,
    getClusterResultsStatusObject,
    getTimeDifferenceAsPhrase,
} from './compliance.coverage.utils';
import StatusIcon from './components/StatusIcon';
import useScanConfigRouter from './hooks/useScanConfigRouter';
import {
    attributeForComplianceCheckStatus,
    clusterSearchFilterConfig,
} from '../searchFilterConfig';

const searchFilterConfig = [clusterSearchFilterConfig];

export const tabContentIdForResults = 'check-details-Results-tab-section';

export type CheckDetailsTableProps = {
    checkResultsCount: number;
    currentDatetime: Date;
    pagination: UseURLPaginationResult;
    profileName: string;
    tableState: TableUIState<ClusterCheckStatus>;
    getSortParams: UseURLSortResult['getSortParams'];
    searchFilter: SearchFilter;
    onFilterChange: (newFilter: SearchFilter) => void;
    onSearch: OnSearchCallback;
    onClearFilters: () => void;
};

function CheckDetailsTable({
    checkResultsCount,
    currentDatetime,
    pagination,
    profileName,
    tableState,
    getSortParams,
    searchFilter,
    onFilterChange,
    onSearch,
    onClearFilters,
}: CheckDetailsTableProps) {
    const { generatePathWithScanConfig } = useScanConfigRouter();
    const { page, perPage, setPage, setPerPage } = pagination;

    const { isFeatureFlagEnabled } = useFeatureFlags();
    const isScanStatusEnabled = isFeatureFlagEnabled('ROX_COMPLIANCE_SURFACE_STALE_DATA');
    const colSpan = isScanStatusEnabled ? 4 : 3;

    return (
        <div id={tabContentIdForResults}>
            <Toolbar>
                <ToolbarContent>
                    <CompoundSearchFilter
                        config={searchFilterConfig}
                        searchFilter={searchFilter}
                        onSearch={onSearch}
                    />
                    <SearchFilterSelectInclusive
                        attribute={attributeForComplianceCheckStatus}
                        isSeparate
                        onSearch={onSearch}
                        searchFilter={searchFilter}
                    />
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
                        <Th sort={getSortParams('Cluster')}>Cluster</Th>
                        <Th>Last scanned</Th>
                        {isScanStatusEnabled && <Th modifier="fitContent">Scan status</Th>}
                        <Th>Compliance status</Th>
                    </Tr>
                </Thead>
                <TbodyUnified
                    tableState={tableState}
                    colSpan={colSpan}
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
                                    dataFreshness,
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
                                        {isScanStatusEnabled && (
                                            <Td dataLabel="Scan status">
                                                {getCheckScanStatus(dataFreshness)}
                                            </Td>
                                        )}
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
