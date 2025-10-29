import { useEffect } from 'react';
import {
    PageSection,
    Title,
    Divider,
    Flex,
    FlexItem,
    Card,
    CardBody,
} from '@patternfly/react-core';
import { useApolloClient } from '@apollo/client';

import PageTitle from 'Components/PageTitle';
import useURLStringUnion from 'hooks/useURLStringUnion';
import useURLPagination from 'hooks/useURLPagination';
import useURLSearch from 'hooks/useURLSearch';
import useAnalytics, {
    PLATFORM_CVE_ENTITY_CONTEXT_VIEWED,
    PLATFORM_CVE_FILTER_APPLIED,
} from 'hooks/useAnalytics';
import { getHasSearchApplied } from 'utils/searchUtils';

import useMap from 'hooks/useMap';
import useURLSort from 'hooks/useURLSort';
import { createFilterTracker } from 'utils/analyticsEventTracking';
import TableEntityToolbar from '../../components/TableEntityToolbar';

import { parseQuerySearchFilter } from '../../utils/searchUtils';
import AdvancedFiltersToolbar from '../../components/AdvancedFiltersToolbar';
import { clusterSearchFilterConfig, platformCVESearchFilterConfig } from '../../searchFilterConfig';
import { DEFAULT_VM_PAGE_SIZE } from '../../constants';
import EntityTypeToggleGroup from '../../components/EntityTypeToggleGroup';
import { platformEntityTabValues } from '../../types';

import ClustersTable, {
    defaultSortOption as clusterDefaultSortOption,
    sortFields as clusterSortFields,
} from './ClustersTable';
import CVEsTable, {
    defaultSortOption as cveDefaultSortOption,
    sortFields as cveSortFields,
} from './CVEsTable';
import { usePlatformCveEntityCounts } from './usePlatformCveEntityCounts';

const searchFilterConfig = [clusterSearchFilterConfig, platformCVESearchFilterConfig];

function PlatformCvesOverviewPage() {
    const apolloClient = useApolloClient();
    const { analyticsTrack } = useAnalytics();
    const trackAppliedFilter = createFilterTracker(analyticsTrack);

    const [activeEntityTabKey] = useURLStringUnion('entityTab', platformEntityTabValues);
    const { searchFilter, setSearchFilter } = useURLSearch();
    const pagination = useURLPagination(DEFAULT_VM_PAGE_SIZE);
    const { sortOption, getSortParams, setSortOption } = useURLSort({
        sortFields: activeEntityTabKey === 'CVE' ? cveSortFields : clusterSortFields,
        defaultSortOption:
            activeEntityTabKey === 'CVE' ? cveDefaultSortOption : clusterDefaultSortOption,
        onSort: () => pagination.setPage(1),
    });

    const querySearchFilter = parseQuerySearchFilter(searchFilter);
    const isFiltered = getHasSearchApplied(querySearchFilter);

    const selectedCves = useMap<string, { cve: string }>();

    function onEntityTabChange(entityTab: 'CVE' | 'Cluster') {
        pagination.setPage(1);
        setSortOption(entityTab === 'CVE' ? cveDefaultSortOption : clusterDefaultSortOption);

        analyticsTrack({
            event: PLATFORM_CVE_ENTITY_CONTEXT_VIEWED,
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

    const { data } = usePlatformCveEntityCounts(querySearchFilter);

    const entityCounts = {
        CVE: data?.platformCVECount ?? 0,
        Cluster: data?.clusterCount ?? 0,
    };

    function onClearFilters() {
        setSearchFilter({});
        pagination.setPage(1);
    }

    const filterToolbar = (
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
    );

    const entityToggleGroup = (
        <EntityTypeToggleGroup
            entityTabs={['CVE', 'Cluster']}
            entityCounts={entityCounts}
            onChange={onEntityTabChange}
        />
    );

    return (
        <>
            <PageTitle title="Kubernetes Components Overview" />
            <Divider component="div" />
            <PageSection
                className="pf-v5-u-display-flex pf-v5-u-flex-direction-row pf-v5-u-align-items-center"
                variant="light"
            >
                <Flex alignItems={{ default: 'alignItemsCenter' }} className="pf-v5-u-flex-grow-1">
                    <Flex direction={{ default: 'column' }} className="pf-v5-u-flex-grow-1">
                        <Title headingLevel="h1">Kubernetes components</Title>
                        <FlexItem>Prioritize and manage scanned CVEs across clusters</FlexItem>
                    </Flex>
                </Flex>
            </PageSection>
            <PageSection isCenterAligned isFilled>
                <Card>
                    <CardBody>
                        <TableEntityToolbar
                            filterToolbar={filterToolbar}
                            entityToggleGroup={entityToggleGroup}
                            pagination={pagination}
                            tableRowCount={
                                activeEntityTabKey === 'CVE'
                                    ? entityCounts.CVE
                                    : entityCounts.Cluster
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
                        {activeEntityTabKey === 'Cluster' && (
                            <ClustersTable
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
        </>
    );
}

export default PlatformCvesOverviewPage;
