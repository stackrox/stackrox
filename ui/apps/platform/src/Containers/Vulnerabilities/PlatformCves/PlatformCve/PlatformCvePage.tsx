import {
    Breadcrumb,
    BreadcrumbItem,
    Content,
    Divider,
    Flex,
    Label,
    LabelGroup,
    PageSection,
    Pagination,
    Skeleton,
    Split,
    SplitItem,
    Title,
    pluralize,
} from '@patternfly/react-core';
import { useParams } from 'react-router-dom-v5-compat';

import PageTitle from 'Components/PageTitle';
import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';

import useURLPagination from 'hooks/useURLPagination';
import useURLSearch from 'hooks/useURLSearch';
import { getTableUIState } from 'utils/getTableUIState';
import ExternalLink from 'Components/PatternFly/IconText/ExternalLink';
import { DynamicTableLabel } from 'Components/DynamicIcon';
import { getHasSearchApplied } from 'utils/searchUtils';
import useURLSort from 'hooks/useURLSort';
import { getDateTime } from 'utils/dateUtils';
import { createFilterTracker } from 'utils/analyticsEventTracking';
import useAnalytics, { PLATFORM_CVE_FILTER_APPLIED } from 'hooks/useAnalytics';
import { clusterSearchFilterConfig } from '../../searchFilterConfig';
import HeaderLoadingSkeleton from '../../components/HeaderLoadingSkeleton';
import { SummaryCard, SummaryCardLayout } from '../../components/SummaryCardLayout';
import { DEFAULT_VM_PAGE_SIZE } from '../../constants';
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
import usePlatformCveSummaryData from './usePlatformCveSummaryData';

const workloadCveOverviewCvePath = getOverviewPagePath('Platform', {
    entityTab: 'CVE',
});

const searchFilterConfig = [clusterSearchFilterConfig];
function PlatformCvePage() {
    const { analyticsTrack } = useAnalytics();
    const trackAppliedFilter = createFilterTracker(analyticsTrack);

    const { searchFilter, setSearchFilter } = useURLSearch();
    const querySearchFilter = parseQuerySearchFilter(searchFilter);

    const params = useParams() as { cveId: string };
    // CVE ID needs to be decoded here as it will contain the `#` character
    const cveId = decodeURIComponent(params.cveId);

    const query = getRegexScopedQueryString({
        ...querySearchFilter,
        'CVE ID': [cveId],
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
    const metadataRequest = usePlatformCveMetadata(cveId);
    const summaryDataRequest = usePlatformCveSummaryData({ cveId, query });
    const cveMetadata = metadataRequest.data?.platformCVE;
    const cveName = cveMetadata?.cve;
    const isFiltered = getHasSearchApplied(querySearchFilter);

    const tableState = getTableUIState({
        isLoading: affectedClustersRequest.loading,
        error: affectedClustersRequest.error,
        data: clusterData,
        searchFilter: querySearchFilter,
    });

    return (
        <>
            <PageTitle title={`Kubernetes components - Vulnerability ${cveName}`} />
            <PageSection>
                <Breadcrumb>
                    <BreadcrumbItemLink to={workloadCveOverviewCvePath}>
                        Kubernetes components
                    </BreadcrumbItemLink>
                    <BreadcrumbItem isActive>
                        {cveName ?? <Skeleton screenreaderText="Loading CVE name" width="200px" />}
                    </BreadcrumbItem>
                </Breadcrumb>
            </PageSection>
            <PageSection>
                {cveMetadata ? (
                    <Flex
                        direction={{ default: 'column' }}
                        alignItems={{ default: 'alignItemsFlexStart' }}
                        spaceItems={{ default: 'spaceItemsSm' }}
                    >
                        <Title headingLevel="h1">{cveMetadata.cve}</Title>
                        {cveMetadata.firstDiscoveredTime && (
                            <LabelGroup numLabels={1}>
                                <Label>
                                    First discovered in system:{' '}
                                    {getDateTime(cveMetadata.firstDiscoveredTime)}
                                </Label>
                            </LabelGroup>
                        )}
                        <Content component="p">{cveMetadata.clusterVulnerability.summary}</Content>
                        <ExternalLink>
                            <a
                                href={cveMetadata.clusterVulnerability.link}
                                target="_blank"
                                rel="noopener noreferrer"
                            >
                                {cveMetadata.clusterVulnerability.link}
                            </a>
                        </ExternalLink>
                    </Flex>
                ) : (
                    <HeaderLoadingSkeleton
                        nameScreenreaderText="Loading CVE name"
                        metadataScreenreaderText="Loading CVE metadata"
                    />
                )}
            </PageSection>
            <Divider component="div" />
            <PageSection hasBodyWrapper={false} isFilled>
                <AdvancedFiltersToolbar
                    searchFilter={searchFilter}
                    searchFilterConfig={searchFilterConfig}
                    cveStatusFilterField="CLUSTER CVE FIXABLE"
                    onFilterChange={(newFilter, searchPayload) => {
                        setSearchFilter(newFilter);
                        trackAppliedFilter(PLATFORM_CVE_FILTER_APPLIED, searchPayload);
                    }}
                    includeCveSeverityFilters={false}
                />
                <SummaryCardLayout
                    error={summaryDataRequest.error}
                    isLoading={summaryDataRequest.loading}
                >
                    <SummaryCard
                        data={summaryDataRequest.data}
                        loadingText="Loading affected nodes summary"
                        renderer={({ data }) => (
                            <AffectedClustersSummaryCard
                                affectedClusterCount={data.clusterCount}
                                totalClusterCount={data.totalClusterCount}
                            />
                        )}
                    />
                    <SummaryCard
                        data={summaryDataRequest.data}
                        loadingText="Loading affected nodes by CVE severity summary"
                        renderer={({ data }) => (
                            <ClustersByTypeSummaryCard
                                clusterCounts={data.platformCVE?.clusterCountByType}
                            />
                        )}
                    />
                </SummaryCardLayout>
                <Divider component="div" />
                <Split hasGutter className="pf-v6-u-align-items-baseline">
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
                        setPage(1);
                    }}
                />
            </PageSection>
        </>
    );
}

export default PlatformCvePage;
