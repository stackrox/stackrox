import React from 'react';
import { Link } from 'react-router-dom';
import {
    Divider,
    Pagination,
    Progress,
    ProgressMeasureLocation,
    Toolbar,
    ToolbarContent,
    ToolbarItem,
    Tooltip,
} from '@patternfly/react-core';
import { InnerScrollContainer, Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import TbodyUnified from 'Components/TableStateTemplates/TbodyUnified';
import { UseURLPaginationResult } from 'hooks/useURLPagination';
import { UseURLSortResult } from 'hooks/useURLSort';
import { ComplianceClusterOverallStats } from 'services/ComplianceCommon';
import { getDistanceStrictAsPhrase } from 'utils/dateUtils';
import { TableUIState } from 'utils/getTableUIState';

import { coverageClusterDetailsPath } from './compliance.coverage.routes';
import { calculateCompliancePercentage, getStatusCounts } from './compliance.coverage.utils';
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
    const { page, perPage, setPage, setPerPage } = pagination;

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
                            <Th modifier="fitContent" width={10}>
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
                                    const passPercentage = calculateCompliancePercentage(
                                        passCount,
                                        totalCount
                                    );
                                    const progressBarId = `progress-bar-${clusterId}`;
                                    const firstDiscoveredAsPhrase = getDistanceStrictAsPhrase(
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
                                                {firstDiscoveredAsPhrase}
                                            </Td>
                                            <Td dataLabel="Pass status" modifier="fitContent">
                                                <StatusCountIcon
                                                    text="check"
                                                    status="pass"
                                                    count={passCount}
                                                />
                                            </Td>
                                            <Td dataLabel="Fail status" modifier="fitContent">
                                                <StatusCountIcon
                                                    text="check"
                                                    status="fail"
                                                    count={failCount}
                                                />
                                            </Td>
                                            <Td dataLabel="Manual status" modifier="fitContent">
                                                <StatusCountIcon
                                                    text="check"
                                                    status="manual"
                                                    count={manualCount}
                                                />
                                            </Td>
                                            <Td dataLabel="Other status" modifier="fitContent">
                                                <StatusCountIcon
                                                    text="check"
                                                    status="other"
                                                    count={otherCount}
                                                />
                                            </Td>
                                            <Td dataLabel="Compliance">
                                                <Progress
                                                    id={progressBarId}
                                                    value={passPercentage}
                                                    measureLocation={
                                                        ProgressMeasureLocation.outside
                                                    }
                                                    aria-label={`${clusterName} compliance percentage`}
                                                />
                                                <Tooltip
                                                    content={
                                                        <div>
                                                            {`${passCount} / ${totalCount} checks are passing for this cluster`}
                                                        </div>
                                                    }
                                                    triggerRef={() =>
                                                        document.getElementById(
                                                            progressBarId
                                                        ) as HTMLButtonElement
                                                    }
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
