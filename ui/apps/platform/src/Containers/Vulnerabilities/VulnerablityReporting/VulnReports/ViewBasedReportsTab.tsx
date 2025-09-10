import React, { useCallback, useMemo } from 'react';
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
import { toggleItemInArray } from 'utils/arrayUtils';
import useRestQuery from 'hooks/useRestQuery';
import useURLPagination from 'hooks/useURLPagination';
import useURLSort from 'hooks/useURLSort';
import useURLSearch from 'hooks/useURLSearch';
import useURLStringUnion from 'hooks/useURLStringUnion';
import { fetchViewBasedReportHistory } from 'services/ReportsService';
import PageTitle from 'Components/PageTitle';
import ReportJobStatusFilter, {
    ensureReportJobStatuses,
} from 'Components/ReportJob/ReportJobStatusFilter';
import MyJobsFilter from 'Components/ReportJob/MyJobsFilter';
import { ReportJobStatus } from 'Components/ReportJob/types';
import useAnalytics, { VIEW_BASED_REPORT_TABLE_INTERACTION } from 'hooks/useAnalytics';
import ViewBasedReportsTable from './ViewBasedReportsTable';

const sortOptions = {
    sortFields: ['Report Completion Time'],
    defaultSortOption: { field: 'Report Completion Time', direction: 'desc' } as const,
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

function ViewBasedReportsTab() {
    const { analyticsTrack } = useAnalytics();
    const { page, perPage, setPage, setPerPage } = useURLPagination(10);
    const { sortOption, getSortParams } = useURLSort(sortOptions);
    const { searchFilter, setSearchFilter } = useURLSearch();
    const [isViewingOnlyMyJobs, setIsViewingOnlyMyJobs] = useURLStringUnion('viewOnlyMyJobs', [
        'false',
        'true',
    ]);

    const reportJobStatusFilters = useMemo(() => {
        return ensureStringArray(searchFilter['Report Job Status']);
    }, [searchFilter]);

    const fetchViewBasedReportsHistoryCallback = useCallback(() => {
        const modifiedSearchFilter = {
            ...createQueryFromReportJobStatusFilters(reportJobStatusFilters),
        };

        return fetchViewBasedReportHistory({
            searchFilter: modifiedSearchFilter,
            page,
            perPage,
            sortOption,
            showMyHistory: isViewingOnlyMyJobs === 'true',
        });
    }, [reportJobStatusFilters, page, perPage, sortOption, isViewingOnlyMyJobs]);

    const { data, isLoading, error, refetch } = useRestQuery(fetchViewBasedReportsHistoryCallback, {
        clearErrorBeforeRequest: false,
    });

    const tableState = getTableUIState({
        isLoading,
        data,
        error,
        searchFilter,
        isPolling: true,
    });

    const onReportJobStatusFilterChange = (_checked: boolean, selectedStatus: ReportJobStatus) => {
        const newFilters = toggleItemInArray(
            reportJobStatusFilters,
            selectedStatus,
            (a, b) => a === b
        );
        setSearchFilter({
            ...searchFilter,
            'Report Job Status': ensureReportJobStatuses(newFilters),
        });
        setPage(1);

        // Track filter interaction with complete filter state
        analyticsTrack({
            event: VIEW_BASED_REPORT_TABLE_INTERACTION,
            properties: {
                action: 'filter',
                filterType: 'Report Job Status',
                filterValue: newFilters,
            },
        });
    };

    const onMyJobsFilterChange = (checked: boolean) => {
        setIsViewingOnlyMyJobs(String(checked));
        setPage(1);

        // Track filter interaction with filter value
        analyticsTrack({
            event: VIEW_BASED_REPORT_TABLE_INTERACTION,
            properties: {
                action: 'filter',
                filterType: 'My Jobs',
                filterValue: String(checked),
            },
        });
    };

    useInterval(refetch, 10000);

    return (
        <>
            <PageTitle title="Vulnerability reporting - View-based reports" />
            <PageSection variant="light">
                <Text>
                    Check job status and download view-based reports in CSV format. Requests are
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
                                    {/* TODO: Change this to determinate pagination pattern */}
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
                        <ViewBasedReportsTable
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

export default ViewBasedReportsTab;
