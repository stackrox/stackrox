/* eslint-disable no-nested-ternary */
import React, { useEffect, useState } from 'react';
import {
    Button,
    Card,
    CardBody,
    Divider,
    Flex,
    FlexItem,
    PageSection,
    Popover,
    Text,
    Title,
} from '@patternfly/react-core';
import { OutlinedQuestionCircleIcon } from '@patternfly/react-icons';
import { gql, useApolloClient, useQuery } from '@apollo/client';
import cloneDeep from 'lodash/cloneDeep';
import difference from 'lodash/difference';
import isEmpty from 'lodash/isEmpty';
import isEqual from 'lodash/isEqual';

import useURLSearch from 'hooks/useURLSearch';
import useURLStringUnion from 'hooks/useURLStringUnion';
import { getSearchFilterConfigWithFeatureFlagDependency } from 'Components/CompoundSearchFilter/utils/utils';
import PageTitle from 'Components/PageTitle';
import useURLPagination from 'hooks/useURLPagination';
import useSelectToggle from 'hooks/patternfly/useSelectToggle';
import useFeatureFlags from 'hooks/useFeatureFlags';
import usePermissions from 'hooks/usePermissions';
import useAnalytics, {
    WATCH_IMAGE_MODAL_OPENED,
    WORKLOAD_CVE_ENTITY_CONTEXT_VIEWED,
    WORKLOAD_CVE_FILTER_APPLIED,
} from 'hooks/useAnalytics';
import useLocalStorage from 'hooks/useLocalStorage';
import { SearchFilter } from 'types/search';
import {
    getWorkloadCveOverviewDefaultSortOption,
    getDefaultZeroCveSortOption,
    getWorkloadCveOverviewSortFields,
    syncSeveritySortOption,
} from 'Containers/Vulnerabilities/utils/sortUtils';
import { useIsFirstRender } from 'hooks/useIsFirstRender';
import useURLSort from 'hooks/useURLSort';
import { getHasSearchApplied } from 'utils/searchUtils';
import { VulnerabilityState } from 'types/cve.proto';
import AdvancedFiltersToolbar from 'Containers/Vulnerabilities/components/AdvancedFiltersToolbar';
import LinkShim from 'Components/PatternFly/LinkShim';

import { createFilterTracker } from 'utils/analyticsEventTracking';
import {
    clusterSearchFilterConfig,
    deploymentSearchFilterConfig,
    imageCVESearchFilterConfig,
    imageComponentSearchFilterConfig,
    imageSearchFilterConfig,
    namespaceSearchFilterConfig,
} from 'Containers/Vulnerabilities/searchFilterConfig';
import {
    DefaultFilters,
    WorkloadEntityTab,
    VulnMgmtLocalStorage,
    workloadEntityTabValues,
    isVulnMgmtLocalStorage,
    observedCveModeValues,
    ObservedCveMode,
} from '../../types';
import {
    parseQuerySearchFilter,
    getVulnStateScopedQueryString,
    getZeroCveScopedQueryString,
    getNamespaceViewPagePath,
} from '../../utils/searchUtils';
import { DEFAULT_VM_PAGE_SIZE } from '../../constants';

import CVEsTableContainer from './CVEsTableContainer';
import DeploymentsTableContainer from './DeploymentsTableContainer';
import ImagesTableContainer, { imageListQuery } from './ImagesTableContainer';
import WatchedImagesModal from '../WatchedImages/WatchedImagesModal';
import UnwatchImageModal from '../WatchedImages/UnwatchImageModal';
import VulnerabilityStateTabs, {
    vulnStateTabContentId,
} from '../components/VulnerabilityStateTabs';
import useVulnerabilityState from '../hooks/useVulnerabilityState';
import useWorkloadCveViewContext from '../hooks/useWorkloadCveViewContext';
import DefaultFilterModal from '../components/DefaultFilterModal';
import EntityTypeToggleGroup from '../../components/EntityTypeToggleGroup';
import ObservedCveModeSelect from './ObservedCveModeSelect';
import { getViewStateDescription, getViewStateTitle } from './string.utils';
import CreateReportDropdown from '../components/CreateReportDropdown';
import CreateOnDemandReportModal from '../components/CreateOnDemandReportModal';

export const entityTypeCountsQuery = gql`
    query getEntityTypeCounts($query: String) {
        imageCount(query: $query)
        deploymentCount(query: $query)
        imageCVECount(query: $query)
    }
`;

