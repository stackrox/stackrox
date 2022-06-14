import React, { ReactElement } from 'react';
import { useDispatch, useSelector } from 'react-redux';
import { useHistory } from 'react-router-dom';
import { createStructuredSelector } from 'reselect';
import { Page } from '@patternfly/react-core';

import { selectors } from 'reducers';
import { actions as globalSearchActions } from 'reducers/globalSearch';

import LoadingSection from 'Components/PatternFly/LoadingSection';
import Notifications from 'Containers/Notifications';
import SearchModal from 'Containers/Search/SearchModal';
import UnreachableWarning from 'Containers/UnreachableWarning';
import AppWrapper from 'Containers/AppWrapper';
import VersionOutOfDate from 'Containers/VersionOutOfDate';
import Body from 'Containers/MainPage/Body';
import useFeatureFlags from 'hooks/useFeatureFlags';
import usePermissions from 'hooks/usePermissions';

import CredentialExpiryBanner from './CredentialExpiryBanner';
import Masthead from './Header/Masthead';
import NavigationSidebar from './Sidebar/NavigationSidebar';

const mainPageSelector = createStructuredSelector({
    isGlobalSearchView: selectors.getGlobalSearchView,
    metadata: selectors.getMetadata,
    publicConfig: selectors.getPublicConfig,
    serverState: selectors.getServerState,
});

function MainPage(): ReactElement {
    const {
        isGlobalSearchView,
        metadata = {
            stale: false,
        },
        publicConfig,
        serverState,
    } = useSelector(mainPageSelector);

    // Follow-up: Replace SearchModal with path like /main/search and component like GlobalSearchPage.
    const dispatch = useDispatch();
    const history = useHistory();
    function onCloseGlobalSearchModal(toURL) {
        dispatch(globalSearchActions.toggleGlobalSearchView());
        if (typeof toURL === 'string') {
            history.push(toURL);
        }
    }

    const { isFeatureFlagEnabled, isLoadingFeatureFlags } = useFeatureFlags();
    const { hasReadAccess, hasReadWriteAccess, isLoadingPermissions } = usePermissions();

    // Render Body and NavigationSideBar only when feature flags and permissions are available.
    if (isLoadingFeatureFlags || isLoadingPermissions) {
        return <LoadingSection message="Loading..." />;
    }

    const hasServiceIdentityWritePermission = hasReadWriteAccess('ServiceIdentity');

    return (
        <AppWrapper publicConfig={publicConfig}>
            <div className="flex flex-1 flex-col h-full relative">
                <UnreachableWarning serverState={serverState} />
                <Notifications />
                <CredentialExpiryBanner
                    component="CENTRAL"
                    hasServiceIdentityWritePermission={hasServiceIdentityWritePermission}
                />
                <CredentialExpiryBanner
                    component="SCANNER"
                    hasServiceIdentityWritePermission={hasServiceIdentityWritePermission}
                />
                {metadata?.stale && <VersionOutOfDate />}
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
                {isGlobalSearchView && <SearchModal onClose={onCloseGlobalSearchModal} />}
            </div>
        </AppWrapper>
    );
}

export default MainPage;
