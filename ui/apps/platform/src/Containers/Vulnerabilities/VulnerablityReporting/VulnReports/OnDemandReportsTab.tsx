import React, { useCallback } from 'react';
import {
    Card,
    CardBody,
    PageSection,
    Pagination,
    Text,
    Toolbar,
    ToolbarContent,
    ToolbarItem,
    useInterval,
} from '@patternfly/react-core';

import { getTableUIState } from 'utils/getTableUIState';
import { ensureBoolean, ensureStringArray } from 'utils/ensure';
import { getRequestQueryStringForSearchFilter } from 'utils/searchUtils';
import useRestQuery from 'hooks/useRestQuery';
import useURLPagination from 'hooks/useURLPagination';
import useURLSort from 'hooks/useURLSort';
import useURLSearch from 'hooks/useURLSearch';
import useURLStringUnion from 'hooks/useURLStringUnion';
import { fetchOnDemandReportHistory } from 'services/ReportsService';
import PageTitle from 'Components/PageTitle';
import ReportJobStatusFilter, {
    ensureReportJobStatuses,
    ReportJobStatus,
} from 'Components/ReportJob/ReportJobStatusFilter';
import MyJobsFilter from 'Components/ReportJob/MyJobsFilter';
import OnDemandReportsTable from './OnDemandReportsTable';

const sortOptions = {
    sortFields: ['Report Completed Time'],
    defaultSortOption: { field: 'Report Completed Time', direction: 'desc' } as const,
};

function createQueryFromReportJobStatusFilters(jobStatusFilters: string[]) {
    const query: Record<string, string[]> = {
        'Report State': [],
    };

    const jobStatusQueryMappings: Record<string, { category: string; value: string }> = {
        PREPARING: { category: 'Report State', value: 'PREPARING' },
        WAITING: { category: 'Report State', value: 'WAITING' },
        ERROR: { category: 'Report State', value: 'FAILURE' },
        DOWNLOAD_GENERATED: {
            category: 'Report State',
            value: 'GENERATED',
        },
    };

    const reportJobStatuses = ensureReportJobStatuses(jobStatusFilters);

    reportJobStatuses.forEach((jobStatus) => {
        const queryMapping = jobStatusQueryMappings[jobStatus];
        if (queryMapping) {
            const { category, value } = queryMapping;
            query[category].push(value);
        }
    });

    return query;
}

function OnDemandReportsTab() {
    const { page, perPage, setPage, setPerPage } = useURLPagination(10);
    const { sortOption, getSortParams } = useURLSort(sortOptions);
    const { searchFilter, setSearchFilter } = useURLSearch();
    const [isViewingOnlyMyJobs, setIsViewingOnlyMyJobs] = useURLStringUnion('viewOnlyMyJobs', [
        'false',
        'true',
    ]);

    const reportJobStatusFilters = ensureStringArray(searchFilter['Report Job Status']);

    const query = getRequestQueryStringForSearchFilter({
        ...createQueryFromReportJobStatusFilters(reportJobStatusFilters),
    });

    const fetchOnDemandReportsHistoryCallback = useCallback(
        () =>
            fetchOnDemandReportHistory({
                query,
                page,
                perPage,
                sortOption,
                showMyHistory: isViewingOnlyMyJobs === 'true',
            }),
        [query, page, perPage, sortOption, isViewingOnlyMyJobs]
    );
    const { data, isLoading, error, refetch } = useRestQuery(fetchOnDemandReportsHistoryCallback, {
        clearErrorBeforeRequest: false,
    });

    // @TODO: Add polling

    const tableState = getTableUIState({
        isLoading,
        data,
        error,
        searchFilter,
        isPolling: true,
    });

    const onReportJobStatusFilterChange = (_checked: boolean, selectedStatus: ReportJobStatus) => {
        const isStatusIncluded = reportJobStatusFilters.includes(selectedStatus);
        const newFilters = isStatusIncluded
            ? ensureReportJobStatuses(
                  reportJobStatusFilters.filter((status) => status !== selectedStatus)
              )
            : ensureReportJobStatuses([...reportJobStatusFilters, selectedStatus]);
        setSearchFilter({
            ...searchFilter,
            'Report Job Status': newFilters,
        });
        setPage(1);
    };

    const onMyJobsFilterChange = (checked: boolean) => {
        setIsViewingOnlyMyJobs(String(checked));
        setPage(1);
    };

    useInterval(refetch, 10000);

    return (
        <>
            <PageTitle title="Vulnerability reporting - On-demand reports" />
            <PageSection variant="light">
                <Text>
                    Check job status and download on-demand reports in CSV format. Requests are
                    purged according to retention settings.
                </Text>
            </PageSection>
            <PageSection>
                <Card>
                    <CardBody className="pf-v5-u-p-0">
                        <Toolbar>
                            <ToolbarContent>
                                <ToolbarItem alignItems="center">
                                    <ReportJobStatusFilter
                                        availableStatuses={[
                                            'WAITING',
                                            'PREPARING',
                                            'DOWNLOAD_GENERATED',
                                            'ERROR',
                                        ]}
                                        selectedStatuses={ensureReportJobStatuses(
                                            reportJobStatusFilters
                                        )}
                                        onChange={onReportJobStatusFilterChange}
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
                        <OnDemandReportsTable
                            tableState={tableState}
                            getSortParams={getSortParams}
                            onClearFilters={() => {
                                setSearchFilter({});
                                setPage(1);
                            }}
                        />
                    </CardBody>
                </Card>
            </PageSection>
        </>
    );
}

export default OnDemandReportsTab;
