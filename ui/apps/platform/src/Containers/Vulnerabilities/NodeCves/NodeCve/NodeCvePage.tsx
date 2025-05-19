import React from 'react';
import {
    Breadcrumb,
    BreadcrumbItem,
    Divider,
    Flex,
    PageSection,
    Pagination,
    Skeleton,
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
import { getHasSearchApplied } from 'utils/searchUtils';
import { DynamicTableLabel } from 'Components/DynamicIcon';

import {
    SummaryCardLayout,
    SummaryCard,
} from 'Containers/Vulnerabilities/components/SummaryCardLayout';
import useURLSort from 'hooks/useURLSort';
import { createFilterTracker } from 'utils/analyticsEventTracking';
import useAnalytics, { NODE_CVE_FILTER_APPLIED } from 'hooks/useAnalytics';
import AdvancedFiltersToolbar from '../../components/AdvancedFiltersToolbar';
import BySeveritySummaryCard from '../../components/BySeveritySummaryCard';
import {
    getHiddenSeverities,
    getOverviewPagePath,
    getRegexScopedQueryString,
    parseQuerySearchFilter,
} from '../../utils/searchUtils';
import CvePageHeader from '../../components/CvePageHeader';
import {
    nodeSearchFilterConfig,
    nodeComponentSearchFilterConfig,
    clusterSearchFilterConfig,
} from '../../searchFilterConfig';
import { DEFAULT_VM_PAGE_SIZE } from '../../constants';
import AffectedNodesTable, { defaultSortOption, sortFields } from './AffectedNodesTable';
import AffectedNodesSummaryCard from './AffectedNodesSummaryCard';
import useAffectedNodes from './useAffectedNodes';
import useNodeCveMetadata from './useNodeCveMetadata';
import useNodeCveSummaryData from './useNodeCveSummaryData';

const nodeCveOverviewCvePath = getOverviewPagePath('Node', { entityTab: 'CVE' });

const searchFilterConfig = [
    nodeSearchFilterConfig,
    nodeComponentSearchFilterConfig,
    clusterSearchFilterConfig,
];

const defaultNodeCveSummary = {
    affectedNodeCountBySeverity: {
        critical: { total: 0 },
        important: { total: 0 },
        moderate: { total: 0 },
        low: { total: 0 },
        unknown: { total: 0 },
    },
    distroTuples: [],
};

function NodeCvePage() {
    const { analyticsTrack } = useAnalytics();
    const trackAppliedFilter = createFilterTracker(analyticsTrack);

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
    const isFiltered = getHasSearchApplied(querySearchFilter);
    const hiddenSeverities = getHiddenSeverities(querySearchFilter);

    const { metadataRequest, cveData: cveMetadata } = useNodeCveMetadata(cveId);
    const { summaryDataRequest, nodeCount } = useNodeCveSummaryData(cveId, query);

    const { affectedNodesRequest, nodeData } = useAffectedNodes({
        query,
        page,
        perPage,
        sortOption,
    });

    const nodeCveName = cveMetadata?.cve;

    const tableState = getTableUIState({
        isLoading: affectedNodesRequest.loading,
        error: affectedNodesRequest.error,
        data: nodeData,
        searchFilter: querySearchFilter,
    });

    return (
        <>
            <PageTitle title={`Node CVEs - NodeCVE ${nodeCveName}`} />
            <PageSection variant="light" className="pf-v5-u-py-md">
                <Breadcrumb>
                    <BreadcrumbItemLink to={nodeCveOverviewCvePath}>Node CVEs</BreadcrumbItemLink>
                    <BreadcrumbItem isActive>
                        {nodeCveName ?? (
                            <Skeleton screenreaderText="Loading CVE name" width="200px" />
                        )}
                    </BreadcrumbItem>
                </Breadcrumb>
            </PageSection>
            <Divider component="div" />
            <PageSection variant="light">
                <CvePageHeader data={cveMetadata} />
            </PageSection>
            <Divider component="div" />
            <PageSection className="pf-v5-u-flex-grow-1">
                <AdvancedFiltersToolbar
                    className="pf-v5-u-pt-lg pf-v5-u-pb-0 pf-v5-u-px-sm"
                    searchFilter={searchFilter}
                    searchFilterConfig={searchFilterConfig}
                    onFilterChange={(newFilter, searchPayload) => {
                        setSearchFilter(newFilter);
                        setPage(1, 'replace');
                        trackAppliedFilter(NODE_CVE_FILTER_APPLIED, searchPayload);
                    }}
                />
                <SummaryCardLayout
                    error={metadataRequest.error}
                    isLoading={metadataRequest.loading}
                >
                    <SummaryCard
                        data={summaryDataRequest.data}
                        loadingText="Loading affected nodes summary"
                        renderer={({ data }) => (
                            <AffectedNodesSummaryCard
                                affectedNodeCount={nodeCount}
                                totalNodeCount={data.totalNodeCount}
                                operatingSystemCount={
                                    (data.nodeCVE ?? defaultNodeCveSummary).distroTuples.length
                                }
                            />
                        )}
                    />
                    <SummaryCard
                        data={summaryDataRequest.data}
                        loadingText="Loading affected nodes by CVE severity summary"
                        renderer={({ data }) => (
                            <BySeveritySummaryCard
                                title="Nodes by severity"
                                severityCounts={
                                    (data.nodeCVE ?? defaultNodeCveSummary)
                                        .affectedNodeCountBySeverity
                                }
                                hiddenSeverities={hiddenSeverities}
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
                                    {pluralize(nodeCount, 'node')} affected
                                </Title>
                                {isFiltered && <DynamicTableLabel />}
                            </Flex>
                        </SplitItem>
                        <SplitItem>
                            <Pagination
                                itemCount={nodeCount}
                                perPage={perPage}
                                page={page}
                                onSetPage={(_, newPage) => setPage(newPage)}
                                onPerPageSelect={(_, newPerPage) => {
                                    setPerPage(newPerPage);
                                }}
                            />
                        </SplitItem>
                    </Split>
                    <AffectedNodesTable
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

export default NodeCvePage;
