import React from 'react';
import { gql, useQuery } from '@apollo/client';
import { Card, CardBody, Flex } from '@patternfly/react-core';

import type { CompoundSearchFilterConfig } from 'Components/CompoundSearchFilter/types';
import type {
    DefaultFilters,
    QuerySearchFilter,
    WorkloadEntityTab,
} from 'Containers/Vulnerabilities/types';
import useAnalytics, { WORKLOAD_CVE_FILTER_APPLIED } from 'hooks/useAnalytics';
import usePermissions from 'hooks/usePermissions';
import type { UseURLPaginationResult } from 'hooks/useURLPagination';
import type { UseURLSortResult } from 'hooks/useURLSort';
import type { ColumnConfigOverrides } from 'hooks/useManagedColumns';
import type { VulnerabilityState } from 'types/cve.proto';
import type { SearchFilter } from 'types/search';
import { createFilterTracker } from 'utils/analyticsEventTracking';
import { getHasSearchApplied } from 'utils/searchUtils';

import CVEsTableContainer from './CVEsTableContainer';
import DeploymentsTableContainer from './DeploymentsTableContainer';
import ImagesTableContainer from './ImagesTableContainer';
import AdvancedFiltersToolbar from '../../components/AdvancedFiltersToolbar';
import EntityTypeToggleGroup from '../../components/EntityTypeToggleGroup';
import useWorkloadCveViewContext from '../hooks/useWorkloadCveViewContext';
import { defaultColumns as cveDefaultColumns } from '../Tables/WorkloadCVEOverviewTable';
import { defaultColumns as imageDefaultColumns } from '../Tables/ImageOverviewTable';
import { defaultColumns as deploymentDefaultColumns } from '../Tables/DeploymentOverviewTable';

function getSearchFilterEntityByTab(
    entityTab: WorkloadEntityTab
): 'CVE' | 'Image' | 'Deployment' | undefined {
    switch (entityTab) {
        case 'CVE':
            return 'CVE';
        case 'Image':
            return 'Image';
        case 'Deployment':
            return 'Deployment';
        default:
            return undefined;
    }
}

export const entityTypeCountsQuery = gql`
    query getEntityTypeCounts($query: String) {
        imageCount(query: $query)
        deploymentCount(query: $query)
        imageCVECount(query: $query)
    }
`;

type VulnerabilitiesOverviewProps = {
    defaultFilters: DefaultFilters;
    searchFilter: SearchFilter;
    setSearchFilter: (searchFilter: SearchFilter) => void;
    querySearchFilter: QuerySearchFilter;
    searchFilterConfig: CompoundSearchFilterConfig;
    workloadCvesScopedQueryString: string;
    pagination: UseURLPaginationResult;
    sort: UseURLSortResult;
    currentVulnerabilityState: VulnerabilityState;
    isViewingWithCves: boolean;
    onWatchImage: (imageName: string) => void;
    onUnwatchImage: (imageName: string) => void;
    activeEntityTabKey: WorkloadEntityTab;
    onEntityTabChange: (entityTab: WorkloadEntityTab) => void;
    additionalToolbarItems: React.ReactNode;
    additionalHeaderItems: React.ReactNode;
    showDeferralUI: boolean;
    cveTableColumnOverrides: ColumnConfigOverrides<keyof typeof cveDefaultColumns>;
    imageTableColumnOverrides: ColumnConfigOverrides<keyof typeof imageDefaultColumns>;
    deploymentTableColumnOverrides: ColumnConfigOverrides<keyof typeof deploymentDefaultColumns>;
};

