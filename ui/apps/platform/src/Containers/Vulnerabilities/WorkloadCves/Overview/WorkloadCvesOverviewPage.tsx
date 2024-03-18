import React, { useEffect, useState } from 'react';
import { PageSection, Title, Flex, FlexItem, Card, CardBody, Button } from '@patternfly/react-core';
import { gql, useApolloClient, useQuery } from '@apollo/client';
import cloneDeep from 'lodash/cloneDeep';
import difference from 'lodash/difference';
import isEmpty from 'lodash/isEmpty';
import { Link } from 'react-router-dom';

import useURLSearch from 'hooks/useURLSearch';
import useURLStringUnion from 'hooks/useURLStringUnion';
import PageTitle from 'Components/PageTitle';
import useURLPagination from 'hooks/useURLPagination';
import useSelectToggle from 'hooks/patternfly/useSelectToggle';
import usePermissions from 'hooks/usePermissions';
import useFeatureFlags from 'hooks/useFeatureFlags';
import useAnalytics, {
    WATCH_IMAGE_MODAL_OPENED,
    WORKLOAD_CVE_ENTITY_CONTEXT_VIEWED,
} from 'hooks/useAnalytics';
import useLocalStorage from 'hooks/useLocalStorage';
import { SearchFilter } from 'types/search';
import { vulnerabilityNamespaceViewPath } from 'routePaths';
import {
    getDefaultWorkloadSortOption,
    getWorkloadSortFields,
} from 'Containers/Vulnerabilities/utils/sortUtils';
import useURLSort from 'hooks/useURLSort';
import {
    SearchOption,
    IMAGE_SEARCH_OPTION,
    DEPLOYMENT_SEARCH_OPTION,
    NAMESPACE_SEARCH_OPTION,
    CLUSTER_SEARCH_OPTION,
    IMAGE_CVE_SEARCH_OPTION,
    COMPONENT_SEARCH_OPTION,
    COMPONENT_SOURCE_SEARCH_OPTION,
} from 'Containers/Vulnerabilities/searchOptions';
import {
    DefaultFilters,
    WorkloadEntityTab,
    VulnMgmtLocalStorage,
    workloadEntityTabValues,
    isVulnMgmtLocalStorage,
} from '../../types';
import {
    parseWorkloadQuerySearchFilter,
    getVulnStateScopedQueryString,
} from '../../utils/searchUtils';
import CVEsTableContainer from './CVEsTableContainer';
import DeploymentsTableContainer from './DeploymentsTableContainer';
import ImagesTableContainer, { imageListQuery } from './ImagesTableContainer';
import WatchedImagesModal from '../WatchedImages/WatchedImagesModal';
import UnwatchImageModal from '../WatchedImages/UnwatchImageModal';
import VulnerabilityStateTabs from '../components/VulnerabilityStateTabs';
import useVulnerabilityState from '../hooks/useVulnerabilityState';
import DefaultFilterModal from '../components/DefaultFilterModal';
import WorkloadCveFilterToolbar from '../components/WorkloadCveFilterToolbar';
import EntityTypeToggleGroup from '../../components/EntityTypeToggleGroup';

const searchOptions: SearchOption[] = [
    IMAGE_SEARCH_OPTION,
    DEPLOYMENT_SEARCH_OPTION,
    NAMESPACE_SEARCH_OPTION,
    CLUSTER_SEARCH_OPTION,
    IMAGE_CVE_SEARCH_OPTION,
    COMPONENT_SEARCH_OPTION,
    COMPONENT_SOURCE_SEARCH_OPTION,
];

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

