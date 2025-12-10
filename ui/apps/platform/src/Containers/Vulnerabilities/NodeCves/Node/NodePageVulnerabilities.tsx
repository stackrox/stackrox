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

import { DynamicTableLabel } from 'Components/DynamicIcon';
import useURLPagination from 'hooks/useURLPagination';
import useURLSearch from 'hooks/useURLSearch';
import useURLSort from 'hooks/useURLSort';
import { getTableUIState } from 'utils/getTableUIState';
import { getHasSearchApplied } from 'utils/searchUtils';
import useAnalytics, { NODE_CVE_FILTER_APPLIED } from 'hooks/useAnalytics';
import { createFilterTracker } from 'utils/analyticsEventTracking';

import BySeveritySummaryCard from '../../components/BySeveritySummaryCard';
import {
    nodeCVESearchFilterConfig,
    nodeComponentSearchFilterConfig,
} from '../../searchFilterConfig';

import {
    getHiddenSeverities,
    getHiddenStatuses,
    getRegexScopedQueryString,
    parseQuerySearchFilter,
} from '../../utils/searchUtils';
import CVEsTable, { sortFields, defaultSortOption } from './CVEsTable';
import useNodeVulnerabilities from './useNodeVulnerabilities';
import useNodeSummaryData from './useNodeSummaryData';
import { DEFAULT_VM_PAGE_SIZE } from '../../constants';
import { SummaryCard, SummaryCardLayout } from '../../components/SummaryCardLayout';
import AdvancedFiltersToolbar from '../../components/AdvancedFiltersToolbar';
import CvesByStatusSummaryCard from '../../components/CvesByStatusSummaryCard';

const searchFilterConfig = [nodeCVESearchFilterConfig, nodeComponentSearchFilterConfig];

export type NodePageVulnerabilitiesProps = {
    nodeId: string;
};

function NodePageVulnerabilities({ nodeId }: NodePageVulnerabilitiesProps) {
    const { analyticsTrack } = useAnalytics();
    const trackAppliedFilter = createFilterTracker(analyticsTrack);
    const { searchFilter, setSearchFilter } = useURLSearch();

    const querySearchFilter = parseQuerySearchFilter(searchFilter);
    const query = getRegexScopedQueryString(querySearchFilter);
    const isFiltered = getHasSearchApplied(querySearchFilter);
    const { page, perPage, setPage, setPerPage } = useURLPagination(DEFAULT_VM_PAGE_SIZE);
    const { sortOption, getSortParams } = useURLSort({
        sortFields,
        defaultSortOption,
        onSort: () => setPage(1, 'replace'),
    });
    const hiddenSeverities = getHiddenSeverities(querySearchFilter);
    const hiddenStatuses = getHiddenStatuses(querySearchFilter);

    const { data, loading, error } = useNodeVulnerabilities({
        nodeId,
        query,
        page,
        perPage,
        sortOption,
    });
    const summaryRequest = useNodeSummaryData(nodeId, query);

    const nodeCount = data?.node?.nodeVulnerabilityCount ?? 0;

    const tableState = getTableUIState({
        isLoading: loading,
        error,
        data: data?.node?.nodeVulnerabilities,
        searchFilter: querySearchFilter,
    });

    return (
        <>
            <PageSection component="div" variant="light" className="pf-v5-u-py-md pf-v5-u-px-xl">
                <Text>Review and triage vulnerability data scanned on this node</Text>
            </PageSection>
            <PageSection isFilled className="pf-v5-u-display-flex pf-v5-u-flex-direction-column">
                <AdvancedFiltersToolbar
                    className="pf-v5-u-px-sm pf-v5-u-pb-0"
                    searchFilter={searchFilter}
                    searchFilterConfig={searchFilterConfig}
                    onFilterChange={(newFilter, searchPayload) => {
                        setSearchFilter(newFilter);
                        setPage(1, 'replace');
                        trackAppliedFilter(NODE_CVE_FILTER_APPLIED, searchPayload);
                    }}
                />
                <SummaryCardLayout isLoading={summaryRequest.loading} error={summaryRequest.error}>
                    <SummaryCard
                        loadingText={'Loading node CVEs by severity summary'}
                        data={summaryRequest.data}
                        renderer={({ data }) => (
                            <BySeveritySummaryCard
                                title="CVEs by severity"
                                severityCounts={data.node.nodeCVECountBySeverity}
                                hiddenSeverities={hiddenSeverities}
                            />
                        )}
                    />
                    <SummaryCard
                        loadingText={'Loading node CVEs by status summary'}
                        data={summaryRequest.data}
                        renderer={({ data }) => (
                            <CvesByStatusSummaryCard
                                cveStatusCounts={data.node.nodeCVECountBySeverity}
                                hiddenStatuses={hiddenStatuses}
                            />
                        )}
                    />
                </SummaryCardLayout>
                <Divider component="div" />
                <div className="pf-v5-u-flex-grow-1 pf-v5-u-background-color-100 pf-v5-u-p-lg">
                    <Split className="pf-v5-u-pb-lg pf-v5-u-align-items-baseline">
                        <SplitItem isFilled>
                            <Flex alignItems={{ default: 'alignItemsCenter' }}>
                                <Title headingLevel="h2">
                                    {data && data.node ? (
                                        `${pluralize(
                                            data.node.nodeVulnerabilityCount,
                                            'result'
                                        )} found`
                                    ) : (
                                        <Skeleton screenreaderText="Loading node vulnerability count" />
                                    )}
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

export default NodePageVulnerabilities;
