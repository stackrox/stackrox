import { useEffect } from 'react';
import {
    Alert,
    Card,
    CardBody,
    Divider,
    Flex,
    FlexItem,
    PageSection,
    Title,
} from '@patternfly/react-core';

import PageTitle from 'Components/PageTitle';
import ExternalLink from 'Components/PatternFly/IconText/ExternalLink';
import useFeatureFlags from 'hooks/useFeatureFlags';
import useMap from 'hooks/useMap';
import useMetadata from 'hooks/useMetadata';
import useURLStringUnion from 'hooks/useURLStringUnion';
import useURLPagination from 'hooks/useURLPagination';
import useURLSearch from 'hooks/useURLSearch';
import useURLSort from 'hooks/useURLSort';
import useAnalytics, {
    NODE_CVE_ENTITY_CONTEXT_VIEWED,
    NODE_CVE_FILTER_APPLIED,
} from 'hooks/useAnalytics';
import { getHasSearchApplied } from 'utils/searchUtils';
import { getVersionedDocs } from 'utils/versioning';
import { createFilterTracker } from 'utils/analyticsEventTracking';

import {
    clusterSearchFilterConfig,
    nodeComponentSearchFilterConfig,
    nodeCVESearchFilterConfig,
    nodeSearchFilterConfig,
} from '../../searchFilterConfig';
import AdvancedFiltersToolbar from '../../components/AdvancedFiltersToolbar';
import TableEntityToolbar from '../../components/TableEntityToolbar';
import EntityTypeToggleGroup from '../../components/EntityTypeToggleGroup';
import { nodeEntityTabValues } from '../../types';
import { DEFAULT_VM_PAGE_SIZE } from '../../constants';
import { parseQuerySearchFilter } from '../../utils/searchUtils';

import CVEsTable, {
    defaultSortOption as cveDefaultSortOption,
    sortFields as cveSortFields,
} from './CVEsTable';
import NodesTable, {
    defaultSortOption as nodeDefaultSortOption,
    sortFields as nodeSortFields,
} from './NodesTable';
import { useNodeCveEntityCounts } from './useNodeCveEntityCounts';

const searchFilterConfig = [
    nodeSearchFilterConfig,
    nodeCVESearchFilterConfig,
    nodeComponentSearchFilterConfig,
    clusterSearchFilterConfig,
];

