import React, { useEffect, useState } from 'react';
import {
    PageSection,
    Title,
    Divider,
    Flex,
    FlexItem,
    Card,
    CardBody,
    Button,
    Toolbar,
    ToolbarItem,
} from '@patternfly/react-core';
import { useApolloClient, useQuery } from '@apollo/client';
import cloneDeep from 'lodash/cloneDeep';
import difference from 'lodash/difference';
import isEmpty from 'lodash/isEmpty';

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
import {
    DefaultFilters,
    EntityTab,
    VulnMgmtLocalStorage,
    entityTabValues,
    isVulnMgmtLocalStorage,
} from '../types';
import { parseQuerySearchFilter, getVulnStateScopedQueryString } from '../searchUtils';
import { entityTypeCountsQuery } from '../components/EntityTypeToggleGroup';
import CVEsTableContainer from './CVEsTableContainer';
import DeploymentsTableContainer from './DeploymentsTableContainer';
import ImagesTableContainer, { imageListQuery } from './ImagesTableContainer';
import WatchedImagesModal from '../WatchedImages/WatchedImagesModal';
import UnwatchImageModal from '../WatchedImages/UnwatchImageModal';
import VulnerabilityStateTabs from '../components/VulnerabilityStateTabs';
import useVulnerabilityState from '../hooks/useVulnerabilityState';
import DefaultFilterModal from '../components/DefaultFilterModal';

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
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const isUnifiedDeferralsEnabled = isFeatureFlagEnabled('ROX_VULN_MGMT_UNIFIED_CVE_DEFERRAL');
    const isFixabilityFiltersEnabled = isFeatureFlagEnabled('ROX_WORKLOAD_CVES_FIXABILITY_FILTERS');

    const { analyticsTrack } = useAnalytics();

    const currentVulnerabilityState = useVulnerabilityState();

    const { searchFilter, setSearchFilter } = useURLSearch();
    const querySearchFilter = parseQuerySearchFilter(searchFilter);
    const [activeEntityTabKey] = useURLStringUnion('entityTab', entityTabValues);

    const { data: countsData = { imageCount: 0, imageCVECount: 0, deploymentCount: 0 } } = useQuery(
        entityTypeCountsQuery,
        {
            variables: {
                query: getVulnStateScopedQueryString(querySearchFilter, currentVulnerabilityState),
            },
        }
    );

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

    function onEntityTabChange(entityTab: EntityTab) {
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

    return (
        <>
            <PageTitle title="Workload CVEs Overview" />
            {isFixabilityFiltersEnabled && (
                <PageSection variant="light" padding={{ default: 'noPadding' }}>
                    <Toolbar>
                        <ToolbarItem alignment={{ default: 'alignRight' }}>
                            <DefaultFilterModal
                                defaultFilters={localStorageValue.preferences.defaultFilters}
                                setLocalStorage={updateDefaultFilters}
                            />
                        </ToolbarItem>
                    </Toolbar>
                </PageSection>
            )}
            <Divider component="div" />
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
                {hasWriteAccessForWatchedImage && (
                    <FlexItem>
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
                    </FlexItem>
                )}
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
                                    defaultFilters={localStorageValue.preferences.defaultFilters}
                                    countsData={countsData}
                                    pagination={pagination}
                                    vulnerabilityState={currentVulnerabilityState}
                                    isUnifiedDeferralsEnabled={isUnifiedDeferralsEnabled}
                                    onEntityTabChange={onEntityTabChange}
                                />
                            )}
                            {activeEntityTabKey === 'Image' && (
                                <ImagesTableContainer
                                    defaultFilters={localStorageValue.preferences.defaultFilters}
                                    countsData={countsData}
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
                                    onEntityTabChange={onEntityTabChange}
                                />
                            )}
                            {activeEntityTabKey === 'Deployment' && (
                                <DeploymentsTableContainer
                                    defaultFilters={localStorageValue.preferences.defaultFilters}
                                    countsData={countsData}
                                    pagination={pagination}
                                    vulnerabilityState={currentVulnerabilityState}
                                    onEntityTabChange={onEntityTabChange}
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
