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
import { gql, useApolloClient } from '@apollo/client';
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
import { hideColumnIf } from 'hooks/useManagedColumns';
import useURLSort from 'hooks/useURLSort';
import { VulnerabilityState } from 'types/cve.proto';
import LinkShim from 'Components/PatternFly/LinkShim';

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
} from '../../types';
import {
    parseQuerySearchFilter,
    getVulnStateScopedQueryString,
    getZeroCveScopedQueryString,
    getNamespaceViewPagePath,
} from '../../utils/searchUtils';
import { DEFAULT_VM_PAGE_SIZE } from '../../constants';

import WatchedImagesModal from '../WatchedImages/WatchedImagesModal';
import UnwatchImageModal from '../WatchedImages/UnwatchImageModal';
import VulnerabilityStateTabs, {
    vulnStateTabContentId,
} from '../components/VulnerabilityStateTabs';
import useVulnerabilityState from '../hooks/useVulnerabilityState';
import useWorkloadCveViewContext from '../hooks/useWorkloadCveViewContext';
import DefaultFilterModal from '../components/DefaultFilterModal';
import CreateReportDropdown from '../components/CreateReportDropdown';
import CreateViewBasedReportModal from '../components/CreateViewBasedReportModal';
import { imageListQuery } from '../Tables/ImageOverviewTable';
import useHasRequestExceptionsAbility from '../../hooks/useHasRequestExceptionsAbility';
import VulnerabilitiesOverview from './VulnerabilitiesOverview';

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

