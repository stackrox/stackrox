import React, { ReactElement } from 'react';
import { useDispatch, useSelector } from 'react-redux';
import { useHistory } from 'react-router-dom';
import { createStructuredSelector } from 'reselect';
import { Page } from '@patternfly/react-core';

import { selectors } from 'reducers';
import { actions as globalSearchActions } from 'reducers/globalSearch';

import LoadingSection from 'Components/PatternFly/LoadingSection';
import Notifications from 'Containers/Notifications';
import NavigationSideBar from 'Containers/Navigation/NavigationSideBar';
import SearchModal from 'Containers/Search/SearchModal';
import UnreachableWarning from 'Containers/UnreachableWarning';
import AppWrapper from 'Containers/AppWrapper';
import CredentialExpiryBanners from 'Containers/CredentialExpiryBanners/CredentialExpiryBanners';
import VersionOutOfDate from 'Containers/VersionOutOfDate';
import Body from 'Containers/MainPage/Body';
import Masthead from 'Containers/MainPage/Masthead';

const mainPageSelector = createStructuredSelector({
    featureFlags: selectors.getFeatureFlags,
    isGlobalSearchView: selectors.getGlobalSearchView,
    isLoadingFeatureFlags: selectors.getIsLoadingFeatureFlags,
    isLoadingPermissions: selectors.getIsLoadingUserRolePermissions,
    metadata: selectors.getMetadata,
    publicConfig: selectors.getPublicConfig,
    serverState: selectors.getServerState,
});

function MainPage(): ReactElement {
    const {
        isGlobalSearchView,
        isLoadingFeatureFlags,
        isLoadingPermissions,
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
        dispatch(globalSearchActions.toggleGlobalSearchView);
        if (typeof toURL === 'string') {
            history.push(toURL);
        }
    }

    if (isLoadingFeatureFlags || isLoadingPermissions) {
        return <LoadingSection message="Loading..." />;
    }

    return (
        <AppWrapper publicConfig={publicConfig}>
            <div className="flex flex-1 flex-col h-full relative">
                <UnreachableWarning serverState={serverState} />
                <Notifications />
                <CredentialExpiryBanners />
                {metadata?.stale && <VersionOutOfDate />}
                <Page
                    mainContainerId="main-page-container"
                    header={<Masthead />}
                    isManagedSidebar
                    sidebar={<NavigationSideBar />}
                >
                    <Body />
                </Page>
                {isGlobalSearchView && <SearchModal onClose={onCloseGlobalSearchModal} />}
            </div>
        </AppWrapper>
    );
}

export default MainPage;
