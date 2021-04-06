import React, { ReactElement } from 'react';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { History } from 'history';

import { selectors } from 'reducers';
import { actions as globalSearchActions } from 'reducers/globalSearch';
import { actions as cliSearchActions } from 'reducers/cli';

import Notifications from 'Containers/Notifications';
import TopNavigation from 'Containers/Navigation/TopNavigation';
import LeftNavigation from 'Containers/Navigation/LeftNavigation';
import SearchModal from 'Containers/Search/SearchModal';
import CLIModal from 'Containers/CLI/CLIModal';
import UnreachableWarning from 'Containers/UnreachableWarning';
import VersionOutOfDate from 'Containers/VersionOutOfDate';
import Body from 'Containers/MainPage/Body';
import AppWrapper from 'Containers/AppWrapper';
import CredentialExpiryBanners from 'Containers/CredentialExpiry/CredentialExpiryBanners';

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
};

function MainPage({
    history,
    toggleGlobalSearchView,
    toggleCLIDownloadView,
    isGlobalSearchView,
    isCliDownloadView,
    metadata = {
        stale: false,
    },
}: MainPageProps): ReactElement {
    return (
        <AppWrapper>
            <div className="flex flex-1 flex-col h-full relative">
                <UnreachableWarning />
                <Notifications />
                <CredentialExpiryBanners />
                <div className="navigation-gradient" />
                {metadata?.stale && <VersionOutOfDate />}
                <header className="flex z-20 ignore-react-onclickoutside">
                    <TopNavigation />
                </header>
                <div className="flex flex-1 flex-row">
                    <LeftNavigation />
                    <Body />
                </div>
                {isGlobalSearchView && (
                    <SearchModal onClose={onCloseHandler(history, toggleGlobalSearchView)} />
                )}
                {isCliDownloadView && (
                    <CLIModal onClose={onCloseHandler(history, toggleCLIDownloadView)} />
                )}
            </div>
        </AppWrapper>
    );
}

const mapStateToProps = createStructuredSelector({
    isGlobalSearchView: selectors.getGlobalSearchView,
    isCliDownloadView: selectors.getCLIDownloadView,
    metadata: selectors.getMetadata,
    featureFlags: selectors.getFeatureFlags,
});

const mapDispatchToProps = {
    toggleGlobalSearchView: globalSearchActions.toggleGlobalSearchView,
    toggleCLIDownloadView: cliSearchActions.toggleCLIDownloadView,
};

export default connect(mapStateToProps, mapDispatchToProps)(MainPage);
