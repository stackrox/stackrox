import React, { useState } from 'react';
import {
    PageSection,
    Title,
    Divider,
    Flex,
    FlexItem,
    Card,
    CardBody,
    Button,
} from '@patternfly/react-core';
import { useApolloClient, useQuery } from '@apollo/client';

import useURLSearch from 'hooks/useURLSearch';
import useURLStringUnion from 'hooks/useURLStringUnion';
import PageTitle from 'Components/PageTitle';
import useURLPagination from 'hooks/useURLPagination';
import useSelectToggle from 'hooks/patternfly/useSelectToggle';
import usePermissions from 'hooks/usePermissions';
import useFeatureFlags from 'hooks/useFeatureFlags';
import useAnalytics, { WATCH_IMAGE_MODAL_OPENED } from 'hooks/useAnalytics';
import { VulnMgmtLocalStorage, entityTabValues } from '../types';
import { parseQuerySearchFilter, getVulnStateScopedQueryString } from '../searchUtils';
import { entityTypeCountsQuery } from '../components/EntityTypeToggleGroup';
import CVEsTableContainer from './CVEsTableContainer';
import DeploymentsTableContainer from './DeploymentsTableContainer';
import ImagesTableContainer, { imageListQuery } from './ImagesTableContainer';
import WatchedImagesModal from '../WatchedImages/WatchedImagesModal';
import UnwatchImageModal from '../WatchedImages/UnwatchImageModal';
import VulnerabilityStateTabs from '../components/VulnerabilityStateTabs';
import useVulnerabilityState from '../hooks/useVulnerabilityState';

const emptyStorage: VulnMgmtLocalStorage = {
    preferences: {
        defaultFilters: {
            // TODO: re-add default filters to include critical, important, and fixable
            Severity: [],
            Fixable: [],
        },
    },
};

function WorkloadCvesOverviewPage() {
    const apolloClient = useApolloClient();
    const { hasReadWriteAccess } = usePermissions();
    const hasWriteAccessForWatchedImage = hasReadWriteAccess('WatchedImage');
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const isUnifiedDeferralsEnabled = isFeatureFlagEnabled('ROX_VULN_MGMT_UNIFIED_CVE_DEFERRAL');

    const { analyticsTrack } = useAnalytics();

    const currentVulnerabilityState = useVulnerabilityState();

    const { searchFilter } = useURLSearch();
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

    const pagination = useURLPagination(20);

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
            {/* Default filters are disabled until fixability filters are fixed */}
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
                    <VulnerabilityStateTabs />
                </PageSection>
                <PageSection isCenterAligned>
                    <Card>
                        <CardBody>
                            {activeEntityTabKey === 'CVE' && (
                                <CVEsTableContainer
                                    defaultFilters={emptyStorage.preferences.defaultFilters}
                                    countsData={countsData}
                                    pagination={pagination}
                                    vulnerabilityState={currentVulnerabilityState}
                                    isUnifiedDeferralsEnabled={isUnifiedDeferralsEnabled}
                                />
                            )}
                            {activeEntityTabKey === 'Image' && (
                                <ImagesTableContainer
                                    defaultFilters={emptyStorage.preferences.defaultFilters}
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
                                />
                            )}
                            {activeEntityTabKey === 'Deployment' && (
                                <DeploymentsTableContainer
                                    defaultFilters={emptyStorage.preferences.defaultFilters}
                                    countsData={countsData}
                                    pagination={pagination}
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