export function VulnerabilitiesOverview({
    defaultFilters,
    searchFilter,
    setSearchFilter,
    querySearchFilter,
    searchFilterConfig,
    workloadCvesScopedQueryString,
    pagination,
    sort,
    currentVulnerabilityState,
    isViewingWithCves,
    onWatchImage,
    onUnwatchImage,
    activeEntityTabKey,
    onEntityTabChange,
    additionalToolbarItems,
    additionalHeaderItems,
    showDeferralUI,
    cveTableColumnOverrides,
    imageTableColumnOverrides,
    deploymentTableColumnOverrides,
}: VulnerabilitiesOverviewProps) {
    const { hasReadWriteAccess } = usePermissions();
    const hasWriteAccessForWatchedImage = hasReadWriteAccess('WatchedImage');

    const { analyticsTrack } = useAnalytics();
    const trackAppliedFilter = createFilterTracker(analyticsTrack);

    const { baseSearchFilter, overviewEntityTabs } = useWorkloadCveViewContext();

    const isFiltered = getHasSearchApplied(querySearchFilter);

    const defaultSearchFilterEntity = getSearchFilterEntityByTab(activeEntityTabKey);

    const { data } = useQuery<{
        imageCount: number;
        imageCVECount: number;
        deploymentCount: number;
    }>(entityTypeCountsQuery, {
        variables: {
            query: workloadCvesScopedQueryString,
        },
    });

    const entityCounts = {
        CVE: data?.imageCVECount ?? 0,
        Image: data?.imageCount ?? 0,
        Deployment: data?.deploymentCount ?? 0,
    };

    const filterToolbar = (
        <AdvancedFiltersToolbar
            className="pf-v5-u-py-md"
            searchFilterConfig={searchFilterConfig}
            searchFilter={searchFilter}
            additionalContextFilter={{
                'Image CVE Count': isViewingWithCves ? '>0' : '0',
                ...baseSearchFilter,
            }}
            defaultFilters={defaultFilters}
            onFilterChange={(newFilter, searchPayload) => {
                setSearchFilter(newFilter);
                pagination.setPage(1);
                trackAppliedFilter(WORKLOAD_CVE_FILTER_APPLIED, searchPayload);
            }}
            includeCveSeverityFilters={isViewingWithCves}
            includeCveStatusFilters={isViewingWithCves}
            defaultSearchFilterEntity={defaultSearchFilterEntity}
        >
            {additionalToolbarItems}
        </AdvancedFiltersToolbar>
    );

    const entityToggleGroup = (
        <EntityTypeToggleGroup
            entityTabs={overviewEntityTabs}
            entityCounts={entityCounts}
            onChange={onEntityTabChange}
        />
    );

    return (
        <Card>
            <CardBody>
                <Flex
                    direction={{ default: 'row' }}
                    alignItems={{ default: 'alignItemsCenter' }}
                    justifyContent={{ default: 'justifyContentSpaceBetween' }}
                    className="pf-v5-u-px-md pf-v5-u-pb-sm"
                >
                    {additionalHeaderItems}
                </Flex>
                {activeEntityTabKey === 'CVE' && (
                    <CVEsTableContainer
                        searchFilter={searchFilter}
                        onFilterChange={setSearchFilter}
                        filterToolbar={filterToolbar}
                        entityToggleGroup={entityToggleGroup}
                        rowCount={entityCounts.CVE}
                        pagination={pagination}
                        sort={sort}
                        workloadCvesScopedQueryString={workloadCvesScopedQueryString}
                        isFiltered={isFiltered}
                        vulnerabilityState={currentVulnerabilityState}
                        showDeferralUI={showDeferralUI}
                        cveTableColumnOverrides={cveTableColumnOverrides}
                    />
                )}
                {activeEntityTabKey === 'Image' && (
                    <ImagesTableContainer
                        searchFilter={searchFilter}
                        onFilterChange={setSearchFilter}
                        filterToolbar={filterToolbar}
                        entityToggleGroup={entityToggleGroup}
                        rowCount={entityCounts.Image}
                        sort={sort}
                        workloadCvesScopedQueryString={workloadCvesScopedQueryString}
                        isFiltered={isFiltered}
                        pagination={pagination}
                        hasWriteAccessForWatchedImage={hasWriteAccessForWatchedImage}
                        onWatchImage={onWatchImage}
                        onUnwatchImage={onUnwatchImage}
                        imageTableColumnOverrides={imageTableColumnOverrides}
                    />
                )}
                {activeEntityTabKey === 'Deployment' && (
                    <DeploymentsTableContainer
                        searchFilter={searchFilter}
                        onFilterChange={setSearchFilter}
                        filterToolbar={filterToolbar}
                        entityToggleGroup={entityToggleGroup}
                        rowCount={entityCounts.Deployment}
                        pagination={pagination}
                        sort={sort}
                        workloadCvesScopedQueryString={workloadCvesScopedQueryString}
                        isFiltered={isFiltered}
                        deploymentTableColumnOverrides={deploymentTableColumnOverrides}
                    />
                )}
            </CardBody>
        </Card>
    );
}
