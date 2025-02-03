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

import Header from './Header/Header';
import PublicConfigFooter from './PublicConfig/PublicConfigFooter';
import NavigationSidebar from './Navigation/NavigationSidebar';
import HorizontalSubnav from './Navigation/HorizontalSubnav';

import Body from './Body';
import AcsFeedbackModal from './AcsFeedbackModal';

function MainPage(): ReactElement {
    const history = useHistory();
    const dispatch = useDispatch();

    const { isFeatureFlagEnabled, isLoadingFeatureFlags } = useFeatureFlags();
    const { hasReadAccess, hasReadWriteAccess, isLoadingPermissions } = usePermissions();
    const isLoadingPublicConfig = useSelector(selectors.isLoadingPublicConfigSelector);
    const isLoadingCentralCapabilities = useSelector(selectors.getIsLoadingCentralCapabilities);
    const [isLoadingClustersCount, setIsLoadingClustersCount] = useState(false);
    const showFeedbackModal = useSelector(selectors.feedbackSelector);

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

    return (
        <>
            <div id="PageParent">
                <Button
                    style={{
                        bottom: 'calc(var(--pf-v5-global--spacer--lg) * 6)',
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
                {showFeedbackModal && <AcsFeedbackModal />}
                <Page
                    mainContainerId="main-page-container"
                    header={<Header />}
                    isManagedSidebar
                    sidebar={
                        <NavigationSidebar
                            hasReadAccess={hasReadAccess}
                            isFeatureFlagEnabled={isFeatureFlagEnabled}
                        />
                    }
                >
                    <HorizontalSubnav
                        hasReadAccess={hasReadAccess}
                        isFeatureFlagEnabled={isFeatureFlagEnabled}
                    />
                    <Body
                        hasReadAccess={hasReadAccess}
                        isFeatureFlagEnabled={isFeatureFlagEnabled}
                    />
                </Page>
            </div>
            <footer>
                <PublicConfigFooter />
            </footer>
        </>
    );
}

export default MainPage;
