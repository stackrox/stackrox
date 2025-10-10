import React from 'react';
import {
    Divider,
    Flex,
    PageSection,
    Pagination,
    Skeleton,
    Split,
    SplitItem,
    Text,
    Title,
    pluralize,
} from '@patternfly/react-core';

import useURLSearch from 'hooks/useURLSearch';
import useURLPagination from 'hooks/useURLPagination';
import { getHasSearchApplied, getRequestQueryStringForSearchFilter } from 'utils/searchUtils';
import { getTableUIState } from 'utils/getTableUIState';

import { DynamicTableLabel } from 'Components/DynamicIcon';
import useURLSort from 'hooks/useURLSort';
import { createFilterTracker } from 'utils/analyticsEventTracking';
import useAnalytics, { PLATFORM_CVE_FILTER_APPLIED } from 'hooks/useAnalytics';
import { platformCVESearchFilterConfig } from '../../searchFilterConfig';
import { SummaryCardLayout, SummaryCard } from '../../components/SummaryCardLayout';
import { getHiddenStatuses, parseQuerySearchFilter } from '../../utils/searchUtils';
import { DEFAULT_VM_PAGE_SIZE } from '../../constants';
import AdvancedFiltersToolbar from '../../components/AdvancedFiltersToolbar';

import useClusterVulnerabilities from './useClusterVulnerabilities';
import useClusterSummaryData from './useClusterSummaryData';
import CVEsTable, { defaultSortOption, sortFields } from './CVEsTable';
import PlatformCvesByStatusSummaryCard from './PlatformCvesByStatusSummaryCard';
import PlatformCvesByTypeSummaryCard from './PlatformCvesByTypeSummaryCard';

const searchFilterConfig = [platformCVESearchFilterConfig];

export type ClusterPageVulnerabilitiesProps = {
    clusterId: string;
};

function ClusterPageVulnerabilities({ clusterId }: ClusterPageVulnerabilitiesProps) {
    const { analyticsTrack } = useAnalytics();
    const trackAppliedFilter = createFilterTracker(analyticsTrack);

    const { searchFilter, setSearchFilter } = useURLSearch();
    const querySearchFilter = parseQuerySearchFilter(searchFilter);
    const query = getRequestQueryStringForSearchFilter(querySearchFilter);
    const isFiltered = getHasSearchApplied(querySearchFilter);
    const { page, perPage, setPage, setPerPage } = useURLPagination(DEFAULT_VM_PAGE_SIZE);
    const { sortOption, getSortParams } = useURLSort({
        sortFields,
        defaultSortOption,
        onSort: () => setPage(1),
    });

    const { data, loading, error } = useClusterVulnerabilities({
        clusterId,
        query,
        page,
        perPage,
        sortOption,
    });

    const summaryRequest = useClusterSummaryData(clusterId, query);

    const hiddenStatuses = getHiddenStatuses(querySearchFilter);
    const clusterVulnerabilityCount = data?.cluster?.clusterVulnerabilityCount ?? 0;

    const tableState = getTableUIState({
        isLoading: loading,
        error,
        data: data?.cluster?.clusterVulnerabilities,
        searchFilter: querySearchFilter,
    });

    return (
        <>
            <PageSection component="div" variant="light" className="pf-v5-u-py-md pf-v5-u-px-xl">
                <Text>Review and triage vulnerability data scanned on this cluster</Text>
            </PageSection>
            <PageSection isFilled className="pf-v5-u-display-flex pf-v5-u-flex-direction-column">
                <AdvancedFiltersToolbar
                    className="pf-v5-u-pb-0 pf-v5-u-px-sm"
                    searchFilter={searchFilter}
                    searchFilterConfig={searchFilterConfig}
                    cveStatusFilterField="CLUSTER CVE FIXABLE"
                    onFilterChange={(newFilter, searchPayload) => {
                        setSearchFilter(newFilter);
                        trackAppliedFilter(PLATFORM_CVE_FILTER_APPLIED, searchPayload);
                    }}
                    includeCveSeverityFilters={false}
                />
                <SummaryCardLayout isLoading={summaryRequest.loading} error={summaryRequest.error}>
                    <SummaryCard
                        loadingText={'Loading platform CVEs by status summary'}
                        data={summaryRequest.data}
                        renderer={({ data }) => (
                            <PlatformCvesByStatusSummaryCard
                                data={data.cluster.platformCVECountByFixability}
                                hiddenStatuses={hiddenStatuses}
                            />
                        )}
                    />
                    <SummaryCard
                        loadingText={'Loading platform CVEs by type summary'}
                        data={summaryRequest.data}
                        renderer={({ data }) => (
                            <PlatformCvesByTypeSummaryCard
                                data={data.cluster.platformCVECountByType}
                            />
                        )}
                    />
                </SummaryCardLayout>
                <Divider component="div" />
                <div className="pf-v5-u-flex-grow-1 pf-v5-u-background-color-100 pf-v5-u-p-lg">
                    <Split className="pf-v5-u-pb-lg pf-v5-u-align-items-baseline">
                        <SplitItem isFilled>
                            <Flex alignItems={{ default: 'alignItemsCenter' }}>
                                <Title headingLevel="h2" className="pf-v5-u-w-50">
                                    {data ? (
                                        `${pluralize(clusterVulnerabilityCount, 'result')} found`
                                    ) : (
                                        <Skeleton screenreaderText="Loading cluster vulnerability count" />
                                    )}
                                </Title>
                                {isFiltered && <DynamicTableLabel />}
                            </Flex>
                        </SplitItem>
                        <SplitItem>
                            <Pagination
                                itemCount={clusterVulnerabilityCount}
                                perPage={perPage}
                                page={page}
                                onSetPage={(_, newPage) => setPage(newPage)}
                                onPerPageSelect={(_, newPerPage) => {
                                    setPerPage(newPerPage);
                                }}
                            />
                        </SplitItem>
                    </Split>
                    <CVEsTable
                        tableState={tableState}
                        getSortParams={getSortParams}
                        onClearFilters={() => {
                            setSearchFilter({});
                            setPage(1);
                        }}
                    />
                </div>
            </PageSection>
        </>
    );
}

export default ClusterPageVulnerabilities;
