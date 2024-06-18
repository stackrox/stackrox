import React from 'react';
import {
    PageSection,
    Breadcrumb,
    Divider,
    BreadcrumbItem,
    Skeleton,
    Flex,
    Pagination,
    Split,
    SplitItem,
    Title,
    pluralize,
} from '@patternfly/react-core';
import { useParams } from 'react-router-dom';

import PageTitle from 'Components/PageTitle';
import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';

import useURLPagination from 'hooks/useURLPagination';
import useURLSearch from 'hooks/useURLSearch';
import { getTableUIState } from 'utils/getTableUIState';
import {
    SummaryCardLayout,
    SummaryCard,
} from 'Containers/Vulnerabilities/components/SummaryCardLayout';
import { DynamicTableLabel } from 'Components/DynamicIcon';
import { getHasSearchApplied } from 'utils/searchUtils';
import useURLSort from 'hooks/useURLSort';
import { DEFAULT_VM_PAGE_SIZE } from '../../constants';
import CvePageHeader from '../../components/CvePageHeader';
import {
    getOverviewPagePath,
    getRegexScopedQueryString,
    parseQuerySearchFilter,
} from '../../utils/searchUtils';
import useAffectedClusters from './useAffectedClusters';
import AffectedClustersTable, { sortFields, defaultSortOption } from './AffectedClustersTable';
import usePlatformCveMetadata from './usePlatformCveMetadata';
import ClustersByTypeSummaryCard from './ClustersByTypeSummaryCard';
import AffectedClustersSummaryCard from './AffectedClustersSummaryCard';
import AdvancedFiltersToolbar from '../../components/AdvancedFiltersToolbar';
import { clusterSearchFilterConfig } from '../../searchFilterConfig';

const workloadCveOverviewCvePath = getOverviewPagePath('Platform', {
    entityTab: 'CVE',
});

const searchFilterConfig = {
    Cluster: clusterSearchFilterConfig,
};

function PlatformCvePage() {
    const { searchFilter, setSearchFilter } = useURLSearch();
    const querySearchFilter = parseQuerySearchFilter(searchFilter);

    // We need to scope all queries to the *exact* CVE name so that we don't accidentally get
    // data that matches a prefix of the CVE name in the nested fields
    const { cveId } = useParams() as { cveId: string };
    const exactCveIdSearchRegex = `^${cveId}$`;
    const query = getRegexScopedQueryString({
        ...querySearchFilter,
        CVE: [exactCveIdSearchRegex],
    });

    const { page, perPage, setPage, setPerPage } = useURLPagination(DEFAULT_VM_PAGE_SIZE);
    const { sortOption, getSortParams } = useURLSort({
        sortFields,
        defaultSortOption,
        onSort: () => setPage(1),
    });

    const { affectedClustersRequest, clusterData, clusterCount } = useAffectedClusters({
        query,
        page,
        perPage,
        sortOption,
    });
    const metadataRequest = usePlatformCveMetadata({ cve: cveId, query, page, perPage });
    const cveName = metadataRequest.data?.platformCVE?.cve;
    const isFiltered = getHasSearchApplied(querySearchFilter);

    const tableState = getTableUIState({
        isLoading: affectedClustersRequest.loading,
        error: affectedClustersRequest.error,
        data: clusterData,
        searchFilter: querySearchFilter,
    });

    return (
        <>
            <PageTitle title={`Platform CVEs - PlatformCVE ${cveName}`} />
            <PageSection variant="light" className="pf-v5-u-py-md">
                <Breadcrumb>
                    <BreadcrumbItemLink to={workloadCveOverviewCvePath}>
                        Platform CVEs
                    </BreadcrumbItemLink>
                    <BreadcrumbItem isActive>
                        {cveName ?? <Skeleton screenreaderText="Loading CVE name" width="200px" />}
                    </BreadcrumbItem>
                </Breadcrumb>
            </PageSection>
            <Divider component="div" />
            <PageSection variant="light">
                <CvePageHeader data={metadataRequest.data?.platformCVE} />
            </PageSection>
            <Divider component="div" />
            <PageSection className="pf-v5-u-flex-grow-1">
                <AdvancedFiltersToolbar
                    className="pf-v5-u-pb-0 pf-v5-u-px-sm"
                    searchFilter={searchFilter}
                    searchFilterConfig={searchFilterConfig}
                    cveStatusFilterField="CLUSTER CVE FIXABLE"
                    onFilterChange={(newFilter, { action }) => {
                        setSearchFilter(newFilter);

                        if (action === 'ADD') {
                            // TODO - Add analytics tracking ROX-24509
                        }
                    }}
                    includeCveSeverityFilters={false}
                />
                <SummaryCardLayout
                    error={metadataRequest.error}
                    isLoading={metadataRequest.loading}
                >
                    <SummaryCard
                        data={metadataRequest.data}
                        loadingText="Loading affected nodes summary"
                        renderer={({ data }) => (
                            <AffectedClustersSummaryCard
                                affectedClusterCount={data.clusterCount}
                                totalClusterCount={data.totalClusterCount}
                            />
                        )}
                    />
                    <SummaryCard
                        data={metadataRequest.data}
                        loadingText="Loading affected nodes by CVE severity summary"
                        renderer={({ data }) => (
                            <ClustersByTypeSummaryCard
                                clusterCounts={data.platformCVE.clusterCountByType}
                            />
                        )}
                    />
                </SummaryCardLayout>
                <Divider component="div" />
                <div className="pf-v5-u-background-color-100 pf-v5-u-flex-grow-1 pf-v5-u-p-lg">
                    <Split className="pf-v5-u-pb-lg pf-v5-u-align-items-baseline">
                        <SplitItem isFilled>
                            <Flex alignItems={{ default: 'alignItemsCenter' }}>
                                <Title headingLevel="h2">
                                    {pluralize(clusterCount, 'cluster')} affected
                                </Title>
                                {isFiltered && <DynamicTableLabel />}
                            </Flex>
                        </SplitItem>
                        <SplitItem>
                            <Pagination
                                itemCount={clusterCount}
                                perPage={perPage}
                                page={page}
                                onSetPage={(_, newPage) => setPage(newPage)}
                                onPerPageSelect={(_, newPerPage) => {
                                    setPerPage(newPerPage);
                                }}
                            />
                        </SplitItem>
                    </Split>
                    <AffectedClustersTable
                        tableState={tableState}
                        getSortParams={getSortParams}
                        onClearFilters={() => {
                            setSearchFilter({});
                            setPage(1, 'replace');
                        }}
                    />
                </div>
            </PageSection>
        </>
    );
}

export default PlatformCvePage;
