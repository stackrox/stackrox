import React, { ReactElement, useEffect, useState } from 'react';
import { useDispatch, useSelector } from 'react-redux';
import { useHistory } from 'react-router-dom';
import { Page, Button } from '@patternfly/react-core';
import { OutlinedCommentsIcon } from '@patternfly/react-icons';

import LoadingSection from 'Components/PatternFly/LoadingSection';
import useFeatureFlags from 'hooks/useFeatureFlags';
import usePermissions from 'hooks/usePermissions';
import { selectors } from 'reducers';
import { actions } from 'reducers/feedback';
import { getClustersForPermissions } from 'services/RolesService';
import { clustersBasePath } from 'routePaths';

import AnnouncementBanner from './Banners/AnnouncementBanner';
import CredentialExpiryBanner from './Banners/CredentialExpiryBanner';
import DatabaseStatusBanner from './Banners/DatabaseStatusBanner';
import OutdatedVersionBanner from './Banners/OutdatedVersionBanner';
import ServerStatusBanner from './Banners/ServerStatusBanner';

import Masthead from './Header/Masthead';

import PublicConfigFooter from './PublicConfig/PublicConfigFooter';
import PublicConfigHeader from './PublicConfig/PublicConfigHeader';

import NavigationSidebar from './Sidebar/NavigationSidebar';

import Body from './Body';
import Notifications from './Notifications';
import AcsFeedbackModal from './AcsFeedbackModal';

function MainPage(): ReactElement {
    const history = useHistory();
    const dispatch = useDispatch();

    const { isFeatureFlagEnabled, isLoadingFeatureFlags } = useFeatureFlags();
    const { hasReadAccess, hasReadWriteAccess, isLoadingPermissions } = usePermissions();
    const isLoadingPublicConfig = useSelector(selectors.isLoadingPublicConfigSelector);
    const isLoadingCentralCapabilities = useSelector(selectors.getIsLoadingCentralCapabilities);
    const [isLoadingClustersCount, setIsLoadingClustersCount] = useState(false);

    const hasWriteAccessForCluster = hasReadWriteAccess('Cluster');
    useEffect(() => {
        if (hasWriteAccessForCluster) {
            setIsLoadingClustersCount(true);
            getClustersForPermissions([])
                .then(({ clusters }) => {
                    // Essential that service function DOES NOT provide a default empty array!
                    if (clusters?.length === 0) {
                        // If no clusters, and user can admin Clusters, redirect to clusters section.
                        // Only applicable in Cloud Services.
                        history.push(clustersBasePath);
                    }
                })
                .catch(() => {})
                .finally(() => {
                    setIsLoadingClustersCount(false);
                });
        }
    }, [hasWriteAccessForCluster, history]);

    // Prerequisites from initial requests for conditional rendering that affects all authenticated routes:
    // feature flags: for NavigationSidebar and Body
    // permissions: for NavigationSidebar and Body
    // public config: for PublicConfigHeader and PublicConfigFooter and analytics
    // central capabilities: for System Health and some integrations
    // clusters: for redirect to clusters
    if (
        isLoadingFeatureFlags ||
        isLoadingPermissions ||
        isLoadingPublicConfig ||
        isLoadingCentralCapabilities ||
        isLoadingClustersCount
    ) {
        return <LoadingSection message="Loading..." />;
    }

    const hasAdministrationWritePermission = hasReadWriteAccess('Administration');

    return (
        <>
            <Notifications />
            <PublicConfigHeader />
            <AnnouncementBanner />
            <CredentialExpiryBanner
                component="CENTRAL"
                hasAdministrationWritePermission={hasAdministrationWritePermission}
            />
            <CredentialExpiryBanner
                component="SCANNER"
                hasAdministrationWritePermission={hasAdministrationWritePermission}
            />
            <OutdatedVersionBanner />
            <DatabaseStatusBanner />
            <ServerStatusBanner />
            <div id="PageParent">
                <Button
                    style={{
                        bottom: 'calc(var(--pf-global--spacer--lg) * 6)',
                        position: 'absolute',
                        right: '0',
                        transform: 'rotate(270deg)',
                        transformOrigin: 'bottom right',
                        zIndex: 20000,
                    }}
                    icon={<OutlinedCommentsIcon />}
                    iconPosition="left"
                    variant="danger"
                    id="feedback-trigger-button"
                    onClick={() => {
                        dispatch(actions.setFeedbackModalVisibility(true));
                    }}
                >
                    Feedback
                </Button>
                <AcsFeedbackModal />
                <Page
                    mainContainerId="main-page-container"
                    header={<Masthead />}
                    isManagedSidebar
                    sidebar={
                        <NavigationSidebar
                            hasReadAccess={hasReadAccess}
                            isFeatureFlagEnabled={isFeatureFlagEnabled}
                        />
                    }
                >
                    <Body
                        hasReadAccess={hasReadAccess}
                        isFeatureFlagEnabled={isFeatureFlagEnabled}
                    />
                </Page>
            </div>
            <PublicConfigFooter />
        </>
    );
}

export default MainPage;