// Merge the default filters with the local filters.
// - Default filters that were removed are removed from the local filters.
// - Default filters that were added are added to the local filters.
// - Existing local filters are preserved.
function mergeDefaultAndLocalFilters(
    oldDefaults: DefaultFilters,
    newDefaults: DefaultFilters,
    searchFilter: SearchFilter
): SearchFilter {
    const filter = cloneDeep(searchFilter);

    let SEVERITY = filter.SEVERITY ?? [];
    let FIXABLE = filter.FIXABLE ?? [];

    // Remove existing applied filters that are no longer in the default filters, then
    // add the new default filters.
    SEVERITY = difference(SEVERITY, oldDefaults.SEVERITY, newDefaults.SEVERITY);
    SEVERITY = SEVERITY.concat(newDefaults.SEVERITY);

    FIXABLE = difference(FIXABLE, oldDefaults.FIXABLE, newDefaults.FIXABLE);
    FIXABLE = FIXABLE.concat(newDefaults.FIXABLE);

    return { ...filter, SEVERITY, FIXABLE };
}

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

const descriptionForVulnerabilityStateMap: Record<VulnerabilityState, string> = {
    OBSERVED: 'Prioritize and triage detected workload vulnerabilities',
    DEFERRED:
        'View workload vulnerabilities that have been postponed for future assessment or action',
    FALSE_POSITIVE:
        'View workload vulnerabilities identified as false positives and excluded from active prioritization',
};

const searchFilterConfigWithFeatureFlagDependency = [
    imageSearchFilterConfig,
    imageCVESearchFilterConfig,
    imageComponentSearchFilterConfig,
    deploymentSearchFilterConfig,
    namespaceSearchFilterConfig,
    clusterSearchFilterConfig,
];

const defaultStorage: VulnMgmtLocalStorage = {
    preferences: {
        defaultFilters: {
            SEVERITY: ['Critical', 'Important'],
            FIXABLE: ['Fixable'],
        },
    },
} as const;

