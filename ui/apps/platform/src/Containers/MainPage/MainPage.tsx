import React, { ReactElement } from 'react';
import { useHistory } from 'react-router-dom';
import { Page } from '@patternfly/react-core';
import { gql, useQuery } from '@apollo/client';

import LoadingSection from 'Components/PatternFly/LoadingSection';
import AppWrapper from 'Containers/AppWrapper';
import useFeatureFlags from 'hooks/useFeatureFlags';
import usePermissions from 'hooks/usePermissions';
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

type ClusterCountResponse = {
    clusterCount: number;
};

const CLUSTER_COUNT = gql`
    query summary_counts {
        clusterCount
    }
`;

function MainPage(): ReactElement {
    const history = useHistory();

    const { isFeatureFlagEnabled, isLoadingFeatureFlags } = useFeatureFlags();
    const { hasReadAccess, hasReadWriteAccess, isLoadingPermissions } = usePermissions();

    // Check for clusters under management
    // if none, and user can admin Clusters, redirect to clusters section
    // (only applicable in Cloud Services version)
    const hasClusterWritePermission = hasReadWriteAccess('Cluster');

    useQuery<ClusterCountResponse>(CLUSTER_COUNT, {
        onCompleted: (data) => {
            if (hasClusterWritePermission && data?.clusterCount < 1) {
                history.push(clustersBasePath);
            }
        },
    });

    // Render Body and NavigationSideBar only when feature flags and permissions are available.
    if (isLoadingFeatureFlags || isLoadingPermissions) {
        return <LoadingSection message="Loading..." />;
    }

    // TODO: ROX-12750 Replace ServiceIdentity with Administration
    const hasServiceIdentityWritePermission = hasReadWriteAccess('ServiceIdentity');

    return (
        <AppWrapper>
            <PublicConfigHeader />
            <div className="flex flex-1 flex-col h-full relative">
                <AnnouncementBanner />
                <ServerStatusBanner />
                <Notifications />
                <CredentialExpiryBanner
                    component="CENTRAL"
                    hasServiceIdentityWritePermission={hasServiceIdentityWritePermission}
                />
                <CredentialExpiryBanner
                    component="SCANNER"
                    hasServiceIdentityWritePermission={hasServiceIdentityWritePermission}
                />
                <OutdatedVersionBanner />
                <DatabaseStatusBanner />
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
        </AppWrapper>
    );
}

export default MainPage;
