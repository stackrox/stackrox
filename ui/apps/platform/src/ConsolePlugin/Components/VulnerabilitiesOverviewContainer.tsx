import React from 'react';
import noop from 'lodash/noop';
import { useActiveNamespace } from '@openshift-console/dynamic-plugin-sdk';

import { getSearchFilterConfigWithFeatureFlagDependency } from 'Components/CompoundSearchFilter/utils/utils';
import { DEFAULT_VM_PAGE_SIZE } from 'Containers/Vulnerabilities/constants';
import { workloadEntityTabValues } from 'Containers/Vulnerabilities/types';
import type { DefaultFilters, WorkloadEntityTab } from 'Containers/Vulnerabilities/types';
import {
    getVulnStateScopedQueryString,
    parseQuerySearchFilter,
} from 'Containers/Vulnerabilities/utils/searchUtils';
import {
    getWorkloadCveOverviewDefaultSortOption,
    getWorkloadCveOverviewSortFields,
} from 'Containers/Vulnerabilities/utils/sortUtils';
import useWorkloadCveViewContext from 'Containers/Vulnerabilities/WorkloadCves/hooks/useWorkloadCveViewContext';
import VulnerabilitiesOverview from 'Containers/Vulnerabilities/WorkloadCves/Overview/VulnerabilitiesOverview';
import {
    deploymentSearchFilterConfig,
    imageCVESearchFilterConfig,
    imageComponentSearchFilterConfig,
    imageSearchFilterConfig,
    namespaceSearchFilterConfig,
} from 'Containers/Vulnerabilities/searchFilterConfig';

import useURLPagination from 'hooks/useURLPagination';
import useURLSearch from 'hooks/useURLSearch';
import useURLSort from 'hooks/useURLSort';
import useURLStringUnion from 'hooks/useURLStringUnion';
import { hideColumnIf } from 'hooks/useManagedColumns';
import useIsScannerV4Enabled from 'hooks/useIsScannerV4Enabled';
import useFeatureFlags from 'hooks/useFeatureFlags';

import { ALL_NAMESPACES_KEY } from '../constants';

const emptyDefaultFilters: DefaultFilters = {
    SEVERITY: [],
    FIXABLE: [],
};

export function VulnerabilitiesOverviewContainer() {
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const isScannerV4Enabled = useIsScannerV4Enabled();
    const [activeNamespace] = useActiveNamespace();
    const { baseSearchFilter } = useWorkloadCveViewContext();
    const { searchFilter, setSearchFilter } = useURLSearch();
    const querySearchFilter = parseQuerySearchFilter(searchFilter);
    const workloadCvesScopedQueryString = getVulnStateScopedQueryString(
        {
            ...baseSearchFilter,
            ...querySearchFilter,
            // If "All Projects" is selected, use the query search filter's Namespace, otherwise override with the active namespace
            Namespace:
                activeNamespace === ALL_NAMESPACES_KEY
                    ? querySearchFilter.Namespace
                    : [activeNamespace],
        },
        'OBSERVED'
    );

    const [activeEntityTabKey] = useURLStringUnion('entityTab', workloadEntityTabValues);

    const pagination = useURLPagination(DEFAULT_VM_PAGE_SIZE);

    const sort = useURLSort({
        sortFields: getWorkloadCveOverviewSortFields(activeEntityTabKey),
        defaultSortOption: getWorkloadCveOverviewDefaultSortOption(
            activeEntityTabKey,
            searchFilter
        ),
        onSort: () => pagination.setPage(1),
    });

    const searchFilterConfig = getSearchFilterConfigWithFeatureFlagDependency(
        isFeatureFlagEnabled,
        [
            imageSearchFilterConfig,
            imageCVESearchFilterConfig,
            imageComponentSearchFilterConfig,
            deploymentSearchFilterConfig,
            ...(activeNamespace === ALL_NAMESPACES_KEY ? [namespaceSearchFilterConfig] : []),
        ]
    );

    function onEntityTabChange(entityTab: WorkloadEntityTab) {
        pagination.setPage(1);
        sort.setSortOption(getWorkloadCveOverviewDefaultSortOption(entityTab, searchFilter));
    }

    return (
        <VulnerabilitiesOverview
            defaultFilters={emptyDefaultFilters}
            searchFilter={searchFilter}
            setSearchFilter={setSearchFilter}
            querySearchFilter={querySearchFilter}
            workloadCvesScopedQueryString={workloadCvesScopedQueryString}
            searchFilterConfig={searchFilterConfig}
            pagination={pagination}
            sort={sort}
            currentVulnerabilityState={'OBSERVED'}
            isViewingWithCves
            onWatchImage={noop}
            onUnwatchImage={noop}
            activeEntityTabKey={activeEntityTabKey}
            onEntityTabChange={onEntityTabChange}
            additionalToolbarItems={undefined}
            additionalHeaderItems={undefined}
            showDeferralUI={false}
            cveTableColumnOverrides={{
                cveSelection: hideColumnIf(true),
                rowActions: hideColumnIf(true),
                requestDetails: hideColumnIf(true),
                affectedImages: hideColumnIf(true),
                topNvdCvss: hideColumnIf(!isScannerV4Enabled),
                epssProbability: hideColumnIf(!isScannerV4Enabled),
            }}
            imageTableColumnOverrides={{
                rowActions: hideColumnIf(true),
            }}
            deploymentTableColumnOverrides={{
                namespace: hideColumnIf(activeNamespace !== ALL_NAMESPACES_KEY),
                cluster: hideColumnIf(true),
            }}
        />
    );
}