const descriptionForVulnerabilityStateMap: Record<VulnerabilityState, string> = {
    OBSERVED: 'Prioritize and triage detected workload vulnerabilities',
    DEFERRED:
        'View workload vulnerabilities that have been postponed for future assessment or action',
    FALSE_POSITIVE:
        'View workload vulnerabilities identified as false positives and excluded from active prioritization',
};

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
    const hasWriteAccessForImage = hasReadWriteAccess('Image'); // SBOM Generation mutates image scan state.
    const hasWorkflowAdminAccess = hasReadAccess('WorkflowAdministration');

    const { analyticsTrack } = useAnalytics();

    const { urlBuilder, pageTitle, pageTitleDescription, baseSearchFilter, viewContext } =
        useWorkloadCveViewContext();
    const currentVulnerabilityState = useVulnerabilityState();

    // TODO We can potentially abstract the detection of "zero cve view"
    // in a way that doesn't require reading the base applied filters
    const isViewingWithCves = !(
        'Image CVE Count' in baseSearchFilter && isEqual(baseSearchFilter['Image CVE Count'], ['0'])
    );

    const [activeEntityTabKey, setActiveEntityTabKey] = useURLStringUnion(
        'entityTab',
        workloadEntityTabValues,
        isViewingWithCves ? 'CVE' : 'Image'
    );

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

    function onVulnerabilityStateChange(vulnerabilityState: VulnerabilityState) {
        // Reset all filters, sorting, and pagination and apply to the current history entry
        setActiveEntityTabKey('CVE');
        setSearchFilter({});
        sort.setSortOption(getWorkloadCveOverviewDefaultSortOption('CVE'));
        pagination.setPage(1);

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

    // Keep searchFilterConfigWithFeatureFlagDependency for ROX_SCANNER_V4.
    const searchFilterConfigWithFeatureFlagDependency = [
        imageSearchFilterConfig,
        imageCVESearchFilterConfig,
        imageComponentSearchFilterConfig,
        deploymentSearchFilterConfig,
        namespaceSearchFilterConfig,
        clusterSearchFilterConfig,
    ];

    const searchFilterConfig = getSearchFilterConfigWithFeatureFlagDependency(
        isFeatureFlagEnabled,
        searchFilterConfigWithFeatureFlagDependency
    );

    // Report-specific state management
    const [isCreateViewBasedReportModalOpen, setIsCreateViewBasedReportModalOpen] = useState(false);
    const isViewBasedReportsEnabled = isFeatureFlagEnabled('ROX_VULNERABILITY_VIEW_BASED_REPORTS');

    const isOnDemandReportsVisible =
        isViewBasedReportsEnabled &&
        hasWorkflowAdminAccess &&
        (viewContext === 'User workloads' ||
            viewContext === 'Platform' ||
            viewContext === 'All vulnerable images' ||
            viewContext === 'Inactive images');

    const hasRequestExceptionsAbility = useHasRequestExceptionsAbility();
    const showDeferralUI = hasRequestExceptionsAbility && currentVulnerabilityState === 'OBSERVED';

    return (
        <>
            <PageTitle title={`${pageTitle} Overview`} />
            <PageSection
                className="pf-v5-u-display-flex pf-v5-u-flex-direction-row pf-v5-u-align-items-center"
                variant="light"
            >
                <Flex
                    direction={{
                        default: 'row',
                    }}
                    alignItems={{
                        default: 'alignItemsCenter',
                    }}
                    spaceItems={{
                        default: 'spaceItemsNone',
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
                </Flex>
            </PageSection>
            <PageSection id={vulnStateTabContentId} padding={{ default: 'noPadding' }}>
                {isViewingWithCves ? (
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
                <PageSection variant="light" component="div">
                    <Text component="p">
                        {isViewingWithCves
                            ? descriptionForVulnerabilityStateMap[currentVulnerabilityState]
                            : 'View images and deployments that do not have detected vulnerabilities'}
                    </Text>
                </PageSection>
                <PageSection isCenterAligned>
                    <Card>
                        <CardBody>
                            <VulnerabilitiesOverview
                                defaultFilters={localStorageValue.preferences.defaultFilters}
                                searchFilter={searchFilter}
                                setSearchFilter={setSearchFilter}
                                querySearchFilter={querySearchFilter}
                                workloadCvesScopedQueryString={workloadCvesScopedQueryString}
                                searchFilterConfig={searchFilterConfig}
                                pagination={pagination}
                                sort={sort}
                                currentVulnerabilityState={currentVulnerabilityState}
                                isViewingWithCves={isViewingWithCves}
                                onWatchImage={(imageName) => {
                                    setDefaultWatchedImageName(imageName);
                                    watchedImagesModalToggle.openSelect();
                                    analyticsTrack(WATCH_IMAGE_MODAL_OPENED);
                                }}
                                onUnwatchImage={(imageName) => {
                                    setUnwatchImageName(imageName);
                                    unwatchImageModalToggle.openSelect();
                                }}
                                onEntityTabChange={onEntityTabChange}
                                activeEntityTabKey={activeEntityTabKey}
                                additionalToolbarItems={
                                    isOnDemandReportsVisible && (
                                        <CreateReportDropdown
                                            onSelect={() => {
                                                setIsCreateViewBasedReportModalOpen(true);
                                            }}
                                        />
                                    )
                                }
                                additionalHeaderItems={
                                    <>
                                        <FlexItem>
                                            <Title headingLevel="h2">
                                                {isViewingWithCves
                                                    ? 'Vulnerability findings'
                                                    : 'Workloads without detected vulnerabilities'}
                                            </Title>
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
                                                                href={urlBuilder.vulnMgmtBase(
                                                                    getNamespaceViewPagePath()
                                                                )}
                                                                component={LinkShim}
                                                            >
                                                                Prioritize by namespace view
                                                            </Button>
                                                        )}
                                                        <DefaultFilterModal
                                                            defaultFilters={
                                                                localStorageValue.preferences
                                                                    .defaultFilters
                                                            }
                                                            setLocalStorage={updateDefaultFilters}
                                                        />
                                                    </Flex>
                                                </FlexItem>
                                            )}
                                    </>
                                }
                                showDeferralUI={showDeferralUI}
                                cveTableColumnOverrides={{
                                    cveSelection: hideColumnIf(!showDeferralUI),
                                    topNvdCvss: hideColumnIf(
                                        !isFeatureFlagEnabled('ROX_SCANNER_V4')
                                    ),
                                    epssProbability: hideColumnIf(
                                        !isFeatureFlagEnabled('ROX_SCANNER_V4')
                                    ),
                                    requestDetails: hideColumnIf(
                                        currentVulnerabilityState === 'OBSERVED'
                                    ),
                                    rowActions: hideColumnIf(!showDeferralUI),
                                }}
                                imageTableColumnOverrides={{
                                    cvesBySeverity: hideColumnIf(!isViewingWithCves),
                                    rowActions: hideColumnIf(
                                        !hasWriteAccessForWatchedImage && !hasWriteAccessForImage
                                    ),
                                }}
                                deploymentTableColumnOverrides={{
                                    cvesBySeverity: hideColumnIf(!isViewingWithCves),
                                }}
                            />
                        </CardBody>
                    </Card>
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
                    <CreateViewBasedReportModal
                        isOpen={isCreateViewBasedReportModalOpen}
                        setIsOpen={setIsCreateViewBasedReportModalOpen}
                        query={workloadCvesScopedQueryString}
                        areaOfConcern={viewContext}
                    />
                )}
            </PageSection>
        </>
    );
}

export default WorkloadCvesOverviewPage;
