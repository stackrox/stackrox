import type { ReactNode } from 'react';
import { Flex } from '@patternfly/react-core';

import type { CompoundSearchFilterConfig } from 'Components/CompoundSearchFilter/types';
import useAnalytics, { WORKLOAD_CVE_FILTER_APPLIED } from 'hooks/useAnalytics';
import usePermissions from 'hooks/usePermissions';
import type { UseURLPaginationResult } from 'hooks/useURLPagination';
import type { UseURLSortResult } from 'hooks/useURLSort';
import type { ColumnConfigOverrides } from 'hooks/useManagedColumns';
import type { VulnerabilityState } from 'types/cve.proto';
import type { SearchFilter } from 'types/search';
import { createFilterTracker } from 'utils/analyticsEventTracking';
import { getHasSearchApplied } from 'utils/searchUtils';
import { ensureExhaustive } from 'utils/type.utils';

import type {
    DefaultFilters,
    QuerySearchFilter,
    SeverityTab,
    WorkloadEntityTab,
} from '../../types';
import SeverityTabs from '../components/SeverityTabs';
import { useVulnMgmtCounts } from './useVulnMgmtCounts';
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
    countsQueryString: string;
    useSeverityTabs: boolean;
    severityTab: SeverityTab;
    onSeverityTabChange: (severity: SeverityTab) => void;
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
    countsQueryString,
    useSeverityTabs,
    severityTab,
    onSeverityTabChange,
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

    const { counts } = useVulnMgmtCounts(countsQueryString);

    const entityCounts = {
        CVE: counts?.cve ?? 0,
        Image: counts?.image ?? 0,
        Deployment: counts?.deployment ?? 0,
    };

    const severityCounts: Partial<Record<SeverityTab, number>> | undefined = counts
        ? {
              Critical: counts.severity.critical,
              Important: counts.severity.important,
              Moderate: counts.severity.moderate,
              Low: counts.severity.low,
              Unknown: counts.severity.unknown,
          }
        : undefined;

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
            includeCveSeverityFilters={isViewingWithCves && !useSeverityTabs}
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
            {useSeverityTabs && (
                <SeverityTabs
                    activeKey={severityTab}
                    counts={severityCounts}
                    onChange={onSeverityTabChange}
                />
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