function NodeCvesOverviewPage() {
    const { analyticsTrack } = useAnalytics();
    const trackAppliedFilter = createFilterTracker(analyticsTrack);
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const scannerV4NodeScanResultsPossible =
        isFeatureFlagEnabled('ROX_SCANNER_V4') && isFeatureFlagEnabled('ROX_NODE_INDEX_ENABLED');

    const [activeEntityTabKey] = useURLStringUnion('entityTab', nodeEntityTabValues);
    const { searchFilter, setSearchFilter } = useURLSearch();
    const pagination = useURLPagination(DEFAULT_VM_PAGE_SIZE);
    const { sortOption, getSortParams, setSortOption } = useURLSort({
        sortFields: activeEntityTabKey === 'CVE' ? cveSortFields : nodeSortFields,
        defaultSortOption:
            activeEntityTabKey === 'CVE' ? cveDefaultSortOption : nodeDefaultSortOption,
        onSort: () => pagination.setPage(1),
    });

    const querySearchFilter = parseQuerySearchFilter(searchFilter);
    const isFiltered = getHasSearchApplied(querySearchFilter);

    const selectedCves = useMap<string, { cve: string }>();
    const { version } = useMetadata();

    function onEntityTabChange(entityTab: 'CVE' | 'Node') {
        pagination.setPage(1);
        setSortOption(entityTab === 'CVE' ? cveDefaultSortOption : nodeDefaultSortOption);

        analyticsTrack({
            event: NODE_CVE_ENTITY_CONTEXT_VIEWED,
            properties: {
                type: entityTab,
                page: 'Overview',
            },
        });
    }

    // Track the current entity tab when the page is initially visited.
    /* eslint-disable react-hooks/exhaustive-deps */
    useEffect(() => {
        onEntityTabChange(activeEntityTabKey);
    }, []);
    // activeEntityTabKey
    // onEntityTabChange
    /* eslint-enable react-hooks/exhaustive-deps */

    function onClearFilters() {
        setSearchFilter({});
        pagination.setPage(1);
    }

    const { data } = useNodeCveEntityCounts(querySearchFilter);

    const entityCounts = {
        CVE: data?.nodeCVECount ?? 0,
        Node: data?.nodeCount ?? 0,
    };

    const filterToolbar = (
        <AdvancedFiltersToolbar
            searchFilter={searchFilter}
            searchFilterConfig={searchFilterConfig}
            onFilterChange={(newFilter, searchPayload) => {
                setSearchFilter(newFilter);
                pagination.setPage(1);
                trackAppliedFilter(NODE_CVE_FILTER_APPLIED, searchPayload);
            }}
        />
    );

    const entityToggleGroup = (
        <EntityTypeToggleGroup
            entityTabs={['CVE', 'Node']}
            entityCounts={entityCounts}
            onChange={onEntityTabChange}
        />
    );

    return (
        <>
            <PageTitle title="Node CVEs Overview" />
            <Divider component="div" />
            <PageSection
                className="pf-v5-u-display-flex pf-v5-u-flex-direction-row pf-v5-u-align-items-center"
                variant="light"
            >
                <Flex alignItems={{ default: 'alignItemsCenter' }} className="pf-v5-u-flex-grow-1">
                    <Flex direction={{ default: 'column' }} className="pf-v5-u-flex-grow-1">
                        <Title headingLevel="h1">Node CVEs</Title>
                        <FlexItem>Prioritize and manage scanned CVEs across nodes</FlexItem>
                    </Flex>
                </Flex>
            </PageSection>
            {scannerV4NodeScanResultsPossible && (
                <PageSection variant="light" className="pf-v5-u-pt-0">
                    <Alert
                        isInline
                        variant="info"
                        title="Results may include Node CVEs obtained from Scanner V4"
                        component="p"
                    >
                        <ExternalLink>
                            <a
                                href={getVersionedDocs(
                                    version,
                                    'operating/managing-vulnerabilities#understanding-node-cves-scanner-v4_scan-rhcos-node-host'
                                )}
                                target="_blank"
                                rel="noopener noreferrer"
                            >
                                Read more about the differences between the node scanning results
                                obtained with the StackRox Scanner and Scanner V4.
                            </a>
                        </ExternalLink>
                    </Alert>
                </PageSection>
            )}
            <PageSection padding={{ default: 'noPadding' }}>
                <PageSection isCenterAligned>
                    <Card>
                        <CardBody>
                            <TableEntityToolbar
                                filterToolbar={filterToolbar}
                                entityToggleGroup={entityToggleGroup}
                                pagination={pagination}
                                tableRowCount={
                                    activeEntityTabKey === 'CVE'
                                        ? entityCounts.CVE
                                        : entityCounts.Node
                                }
                                isFiltered={isFiltered}
                            />
                            <Divider component="div" />
                            {activeEntityTabKey === 'CVE' && (
                                <CVEsTable
                                    querySearchFilter={querySearchFilter}
                                    isFiltered={isFiltered}
                                    pagination={pagination}
                                    selectedCves={selectedCves}
                                    canSelectRows={false}
                                    createRowActions={() => []}
                                    sortOption={sortOption}
                                    getSortParams={getSortParams}
                                    onClearFilters={onClearFilters}
                                />
                            )}
                            {activeEntityTabKey === 'Node' && (
                                <NodesTable
                                    querySearchFilter={querySearchFilter}
                                    isFiltered={isFiltered}
                                    pagination={pagination}
                                    sortOption={sortOption}
                                    getSortParams={getSortParams}
                                    onClearFilters={onClearFilters}
                                />
                            )}
                        </CardBody>
                    </Card>
                </PageSection>
            </PageSection>
        </>
    );
}

export default NodeCvesOverviewPage;
