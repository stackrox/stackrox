import { useEffect, useState } from 'react';
import type { ReactElement } from 'react';
import { useSelector } from 'react-redux';
import { useNavigate } from 'react-router-dom-v5-compat';
import { Page } from '@patternfly/react-core';

import ErrorBoundary from 'Components/PatternFly/ErrorBoundary/ErrorBoundary';
import LoadingSection from 'Components/PatternFly/LoadingSection';
import useFeatureFlags from 'hooks/useFeatureFlags';
import usePermissions from 'hooks/usePermissions';
import usePublicConfig from 'hooks/usePublicConfig';
import { selectors } from 'reducers';
import { getClustersForPermissions } from 'services/RolesService';
import { clustersBasePath } from 'routePaths';

import Banners from './Banners/Banners';
import Header from './Header/Header';
import PublicConfigHeader from './PublicConfig/PublicConfigHeader';
import PublicConfigFooter from './PublicConfig/PublicConfigFooter';
import NavigationSidebar from './Navigation/NavigationSidebar';
import HorizontalSubnav from './Navigation/HorizontalSubnav';

import Body from './Body';

function MainPage(): ReactElement {
    const navigate = useNavigate();

    const { isFeatureFlagEnabled, isLoadingFeatureFlags } = useFeatureFlags();
    const { hasReadAccess, hasReadWriteAccess, isLoadingPermissions } = usePermissions();
    const { publicConfig, isLoadingPublicConfig } = usePublicConfig();
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
                        navigate(clustersBasePath);
                    }
                })
                .catch(() => {})
                .finally(() => {
                    setIsLoadingClustersCount(false);
                });
        }
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [hasWriteAccessForCluster]);

    // Prerequisites from initial requests for conditional rendering that affects all authenticated routes:
    // feature flags: for NavigationSidebar and Body
    // permissions: for NavigationSidebar and Body
    // public config: for PublicConfigHeader and PublicConfigFooter and analytics
    // central capabilities: for System Health and some integrations
    // clusters: for redirect to clusters
    if (
        isLoadingFeatureFlags ||
        isLoadingPermissions ||
        (isLoadingPublicConfig && !publicConfig) ||
        isLoadingCentralCapabilities ||
        isLoadingClustersCount
    ) {
        return <LoadingSection message="Loading..." />;
    }

    return (
        <>
            <div id="PageParent">
                <PublicConfigHeader />
                <Banners />
                <Page
                    mainContainerId="main-page-container"
                    masthead={<Header />}
                    isManagedSidebar
                    sidebar={
                        <NavigationSidebar
                            hasReadAccess={hasReadAccess}
                            isFeatureFlagEnabled={isFeatureFlagEnabled}
                        />
                    }
                >
                    <ErrorBoundary>
                        <HorizontalSubnav
                            hasReadAccess={hasReadAccess}
                            isFeatureFlagEnabled={isFeatureFlagEnabled}
                        />
                        <Body
                            hasReadAccess={hasReadAccess}
                            isFeatureFlagEnabled={isFeatureFlagEnabled}
                        />
                    </ErrorBoundary>
                </Page>
            </div>
            <footer>
                <PublicConfigFooter />
            </footer>
        </>
    );
}

export default MainPage;
