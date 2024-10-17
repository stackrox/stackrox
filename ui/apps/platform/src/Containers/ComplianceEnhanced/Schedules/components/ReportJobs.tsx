import React, { useCallback } from 'react';
import {
    Card,
    CardBody,
    Divider,
    Pagination,
    Toolbar,
    ToolbarContent,
    ToolbarItem,
    useInterval,
} from '@patternfly/react-core';

import {
    ComplianceReportSnapshot,
    fetchComplianceReportHistory,
} from 'services/ComplianceScanConfigurationService';
import JobDetails from 'Containers/Vulnerabilities/VulnerablityReporting/ViewVulnReport/JobDetails';
import ReportJobsTable from 'Components/ReportJob/ReportJobsTable';
import useURLPagination from 'hooks/useURLPagination';
import useURLSearch from 'hooks/useURLSearch';
import { ensureBoolean, ensureStringArray } from 'utils/ensure';
import useURLStringUnion from 'hooks/useURLStringUnion';
import { RunState } from 'types/reportJob';
import useAnalytics from 'hooks/useAnalytics';
import useRestQuery from 'hooks/useRestQuery';
import useURLSort from 'hooks/useURLSort';
import { getRequestQueryStringForSearchFilter } from 'utils/searchUtils';
import { getTableUIState } from 'utils/getTableUIState';
import ConfigDetails from './ConfigDetails';
import ReportRunStatesFilter, { ensureReportRunStates } from './ReportRunStatesFilter';
import MyJobsFilter from './MyJobsFilter';

function getJobId(snapshot: ComplianceReportSnapshot) {
    return snapshot.reportJobId;
}

function getConfigName(snapshot: ComplianceReportSnapshot) {
    return snapshot.name;
}

const sortOptions = {
    sortFields: ['Compliance Report Completed Time'],
    defaultSortOption: { field: 'Compliance Report Completed Time', direction: 'desc' } as const,
};

type ReportJobsProps = {
    scanConfigId: string;
    isComplianceReportingEnabled: boolean;
};

function ReportJobs({ scanConfigId, isComplianceReportingEnabled }: ReportJobsProps) {
    const { analyticsTrack } = useAnalytics();

    const { page, perPage, setPage, setPerPage } = useURLPagination(10);
    const { sortOption, getSortParams } = useURLSort(sortOptions);
    const { searchFilter, setSearchFilter } = useURLSearch();
    const [isViewingOnlyMyJobs, setIsViewingOnlyMyJobs] = useURLStringUnion('viewOnlyMyJobs', [
        'false',
        'true',
    ]);

    const filteredReportRunStates = ensureStringArray(searchFilter['Compliance Report State']);

    const query = getRequestQueryStringForSearchFilter({
        'Compliance Report State': filteredReportRunStates,
    });

    const fetchComplianceReportHistoryCallback = useCallback(
        () =>
            fetchComplianceReportHistory({
                id: scanConfigId,
                query,
                page,
                perPage,
                sortOption,
                showMyHistory: isViewingOnlyMyJobs === 'true',
            }),
        [isViewingOnlyMyJobs, page, perPage, query, scanConfigId, sortOption]
    );
    const {
        data: complianceScanSnapshots,
        isLoading,
        error,
        refetch,
    } = useRestQuery(fetchComplianceReportHistoryCallback);

    const tableState = getTableUIState({
        isLoading,
        data: complianceScanSnapshots,
        error,
        searchFilter,
        isPolling: true,
    });

    const onReportStatesFilterChange = (_checked: boolean, selectedStatus: RunState) => {
        const isStatusIncluded = filteredReportRunStates.includes(selectedStatus);
        const newFilters = isStatusIncluded
            ? ensureReportRunStates(
                  filteredReportRunStates.filter((status) => status !== selectedStatus)
              )
            : ensureReportRunStates([...filteredReportRunStates, selectedStatus]);
        analyticsTrack({
            event: 'Compliance Report Run State Filtered',
            properties: {
                value: newFilters,
            },
        });
        setSearchFilter({
            ...searchFilter,
            'Compliance Report State': newFilters,
        });
        setPage(1);
    };

    const onMyJobsFilterChange = (checked: boolean) => {
        analyticsTrack({
            event: 'Compliance Report Jobs View Toggled',
            properties: {
                view: 'My jobs',
                state: checked,
            },
        });
        setIsViewingOnlyMyJobs(String(checked));
        setPage(1);
    };

    useInterval(refetch, 10000);

    return (
        <>
            <Toolbar>
                <ToolbarContent>
                    <ToolbarItem alignItems="center">
                        <ReportRunStatesFilter
                            reportRunStates={ensureReportRunStates(filteredReportRunStates)}
                            onChange={onReportStatesFilterChange}
                        />
                    </ToolbarItem>
                    <ToolbarItem className="pf-v5-u-flex-grow-1" alignSelf="center">
                        <MyJobsFilter
                            isViewingOnlyMyJobs={ensureBoolean(isViewingOnlyMyJobs)}
                            onMyJobsFilterChange={onMyJobsFilterChange}
                        />
                    </ToolbarItem>
                    <ToolbarItem variant="pagination" align={{ default: 'alignRight' }}>
                        <Pagination
                            toggleTemplate={({ firstIndex, lastIndex }) => (
                                <span>
                                    <b>
                                        {firstIndex} - {lastIndex}
                                    </b>{' '}
                                    of <b>many</b>
                                </span>
                            )}
                            page={page}
                            perPage={perPage}
                            onSetPage={(_, newPage) => setPage(newPage)}
                            onPerPageSelect={(_, newPerPage) => setPerPage(newPerPage)}
                            isCompact
                        />
                    </ToolbarItem>
                </ToolbarContent>
            </Toolbar>
            <ReportJobsTable
                tableState={tableState}
                getSortParams={getSortParams}
                getJobId={getJobId}
                getConfigName={getConfigName}
                onClearFilters={() => {
                    setSearchFilter({});
                    setPage(1);
                }}
                onDeleteDownload={() => {}}
                renderExpandableRowContent={(snapshot: ComplianceReportSnapshot) => {
                    return (
                        <>
                            <Card isFlat>
                                <CardBody>
                                    <JobDetails
                                        reportStatus={snapshot.reportStatus}
                                        isDownloadAvailable={snapshot.isDownloadAvailable}
                                    />
                                    <Divider component="div" className="pf-v5-u-my-md" />
                                    <ConfigDetails
                                        scanConfig={snapshot.reportData}
                                        isComplianceReportingEnabled={isComplianceReportingEnabled}
                                    />
                                </CardBody>
                            </Card>
                        </>
                    );
                }}
            />
        </>
    );
}

export default ReportJobs;
