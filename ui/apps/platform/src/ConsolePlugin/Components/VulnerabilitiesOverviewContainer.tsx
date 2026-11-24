import noop from 'lodash/noop';
import { useActiveNamespace } from '@openshift-console/dynamic-plugin-sdk';

import { getSearchFilterConfigWithFeatureFlagDependency } from 'Components/CompoundSearchFilter/utils/utils';
import { DEFAULT_VM_PAGE_SIZE } from 'Containers/Vulnerabilities/constants';
import { severityTabValues, workloadEntityTabValues } from 'Containers/Vulnerabilities/types';
import type {
    DefaultFilters,
    SeverityTab,
    WorkloadEntityTab,
} from 'Containers/Vulnerabilities/types';
import {
    getVulnStateScopedQueryString,
    parseQuerySearchFilter,
    severityLabelToSeverity,
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
    Severity: [],
    Fixable: [],
};

export function VulnerabilitiesOverviewContainer() {
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const isScannerV4Enabled = useIsScannerV4Enabled();
    const [activeNamespace] = useActiveNamespace();
    const { baseSearchFilter } = useWorkloadCveViewContext();
    const { searchFilter, setSearchFilter } = useURLSearch();
    const querySearchFilter = parseQuerySearchFilter(searchFilter);
    const [severityTab, setSeverityTab] = useURLStringUnion('severityTab', severityTabValues);
    const useSeverityTabs = true;

    const countsQuerySearchFilter = { ...querySearchFilter };
    delete countsQuerySearchFilter.Severity;

    // If "All Projects" is selected, use the query search filter's Namespace, otherwise override with the active namespace
    const namespaceFilter =
        activeNamespace === ALL_NAMESPACES_KEY
            ? querySearchFilter.Namespace
            : [activeNamespace];

    const countsQueryString = getVulnStateScopedQueryString(
        {
            ...baseSearchFilter,
            ...countsQuerySearchFilter,
            Namespace: namespaceFilter,
        },
        'OBSERVED'
    );
    const workloadCvesScopedQueryString = getVulnStateScopedQueryString(
        {
            ...baseSearchFilter,
            ...querySearchFilter,
            Namespace: namespaceFilter,
            Severity: [severityLabelToSeverity(severityTab)],
        },
        'OBSERVED'
    );

    const [activeEntityTabKey] = useURLStringUnion('entityTab', workloadEntityTabValues);

    const pagination = useURLPagination(DEFAULT_VM_PAGE_SIZE);

    const sort = useURLSort({
        sortFields: getWorkloadCveOverviewSortFields(activeEntityTabKey),
        defaultSortOption: getWorkloadCveOverviewDefaultSortOption(activeEntityTabKey),
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
        sort.setSortOption(getWorkloadCveOverviewDefaultSortOption(entityTab));
    }

    function onSeverityTabChange(severity: SeverityTab) {
        setSeverityTab(severity);
        pagination.setPage(1);
        if (activeEntityTabKey === 'CVE') {
            sort.setSortOption(getWorkloadCveOverviewDefaultSortOption('CVE'));
        }
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
            countsQueryString={countsQueryString}
            useSeverityTabs={useSeverityTabs}
            severityTab={severityTab}
            onSeverityTabChange={onSeverityTabChange}
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
