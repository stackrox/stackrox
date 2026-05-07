import type { ReactNode } from 'react';
import { gql, useQuery } from '@apollo/client';
import { Flex } from '@patternfly/react-core';

import type { CompoundSearchFilterConfig } from 'Components/CompoundSearchFilter/types';
import useAnalytics, { WORKLOAD_CVE_FILTER_APPLIED } from 'hooks/useAnalytics';
import useFeatureFlags from 'hooks/useFeatureFlags';
import usePermissions from 'hooks/usePermissions';
import type { UseURLPaginationResult } from 'hooks/useURLPagination';
import type { UseURLSortResult } from 'hooks/useURLSort';
import type { ColumnConfigOverrides } from 'hooks/useManagedColumns';
import type { VulnerabilityState } from 'types/cve.proto';
import type { SearchFilter } from 'types/search';
import { createFilterTracker } from 'utils/analyticsEventTracking';
import { withActiveDeploymentQuery } from 'utils/deploymentUtils';
import { getHasSearchApplied } from 'utils/searchUtils';
import { ensureExhaustive } from 'utils/type.utils';

import type { DefaultFilters, QuerySearchFilter, WorkloadEntityTab } from '../../types';
import CVEsTableContainer from './CVEsTableContainer';
import DeploymentsTableContainer from './DeploymentsTableContainer';
import ImagesTableContainer from './ImagesTableContainer';
import AdvancedFiltersToolbar from '../../components/AdvancedFiltersToolbar';
import EntityTypeToggleGroup from '../../components/EntityTypeToggleGroup';
import useWorkloadCveViewContext from '../hooks/useWorkloadCveViewContext';
import type { defaultColumns as cveDefaultColumns } from '../Tables/WorkloadCVEOverviewTable';
import type { defaultColumns as imageDefaultColumns } from '../Tables/ImageOverviewTable';
import type { defaultColumns as deploymentDefaultColumns } from '../Tables/DeploymentOverviewTable';

function getSearchFilterEntityByTab(entityTab: WorkloadEntityTab): 'CVE' | 'Image' | 'Deployment' {
    switch (entityTab) {
        case 'CVE':
            return 'CVE';
        case 'Image':
            return 'Image';
        case 'Deployment':
            return 'Deployment';
        default:
            return ensureExhaustive(entityTab);
    }
}

export const entityTypeCountsQuery = gql`
    query getEntityTypeCounts($query: String) {
        imageCount(query: $query)
        deploymentCount(query: $query)
        imageCVECount(query: $query)
    }
`;

// Lightweight query used on the inactive images page to compute entity counts
// that correctly exclude images with active deployments. The main imageCount
// and imageCVECount resolvers cannot express this constraint, so we fetch
// minimal per-image data, filter client-side, and then re-query the CVE count
// scoped to only the truly inactive images.
const inactiveImageCountQuery = gql`
    query getInactiveImageCount($query: String, $activeDeploymentQuery: String) {
        images(query: $query) {
            id
            activeDeploymentCount: deploymentCount(query: $activeDeploymentQuery)
        }
    }
`;

const inactiveCVECountQuery = gql`
    query getInactiveCVECount($query: String) {
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
    additionalToolbarItems: ReactNode;
    additionalHeaderItems: ReactNode;
    showDeferralUI: boolean;
    cveTableColumnOverrides: ColumnConfigOverrides<keyof typeof cveDefaultColumns>;
    imageTableColumnOverrides: ColumnConfigOverrides<keyof typeof imageDefaultColumns>;
    deploymentTableColumnOverrides: ColumnConfigOverrides<keyof typeof deploymentDefaultColumns>;
};

function VulnerabilitiesOverview({
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
    const { isFeatureFlagEnabled } = useFeatureFlags();

    const { analyticsTrack } = useAnalytics();
    const trackAppliedFilter = createFilterTracker(analyticsTrack);

    const { baseSearchFilter, overviewEntityTabs, viewContext } = useWorkloadCveViewContext();

    const isFiltered = getHasSearchApplied(querySearchFilter);

    const defaultSearchFilterEntity = getSearchFilterEntityByTab(activeEntityTabKey);

    const isInactiveWithSoftDeletion =
        viewContext === 'Inactive images' && isFeatureFlagEnabled('ROX_DEPLOYMENT_SOFT_DELETION');

    const { data } = useQuery<{
        imageCount: number;
        imageCVECount: number;
        deploymentCount: number;
    }>(entityTypeCountsQuery, {
        variables: {
            query: workloadCvesScopedQueryString,
        },
    });

    // On the inactive images page with soft deletion, the server-side imageCount
    // and imageCVECount include images that also have active deployments. The
    // image table filters these out client-side (activeDeploymentCount === 0), so
    // the tab counts must be corrected to match. This lightweight query fetches
    // per-image deployment counts so we can identify truly inactive images and
    // re-query the CVE count scoped to only those images.
    const { data: inactiveCountData } = useQuery<{
        images: { id: string; activeDeploymentCount: number }[];
    }>(inactiveImageCountQuery, {
        variables: {
            query: workloadCvesScopedQueryString,
            activeDeploymentQuery: withActiveDeploymentQuery(workloadCvesScopedQueryString, true),
        },
        skip: !isInactiveWithSoftDeletion,
    });

    const inactiveImageIds = isInactiveWithSoftDeletion
        ? (inactiveCountData?.images
              .filter((img) => img.activeDeploymentCount === 0)
              .map((img) => img.id) ?? [])
        : [];

    // Re-query CVE count scoped to only the truly inactive images so the CVE tab
    // count excludes CVEs that belong exclusively to images with active
    // deployments.
    const imageIdTerm = isFeatureFlagEnabled('ROX_FLATTEN_IMAGE_DATA') ? 'Image ID' : 'Image SHA';
    const inactiveCveScopedQuery = `${workloadCvesScopedQueryString}+${imageIdTerm}:${inactiveImageIds.join(',')}`;

    const { data: inactiveCveData } = useQuery<{ imageCVECount: number }>(inactiveCVECountQuery, {
        variables: { query: inactiveCveScopedQuery },
        skip: !isInactiveWithSoftDeletion || inactiveImageIds.length === 0,
    });

    const correctedImageCount = isInactiveWithSoftDeletion
        ? inactiveImageIds.length
        : (data?.imageCount ?? 0);

    const correctedCveCount = isInactiveWithSoftDeletion
        ? inactiveImageIds.length === 0
            ? 0
            : (inactiveCveData?.imageCVECount ?? 0)
        : (data?.imageCVECount ?? 0);

    const entityCounts = {
        CVE: correctedCveCount,
        Image: correctedImageCount,
        Deployment: data?.deploymentCount ?? 0,
    };

    const filterToolbar = (
        <AdvancedFiltersToolbar
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
        <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsMd' }}>
            {additionalHeaderItems && (
                <Flex
                    direction={{ default: 'row' }}
                    alignItems={{ default: 'alignItemsCenter' }}
                    justifyContent={{ default: 'justifyContentSpaceBetween' }}
                >
                    {additionalHeaderItems}
                </Flex>
            )}
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
        </Flex>
    );
}

export default VulnerabilitiesOverview;