function WorkloadCvesOverviewPage() {
    const apolloClient = useApolloClient();

    const { isFeatureFlagEnabled } = useFeatureFlags();

    const { hasReadAccess, hasReadWriteAccess } = usePermissions();
    const hasWriteAccessForWatchedImage = hasReadWriteAccess('WatchedImage');
    const hasReadAccessForNamespaces = hasReadAccess('Namespace');

    const { analyticsTrack } = useAnalytics();
    const trackAppliedFilter = createFilterTracker(analyticsTrack);

    const {
        getAbsoluteUrl,
        pageTitle,
        pageTitleDescription,
        baseSearchFilter,
        overviewEntityTabs,
        viewContext,
    } = useWorkloadCveViewContext();
    const currentVulnerabilityState = useVulnerabilityState();

    const [observedCveMode, setObservedCveMode] = useURLStringUnion(
        'observedCveMode',
        observedCveModeValues
    );

    // TODO Once the 'ROX_PLATFORM_CVE_SPLIT' flag is removed, we can get rid
    // of the `observedCveMode` state and potentially abstract the detection of "zero cve view"
    // in a way that doesn't require reading the base applied filters
    const isViewingWithCves = isFeatureFlagEnabled('ROX_PLATFORM_CVE_SPLIT')
        ? !(
              'Image CVE Count' in baseSearchFilter &&
              isEqual(baseSearchFilter['Image CVE Count'], ['0'])
          )
        : observedCveMode === 'WITH_CVES';

    const [activeEntityTabKey, setActiveEntityTabKey] = useURLStringUnion(
        'entityTab',
        workloadEntityTabValues,
        isViewingWithCves ? 'CVE' : 'Image'
    );
    const defaultSearchFilterEntity = getSearchFilterEntityByTab(activeEntityTabKey);

    const [localStorageValue, setStoredValue] = useLocalStorage(
        'vulnerabilityManagement',
        defaultStorage,
        isVulnMgmtLocalStorage
    );

    const { searchFilter: urlSearchFilter, setSearchFilter: setURLSearchFilter } = useURLSearch();
    const isFirstRender = useIsFirstRender();

    // If this is the first render of the page, and no other filters are applied, use the default filters
    // as the search filters to apply on the first run of the query. This will only happen once, and on a
    // subsequent render the default filters will be synced with the URL params and page state, if needed.
    const shouldSyncDefaultFilters = isFirstRender && isEmpty(urlSearchFilter) && isViewingWithCves;
    const searchFilter = shouldSyncDefaultFilters
        ? localStorageValue.preferences.defaultFilters
        : urlSearchFilter;

    const querySearchFilter = parseQuerySearchFilter(searchFilter);

    // If the user is viewing observed CVEs, we need to scope the query based on
    // the selected vulnerability state. If the user is viewing _without_ CVEs, we
    // need to scope the query to only show images/deployments with 0 CVEs.
    const workloadCvesScopedQueryString = isViewingWithCves
        ? getVulnStateScopedQueryString(
              {
                  ...baseSearchFilter,
                  ...querySearchFilter,
              },
              currentVulnerabilityState
          )
        : getZeroCveScopedQueryString({
              ...baseSearchFilter,
              ...querySearchFilter,
          });

    const getDefaultSortOption = isViewingWithCves
        ? getWorkloadCveOverviewDefaultSortOption
        : getDefaultZeroCveSortOption;

    const isFiltered = getHasSearchApplied(querySearchFilter);

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

    const pagination = useURLPagination(DEFAULT_VM_PAGE_SIZE);

    const sort = useURLSort({
        sortFields: getWorkloadCveOverviewSortFields(activeEntityTabKey),
        defaultSortOption: getDefaultSortOption(activeEntityTabKey, searchFilter),
        onSort: () => pagination.setPage(1),
    });

    function setSearchFilter(searchFilter: SearchFilter) {
        setURLSearchFilter(searchFilter);
        syncSeveritySortOption(searchFilter, sort.sortOption, sort.setSortOption);
    }

    function updateDefaultFilters(values: DefaultFilters) {
        pagination.setPage(1);
        setStoredValue({ preferences: { defaultFilters: values } });
        setSearchFilter(
            mergeDefaultAndLocalFilters(
                localStorageValue.preferences.defaultFilters,
                values,
                searchFilter
            )
        );
    }

    function onEntityTabChange(entityTab: WorkloadEntityTab) {
        pagination.setPage(1);
        sort.setSortOption(getDefaultSortOption(entityTab, searchFilter));

        analyticsTrack({
            event: WORKLOAD_CVE_ENTITY_CONTEXT_VIEWED,
            properties: {
                type: entityTab,
                page: 'Overview',
            },
        });
    }

    function onChangeObservedCveMode(mode: ObservedCveMode) {
        // Set the observed CVE mode, pushing a new history entry to the stack
        setObservedCveMode(mode);
        // Reset all filters, sorting, and pagination and apply to the current history entry
        pagination.setPage(1);
        setSearchFilter({});
        if (mode === 'WITHOUT_CVES' && activeEntityTabKey !== 'Deployment') {
            setActiveEntityTabKey('Image');
            sort.setSortOption(getDefaultZeroCveSortOption('Image'));
        }

        // Re-apply the default filters when changing modes to the "WITH_CVES" mode
        if (mode === 'WITH_CVES') {
            applyDefaultFilters();
        }
    }

    function onVulnerabilityStateChange(vulnerabilityState: VulnerabilityState) {
        // Reset all filters, sorting, and pagination and apply to the current history entry
        setActiveEntityTabKey('CVE');
        setSearchFilter({});
        sort.setSortOption(getWorkloadCveOverviewDefaultSortOption('CVE'));
        pagination.setPage(1);
        setObservedCveMode('WITH_CVES');

        // Re-apply the default filters when changing to the "OBSERVED" state
        if (vulnerabilityState === 'OBSERVED') {
            applyDefaultFilters();
        }
    }

    function applyDefaultFilters() {
        setSearchFilter(localStorageValue.preferences.defaultFilters);
    }

    // Track the current entity tab when the page is initially visited.
    useEffect(() => {
        onEntityTabChange(activeEntityTabKey);
    }, []);

    // When the page is initially visited and no local filters are applied, apply the default filters.
    //
    // Note that this _does not_ take into account a direct navigation via the left navigation when the user
    // is already on the page. This is because we do not distinguish between navigation via the
    // sidebar and e.g. clearing the page filters.
    useEffect(() => {
        if (shouldSyncDefaultFilters) {
            applyDefaultFilters();
        }
    }, []);

    const [defaultWatchedImageName, setDefaultWatchedImageName] = useState('');
    const watchedImagesModalToggle = useSelectToggle();

    const [unwatchImageName, setUnwatchImageName] = useState('');
    const unwatchImageModalToggle = useSelectToggle();

    function onWatchedImagesChange() {
        return apolloClient.refetchQueries({ include: [imageListQuery] });
    }

    const searchFilterConfig = getSearchFilterConfigWithFeatureFlagDependency(
        isFeatureFlagEnabled,
        searchFilterConfigWithFeatureFlagDependency
    );

    const filterToolbar = (
        <AdvancedFiltersToolbar
            className="pf-v5-u-py-md"
            searchFilterConfig={searchFilterConfig}
            searchFilter={searchFilter}
            additionalContextFilter={{
                'Image CVE Count': isViewingWithCves ? '>0' : '0',
                ...baseSearchFilter,
            }}
            defaultFilters={localStorageValue.preferences.defaultFilters}
            onFilterChange={(newFilter, searchPayload) => {
                setSearchFilter(newFilter);
                pagination.setPage(1);
                trackAppliedFilter(WORKLOAD_CVE_FILTER_APPLIED, searchPayload);
            }}
            includeCveSeverityFilters={isViewingWithCves}
            includeCveStatusFilters={isViewingWithCves}
            defaultSearchFilterEntity={defaultSearchFilterEntity}
        />
    );

    const entityToggleGroup = (
        <EntityTypeToggleGroup
            entityTabs={overviewEntityTabs}
            entityCounts={entityCounts}
            onChange={onEntityTabChange}
        />
    );

    // Report-specific state management
    const [isCreateOnDemandReportModalOpen, setIsCreateOnDemandReportModalOpen] = useState(false);
    const isOnDemandReportsEnabled = isFeatureFlagEnabled('ROX_VULNERABILITY_ON_DEMAND_REPORTS');

    const isOnDemandReportsVisible =
        isOnDemandReportsEnabled &&
        (viewContext === 'User workloads' ||
            viewContext === 'Platform' ||
            viewContext === 'All vulnerable images' ||
            viewContext === 'Inactive images');

    return (
        <>
            <PageTitle title={`${pageTitle} Overview`} />
            <PageSection
                className="pf-v5-u-display-flex pf-v5-u-flex-direction-row pf-v5-u-align-items-center"
                variant="light"
            >
                <Flex
                    direction={{
                        default: isFeatureFlagEnabled('ROX_PLATFORM_CVE_SPLIT') ? 'row' : 'column',
                    }}
                    alignItems={{
                        default: isFeatureFlagEnabled('ROX_PLATFORM_CVE_SPLIT')
                            ? 'alignItemsCenter'
                            : undefined,
                    }}
                    spaceItems={{
                        default: isFeatureFlagEnabled('ROX_PLATFORM_CVE_SPLIT')
                            ? 'spaceItemsNone'
                            : undefined,
                    }}
                    className="pf-v5-u-flex-grow-1"
                >
                    <Title headingLevel="h1">{pageTitle}</Title>
                    {pageTitleDescription && (
                        <Popover
                            aria-label="More information about the current page"
                            bodyContent={pageTitleDescription}
                        >
                            <Button title="Page description" variant="plain">
                                <OutlinedQuestionCircleIcon />
                            </Button>
                        </Popover>
                    )}
                    {!isFeatureFlagEnabled('ROX_PLATFORM_CVE_SPLIT') && (
                        <FlexItem>
                            Prioritize and manage scanned CVEs across images and deployments
                        </FlexItem>
                    )}
                </Flex>
                <Flex>
                    {hasWriteAccessForWatchedImage && (
                        <Button
                            variant="secondary"
                            onClick={() => {
                                setDefaultWatchedImageName('');
                                watchedImagesModalToggle.openSelect();
                                analyticsTrack(WATCH_IMAGE_MODAL_OPENED);
                            }}
                        >
                            Manage watched images
                        </Button>
                    )}
                    {isOnDemandReportsVisible && (
                        <FlexItem>
                            <CreateReportDropdown
                                onSelect={() => {
                                    setIsCreateOnDemandReportModalOpen(true);
                                }}
                            />
                        </FlexItem>
                    )}
                </Flex>
            </PageSection>
            <PageSection id={vulnStateTabContentId} padding={{ default: 'noPadding' }}>
                {!isFeatureFlagEnabled('ROX_PLATFORM_CVE_SPLIT') || isViewingWithCves ? (
                    <PageSection
                        padding={{ default: 'noPadding' }}
                        component="div"
                        className="pf-v5-u-pl-lg pf-v5-u-background-color-100"
                    >
                        <VulnerabilityStateTabs onChange={onVulnerabilityStateChange} />
                    </PageSection>
                ) : (
                    <Divider component="div" />
                )}
                {isFeatureFlagEnabled('ROX_PLATFORM_CVE_SPLIT') && (
                    <PageSection variant="light" component="div">
                        <Text component="p">
                            {isViewingWithCves
                                ? descriptionForVulnerabilityStateMap[currentVulnerabilityState]
                                : 'View images and deployments that do not have detected vulnerabilities'}
                        </Text>
                    </PageSection>
                )}
                {currentVulnerabilityState === 'OBSERVED' &&
                    !isFeatureFlagEnabled('ROX_PLATFORM_CVE_SPLIT') && (
                        <PageSection className="pf-v5-u-py-md" component="div" variant="light">
                            <ObservedCveModeSelect
                                observedCveMode={observedCveMode}
                                setObservedCveMode={onChangeObservedCveMode}
                            />
                        </PageSection>
                    )}
                <PageSection isCenterAligned>
                    <Card>
                        <CardBody>
                            <Flex
                                direction={{ default: 'row' }}
                                alignItems={{ default: 'alignItemsCenter' }}
                                justifyContent={{ default: 'justifyContentSpaceBetween' }}
                                className="pf-v5-u-px-md pf-v5-u-pb-sm"
                            >
                                <FlexItem>
                                    <Title headingLevel="h2">
                                        {isFeatureFlagEnabled('ROX_PLATFORM_CVE_SPLIT')
                                            ? isViewingWithCves
                                                ? 'Vulnerability findings'
                                                : 'Workloads without detected vulnerabilities'
                                            : getViewStateTitle(
                                                  currentVulnerabilityState ?? 'OBSERVED',
                                                  isViewingWithCves
                                              )}
                                    </Title>
                                    {!isFeatureFlagEnabled('ROX_PLATFORM_CVE_SPLIT') && (
                                        <Text className="pf-v5-u-font-size-sm">
                                            {getViewStateDescription(
                                                currentVulnerabilityState ?? 'OBSERVED',
                                                isViewingWithCves
                                            )}
                                        </Text>
                                    )}
                                </FlexItem>
                                {isViewingWithCves &&
                                    (currentVulnerabilityState === 'OBSERVED' ||
                                        currentVulnerabilityState === undefined) && (
                                        <FlexItem>
                                            <Flex
                                                direction={{ default: 'row' }}
                                                alignItems={{ default: 'alignItemsCenter' }}
                                                spaceItems={{ default: 'spaceItemsSm' }}
                                            >
                                                {hasReadAccessForNamespaces && (
                                                    <Button
                                                        variant="secondary"
                                                        href={getAbsoluteUrl(
                                                            getNamespaceViewPagePath()
                                                        )}
                                                        component={LinkShim}
                                                    >
                                                        Prioritize by namespace view
                                                    </Button>
                                                )}
                                                <DefaultFilterModal
                                                    defaultFilters={
                                                        localStorageValue.preferences.defaultFilters
                                                    }
                                                    setLocalStorage={updateDefaultFilters}
                                                />
                                            </Flex>
                                        </FlexItem>
                                    )}
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
                                    onWatchImage={(imageName) => {
                                        setDefaultWatchedImageName(imageName);
                                        watchedImagesModalToggle.openSelect();
                                        analyticsTrack(WATCH_IMAGE_MODAL_OPENED);
                                    }}
                                    onUnwatchImage={(imageName) => {
                                        setUnwatchImageName(imageName);
                                        unwatchImageModalToggle.openSelect();
                                    }}
                                    showCveDetailFields={isViewingWithCves}
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
                                    showCveDetailFields={isViewingWithCves}
                                />
                            )}
                        </CardBody>
                    </Card>
                </PageSection>
            </PageSection>
            <WatchedImagesModal
                defaultWatchedImageName={defaultWatchedImageName}
                isOpen={watchedImagesModalToggle.isOpen}
                onClose={() => {
                    setDefaultWatchedImageName('');
                    watchedImagesModalToggle.closeSelect();
                }}
                onWatchedImagesChange={onWatchedImagesChange}
            />
            <UnwatchImageModal
                unwatchImageName={unwatchImageName}
                isOpen={unwatchImageModalToggle.isOpen}
                onClose={() => {
                    setUnwatchImageName('');
                    unwatchImageModalToggle.closeSelect();
                }}
                onWatchedImagesChange={onWatchedImagesChange}
            />
            {isOnDemandReportsVisible && (
                <CreateOnDemandReportModal
                    isOpen={isCreateOnDemandReportModalOpen}
                    setIsOpen={setIsCreateOnDemandReportModalOpen}
                    query={workloadCvesScopedQueryString}
                    areaOfConcern={viewContext}
                />
            )}
        </>
    );
}

export default WorkloadCvesOverviewPage;
