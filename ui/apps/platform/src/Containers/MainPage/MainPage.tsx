import React, { ReactElement } from 'react';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { History } from 'history';
import { Page } from '@patternfly/react-core';

import { selectors } from 'reducers';
import { actions as globalSearchActions } from 'reducers/globalSearch';

import Notifications from 'Containers/Notifications';
import NavigationSideBar from 'Containers/Navigation/NavigationSideBar';
import SearchModal from 'Containers/Search/SearchModal';
import UnreachableWarning, { ServerState } from 'Containers/UnreachableWarning';
import AppWrapper, { PublicConfig } from 'Containers/AppWrapper';
import CredentialExpiryBanners from 'Containers/CredentialExpiryBanners/CredentialExpiryBanners';
import VersionOutOfDate from 'Containers/VersionOutOfDate';
import Body from 'Containers/MainPage/Body';
import Masthead from 'Containers/MainPage/Masthead';

const onCloseHandler = (history, callBack) => (toURL) => {
    callBack();
    if (toURL && typeof toURL === 'string') {
        history.push(toURL);
    }
};

export type MainPageProps = {
    history: History;
    toggleGlobalSearchView: () => { type: string };
    toggleCLIDownloadView: () => { type: string };
    isGlobalSearchView: boolean;
    isCliDownloadView: boolean;
    metadata: {
        stale?: boolean;
    };
    publicConfig: PublicConfig;
    serverState: ServerState;
};

function MainPage({
    history,
    toggleGlobalSearchView,
    isGlobalSearchView,
    metadata = {
        stale: false,
    },
    publicConfig,
    serverState,
}: MainPageProps): ReactElement {
    const Header = <Masthead />;

    return (
        <AppWrapper publicConfig={publicConfig}>
            <div className="flex flex-1 flex-col h-full relative">
                <UnreachableWarning serverState={serverState} />
                <Notifications />
                <CredentialExpiryBanners />
                {metadata?.stale && <VersionOutOfDate />}
                <Page
                    mainContainerId="main-page-container"
                    header={Header}
                    isManagedSidebar
                    sidebar={<NavigationSideBar />}
                >
                    <Body />
                </Page>
                {isGlobalSearchView && (
                    <SearchModal onClose={onCloseHandler(history, toggleGlobalSearchView)} />
                )}
            </div>
        </AppWrapper>
    );
}

const mapStateToProps = createStructuredSelector({
    isGlobalSearchView: selectors.getGlobalSearchView,
    metadata: selectors.getMetadata,
    publicConfig: selectors.getPublicConfig,
    serverState: selectors.getServerState,
});

const mapDispatchToProps = {
    toggleGlobalSearchView: globalSearchActions.toggleGlobalSearchView,
};

export default connect(mapStateToProps, mapDispatchToProps)(MainPage);
