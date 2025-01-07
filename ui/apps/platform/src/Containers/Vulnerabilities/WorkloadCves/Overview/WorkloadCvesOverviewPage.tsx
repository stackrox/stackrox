import React, { useEffect, useState } from 'react';
import {
    Button,
    Card,
    CardBody,
    Flex,
    FlexItem,
    PageSection,
    Text,
    Title,
} from '@patternfly/react-core';
import { gql, useApolloClient, useQuery } from '@apollo/client';
import cloneDeep from 'lodash/cloneDeep';
import difference from 'lodash/difference';
import isEmpty from 'lodash/isEmpty';

import useURLSearch from 'hooks/useURLSearch';
import useURLStringUnion from 'hooks/useURLStringUnion';
import PageTitle from 'Components/PageTitle';
import useURLPagination from 'hooks/useURLPagination';
import useSelectToggle from 'hooks/patternfly/useSelectToggle';
import usePermissions from 'hooks/usePermissions';
import useAnalytics, {
    WATCH_IMAGE_MODAL_OPENED,
    WORKLOAD_CVE_ENTITY_CONTEXT_VIEWED,
    WORKLOAD_CVE_FILTER_APPLIED,
} from 'hooks/useAnalytics';
import useLocalStorage from 'hooks/useLocalStorage';
import { SearchFilter } from 'types/search';
import {
    getDefaultWorkloadSortOption,
    getDefaultZeroCveSortOption,
    getWorkloadSortFields,
    syncSeveritySortOption,
} from 'Containers/Vulnerabilities/utils/sortUtils';
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

const searchFilterConfig = [
    imageSearchFilterConfig,
    imageCVESearchFilterConfig,
    imageComponentSearchFilterConfig,
    deploymentSearchFilterConfig,
    namespaceSearchFilterConfig,
    clusterSearchFilterConfig,
];

function WorkloadCvesOverviewPage() {
    const apolloClient = useApolloClient();

    const { hasReadAccess, hasReadWriteAccess } = usePermissions();
    const hasWriteAccessForWatchedImage = hasReadWriteAccess('WatchedImage');
    const hasReadAccessForNamespaces = hasReadAccess('Namespace');

    const { analyticsTrack } = useAnalytics();
    const trackAppliedFilter = createFilterTracker(analyticsTrack);

    const { getAbsoluteUrl, pageTitle, baseSearchFilter } = useWorkloadCveViewContext();
    const currentVulnerabilityState = useVulnerabilityState();

    const { searchFilter, setSearchFilter: setURLSearchFilter } = useURLSearch();
    const querySearchFilter = parseQuerySearchFilter(searchFilter);
    const [activeEntityTabKey, setActiveEntityTabKey] = useURLStringUnion(
        'entityTab',
        workloadEntityTabValues
    );
    const [observedCveMode, setObservedCveMode] = useURLStringUnion(
        'observedCveMode',
        observedCveModeValues
    );

    const defaultSearchFilterEntity = getSearchFilterEntityByTab(activeEntityTabKey);

    const isViewingWithCves = observedCveMode === 'WITH_CVES';

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
        ? getDefaultWorkloadSortOption
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

    const defaultStorage: VulnMgmtLocalStorage = {
        preferences: {
            defaultFilters: {
                SEVERITY: ['Critical', 'Important'],
                FIXABLE: ['Fixable'],
            },
        },
    } as const;

    const [localStorageValue, setStoredValue] = useLocalStorage(
        'vulnerabilityManagement',
        defaultStorage,
        isVulnMgmtLocalStorage
    );

    const pagination = useURLPagination(DEFAULT_VM_PAGE_SIZE);

    const sort = useURLSort({
        sortFields: getWorkloadSortFields(activeEntityTabKey),
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
        if (activeEntityTabKey === 'CVE') {
            setActiveEntityTabKey('Image');
            sort.setSortOption(getDefaultSortOption('Image'));
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
        sort.setSortOption(getDefaultWorkloadSortOption('CVE'));
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
        if (isEmpty(searchFilter) && isViewingWithCves) {
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
            entityTabs={
                isViewingWithCves ? ['CVE', 'Image', 'Deployment'] : ['Image', 'Deployment']
            }
            entityCounts={entityCounts}
            onChange={onEntityTabChange}
        />
    );

    return (
        <>
            <PageTitle title={`${pageTitle} Overview`} />
            <PageSection
                className="pf-v5-u-display-flex pf-v5-u-flex-direction-row pf-v5-u-align-items-center"
                variant="light"
            >
                <Flex direction={{ default: 'column' }} className="pf-v5-u-flex-grow-1">
                    <Title headingLevel="h1">{pageTitle}</Title>
                    <FlexItem>
                        Prioritize and manage scanned CVEs across images and deployments
                    </FlexItem>
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
                <PageSection
                    padding={{ default: 'noPadding' }}
                    component="div"
                    className="pf-v5-u-pl-lg pf-v5-u-background-color-100"
                >
                    <VulnerabilityStateTabs onChange={onVulnerabilityStateChange} />
                </PageSection>
                {currentVulnerabilityState === 'OBSERVED' && (
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
                                        {getViewStateTitle(
                                            currentVulnerabilityState ?? 'OBSERVED',
                                            observedCveMode
                                        )}
                                    </Title>
                                    <Text className="pf-v5-u-font-size-sm">
                                        {getViewStateDescription(
                                            currentVulnerabilityState ?? 'OBSERVED',
                                            observedCveMode
                                        )}
                                    </Text>
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
        </>
    );
}

export default WorkloadCvesOverviewPage;