function WorkloadCvesOverviewPage() {
    const apolloClient = useApolloClient();

    const { hasReadWriteAccess } = usePermissions();
    const hasWriteAccessForWatchedImage = hasReadWriteAccess('WatchedImage');
    const hasReadAccessForNamespaces = hasReadWriteAccess('Namespace');

    const { isFeatureFlagEnabled } = useFeatureFlags();
    const isUnifiedDeferralsEnabled = isFeatureFlagEnabled('ROX_VULN_MGMT_UNIFIED_CVE_DEFERRAL');
    const isFixabilityFiltersEnabled = isFeatureFlagEnabled('ROX_WORKLOAD_CVES_FIXABILITY_FILTERS');

    const { analyticsTrack } = useAnalytics();

    const currentVulnerabilityState = useVulnerabilityState();

    const { searchFilter, setSearchFilter } = useURLSearch();
    const querySearchFilter = parseWorkloadQuerySearchFilter(searchFilter);
    const [activeEntityTabKey] = useURLStringUnion('entityTab', workloadEntityTabValues);

    const { data } = useQuery<{
        imageCount: number;
        imageCVECount: number;
        deploymentCount: number;
    }>(entityTypeCountsQuery, {
        variables: {
            query: getVulnStateScopedQueryString(querySearchFilter, currentVulnerabilityState),
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
                SEVERITY: isFixabilityFiltersEnabled ? ['Critical', 'Important'] : [],
                FIXABLE: isFixabilityFiltersEnabled ? ['Fixable'] : [],
            },
        },
    } as const;

    const [storedValue, setStoredValue] = useLocalStorage(
        'vulnerabilityManagement',
        defaultStorage,
        isVulnMgmtLocalStorage
    );
    // Until the ROX_VULN_MGMT_FIXABILITY_FILTERS feature flag is removed, we need to used empty default filters
    // as a fallback
    const localStorageValue = isFixabilityFiltersEnabled ? storedValue : defaultStorage;

    const pagination = useURLPagination(20);

    const sort = useURLSort({
        sortFields: getWorkloadSortFields(activeEntityTabKey),
        defaultSortOption: getDefaultWorkloadSortOption(activeEntityTabKey),
        onSort: () => pagination.setPage(1),
    });

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
        sort.setSortOption(getDefaultWorkloadSortOption(entityTab));

        analyticsTrack({
            event: WORKLOAD_CVE_ENTITY_CONTEXT_VIEWED,
            properties: {
                type: entityTab,
                page: 'Overview',
            },
        });
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
        if (isFixabilityFiltersEnabled && isEmpty(searchFilter)) {
            setSearchFilter(localStorageValue.preferences.defaultFilters, 'replace');
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
        <WorkloadCveFilterToolbar
            defaultFilters={localStorageValue.preferences.defaultFilters}
            onFilterChange={() => pagination.setPage(1)}
            searchOptions={searchOptions}
        />
    );

    const entityToggleGroup = (
        <EntityTypeToggleGroup
            entityTabs={['CVE', 'Image', 'Deployment']}
            entityCounts={entityCounts}
            onChange={onEntityTabChange}
        />
    );

    return (
        <>
            <PageTitle title="Workload CVEs Overview" />
            <PageSection
                className="pf-u-display-flex pf-u-flex-direction-row pf-u-align-items-center"
                variant="light"
            >
                <Flex direction={{ default: 'column' }} className="pf-u-flex-grow-1">
                    <Title headingLevel="h1">Workload CVEs</Title>
                    <FlexItem>
                        Prioritize and manage scanned CVEs across images and deployments
                    </FlexItem>
                </Flex>
                <Flex>
                    {hasReadAccessForNamespaces && (
                        <Link to={vulnerabilityNamespaceViewPath}>
                            <Button variant="secondary" onClick={() => {}}>
                                Namespace view
                            </Button>
                        </Link>
                    )}
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
                    {isFixabilityFiltersEnabled && (
                        <DefaultFilterModal
                            defaultFilters={localStorageValue.preferences.defaultFilters}
                            setLocalStorage={updateDefaultFilters}
                        />
                    )}
                </Flex>
            </PageSection>
            <PageSection padding={{ default: 'noPadding' }}>
                <PageSection
                    padding={{ default: 'noPadding' }}
                    component="div"
                    className="pf-u-pl-lg pf-u-background-color-100"
                >
                    <VulnerabilityStateTabs onChange={() => pagination.setPage(1)} />
                </PageSection>
                <PageSection isCenterAligned>
                    <Card>
                        <CardBody>
                            {activeEntityTabKey === 'CVE' && (
                                <CVEsTableContainer
                                    filterToolbar={filterToolbar}
                                    entityToggleGroup={entityToggleGroup}
                                    rowCount={entityCounts.CVE}
                                    pagination={pagination}
                                    sort={sort}
                                    vulnerabilityState={currentVulnerabilityState}
                                    isUnifiedDeferralsEnabled={isUnifiedDeferralsEnabled}
                                />
                            )}
                            {activeEntityTabKey === 'Image' && (
                                <ImagesTableContainer
                                    filterToolbar={filterToolbar}
                                    entityToggleGroup={entityToggleGroup}
                                    rowCount={entityCounts.Image}
                                    sort={sort}
                                    pagination={pagination}
                                    hasWriteAccessForWatchedImage={hasWriteAccessForWatchedImage}
                                    vulnerabilityState={currentVulnerabilityState}
                                    onWatchImage={(imageName) => {
                                        setDefaultWatchedImageName(imageName);
                                        watchedImagesModalToggle.openSelect();
                                        analyticsTrack(WATCH_IMAGE_MODAL_OPENED);
                                    }}
                                    onUnwatchImage={(imageName) => {
                                        setUnwatchImageName(imageName);
                                        unwatchImageModalToggle.openSelect();
                                    }}
                                />
                            )}
                            {activeEntityTabKey === 'Deployment' && (
                                <DeploymentsTableContainer
                                    filterToolbar={filterToolbar}
                                    entityToggleGroup={entityToggleGroup}
                                    rowCount={entityCounts.Deployment}
                                    pagination={pagination}
                                    sort={sort}
                                    vulnerabilityState={currentVulnerabilityState}
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
