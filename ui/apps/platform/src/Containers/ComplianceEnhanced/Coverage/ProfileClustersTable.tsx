import React from 'react';
import { Link } from 'react-router-dom';
import { Divider, Pagination, Toolbar, ToolbarContent, ToolbarItem } from '@patternfly/react-core';
import { InnerScrollContainer, Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import TbodyUnified from 'Components/TableStateTemplates/TbodyUnified';
import { UseURLPaginationResult } from 'hooks/useURLPagination';
import useURLSearch from 'hooks/useURLSearch';
import { UseURLSortResult } from 'hooks/useURLSort';
import { ComplianceClusterOverallStats } from 'services/ComplianceCommon';
import { TableUIState } from 'utils/getTableUIState';
import { getPercentage } from 'utils/mathUtils';

import { CHECK_STATUS_QUERY } from './compliance.coverage.constants';
import { coverageClusterDetailsPath } from './compliance.coverage.routes';
import { getStatusCounts, getTimeDifferenceAsPhrase } from './compliance.coverage.utils';
import ComplianceProgressBar from './components/ComplianceProgressBar';
import ProfilesTableToggleGroup from './components/ProfilesTableToggleGroup';
import StatusCountIcon from './components/StatusCountIcon';
import useScanConfigRouter from './hooks/useScanConfigRouter';

export type ProfileClustersTableProps = {
    currentDatetime: Date;
    pagination: UseURLPaginationResult;
    profileClustersResultsCount: number;
    profileName: string;
    tableState: TableUIState<ComplianceClusterOverallStats>;
    getSortParams: UseURLSortResult['getSortParams'];
    onClearFilters: () => void;
};

function ProfileClustersTable({
    currentDatetime,
    pagination,
    profileClustersResultsCount,
    profileName,
    tableState,
    getSortParams,
    onClearFilters,
}: ProfileClustersTableProps) {
    const { generatePathWithScanConfig } = useScanConfigRouter();
    const { searchFilter } = useURLSearch();
    const { page, perPage, setPage, setPerPage } = pagination;

    function isComplianceStatusFiltered() {
        return CHECK_STATUS_QUERY in searchFilter;
    }

    function shouldDisableIcon(statuses) {
        const statusFilter = searchFilter[CHECK_STATUS_QUERY];
        if (!statusFilter) {
            return false;
        }
        return !statuses.some((status) => statusFilter.includes(status));
    }

    return (
        <>
            <Toolbar>
                <ToolbarContent>
                    <ToolbarItem>
                        <ProfilesTableToggleGroup activeToggle="clusters" />
                    </ToolbarItem>
                    <ToolbarItem variant="pagination" align={{ default: 'alignRight' }}>
                        <Pagination
                            itemCount={profileClustersResultsCount}
                            page={page}
                            perPage={perPage}
                            onSetPage={(_, newPage) => setPage(newPage)}
                            onPerPageSelect={(_, newPerPage) => setPerPage(newPerPage)}
                        />
                    </ToolbarItem>
                </ToolbarContent>
            </Toolbar>
            <Divider />
            <InnerScrollContainer>
                <Table>
                    <Thead>
                        <Tr>
                            <Th sort={getSortParams('Cluster')} modifier="fitContent" width={10}>
                                Cluster
                            </Th>
                            <Th modifier="fitContent">Last scanned</Th>
                            <Th modifier="fitContent">Pass status</Th>
                            <Th modifier="fitContent">Fail status</Th>
                            <Th modifier="fitContent">Manual status</Th>
                            <Th modifier="fitContent">Other status</Th>
                            <Th
                                modifier="fitContent"
                                width={10}
                                info={{
                                    tooltip:
                                        'Compliance is calculated as the percentage of passing checks out of the total checks. Compliance cannot be calculated when status filters are applied.',
                                }}
                            >
                                Compliance
                            </Th>
                        </Tr>
                    </Thead>
                    <TbodyUnified
                        tableState={tableState}
                        colSpan={7}
                        errorProps={{
                            title: 'There was an error loading profile clusters',
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
                                        checkStats,
                                    } = clusterInfo;
                                    const {
                                        passCount,
                                        failCount,
                                        manualCount,
                                        otherCount,
                                        totalCount,
                                    } = getStatusCounts(checkStats);
                                    const passPercentage = getPercentage(passCount, totalCount);
                                    const progressBarId = `progress-bar-${clusterId}`;
                                    const lastScanTimeAsPhrase = getTimeDifferenceAsPhrase(
                                        lastScanTime,
                                        currentDatetime
                                    );

                                    return (
                                        <Tr key={clusterId}>
                                            <Td dataLabel="Cluster" modifier="fitContent">
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
                                            <Td dataLabel="Last scanned" modifier="fitContent">
                                                {lastScanTimeAsPhrase}
                                            </Td>
                                            <Td dataLabel="Pass status" modifier="fitContent">
                                                <StatusCountIcon
                                                    text="check"
                                                    status="pass"
                                                    count={passCount}
                                                    disabled={shouldDisableIcon(['Pass'])}
                                                />
                                            </Td>
                                            <Td dataLabel="Fail status" modifier="fitContent">
                                                <StatusCountIcon
                                                    text="check"
                                                    status="fail"
                                                    count={failCount}
                                                    disabled={shouldDisableIcon(['Fail'])}
                                                />
                                            </Td>
                                            <Td dataLabel="Manual status" modifier="fitContent">
                                                <StatusCountIcon
                                                    text="check"
                                                    status="manual"
                                                    count={manualCount}
                                                    disabled={shouldDisableIcon(['Manual'])}
                                                />
                                            </Td>
                                            <Td dataLabel="Other status" modifier="fitContent">
                                                <StatusCountIcon
                                                    text="check"
                                                    status="other"
                                                    count={otherCount}
                                                    disabled={shouldDisableIcon([
                                                        'Error',
                                                        'Info',
                                                        'Not Applicable',
                                                        'Inconsistent',
                                                        'Unset Check Status',
                                                    ])}
                                                />
                                            </Td>
                                            <Td dataLabel="Compliance">
                                                <ComplianceProgressBar
                                                    ariaLabel={`${clusterName} compliance percentage`}
                                                    isDisabled={isComplianceStatusFiltered()}
                                                    passPercentage={passPercentage}
                                                    progressBarId={progressBarId}
                                                    tooltipText={`${passCount} / ${totalCount} checks are passing for this cluster`}
                                                />
                                            </Td>
                                        </Tr>
                                    );
                                })}
                            </Tbody>
                        )}
                    />
                </Table>
            </InnerScrollContainer>
        </>
    );
}

export default ProfileClustersTable;
