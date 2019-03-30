import React, { Component } from 'react';
import PropTypes from 'prop-types';
import ReactRouterPropTypes from 'react-router-prop-types';
import { Redirect, Switch } from 'react-router-dom';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';

import {
    mainPath,
    dashboardPath,
    networkPath,
    violationsPath,
    compliancePath,
    integrationsPath,
    policiesPath,
    riskPath,
    imagesPath,
    secretsPath,
    apidocsPath,
    accessControlPath,
    licensePath
} from 'routePaths';
import { selectors } from 'reducers';
import { actions as globalSearchActions } from 'reducers/globalSearch';
import { actions as cliSearchActions } from 'reducers/cli';

import asyncComponent from 'Components/AsyncComponent';
import Button from 'Components/Button';
import ProtectedRoute from 'Components/ProtectedRoute';
import Notifications from 'Containers/Notifications';
import TopNavigation from 'Containers/Navigation/TopNavigation';
import LeftNavigation from 'Containers/Navigation/LeftNavigation';
import SearchModal from 'Containers/Search/SearchModal';
import CLIModal from 'Containers/CLI/CLIModal';
import LicenseReminder from 'Containers/License/LicenseReminder';

import ErrorBoundary from 'Containers/ErrorBoundary';
import UnreachableWarning from 'Containers/UnreachableWarning';
import Loader from 'Components/Loader';

const AsyncApiDocsPage = asyncComponent(() => import('Containers/Docs/ApiPage'));
const AsyncDashboardPage = asyncComponent(() => import('Containers/Dashboard/DashboardPage'));
const AsyncNetworkPage = asyncComponent(() => import('Containers/Network/Page'));
const AsyncIntegrationsPage = asyncComponent(() =>
    import('Containers/Integrations/IntegrationsPage')
);
const AsyncViolationsPage = asyncComponent(() => import('Containers/Violations/ViolationsPage'));
const AsyncPoliciesPage = asyncComponent(() => import('Containers/Policies/Page'));
const AsyncImagesPage = asyncComponent(() => import('Containers/Images/ImagesPage'));
const AsyncCompliancePage = asyncComponent(() => import('Containers/Compliance/Page'));
const AsyncRiskPage = asyncComponent(() => import('Containers/Risk/RiskPage'));
const AsyncSecretsPage = asyncComponent(() => import('Containers/Secrets/SecretsPage'));
const AsyncAccessControlPage = asyncComponent(() => import('Containers/AccessControl/Page'));
const AsyncLicensePage = asyncComponent(() => import('Containers/License/Page'));

class MainPage extends Component {
    static propTypes = {
        history: ReactRouterPropTypes.history.isRequired,
        toggleGlobalSearchView: PropTypes.func.isRequired,
        toggleCLIDownloadView: PropTypes.func.isRequired,
        globalSearchView: PropTypes.bool.isRequired,
        cliDownloadView: PropTypes.bool.isRequired,
        metadata: PropTypes.shape({ stale: PropTypes.bool.isRequired }),
        pdfLoadingStatus: PropTypes.bool
    };

    static defaultProps = {
        metadata: { stale: false },
        pdfLoadingStatus: false
    };

    onSearchCloseHandler = toURL => {
        this.props.toggleGlobalSearchView();
        if (toURL && typeof toURL === 'string') this.props.history.push(toURL);
    };

    onCLICloseHandler = toURL => {
        this.props.toggleCLIDownloadView();
        if (toURL && typeof toURL === 'string') this.props.history.push(toURL);
    };

    renderPDFLoader = () =>
        this.props.pdfLoadingStatus && (
            <div className="absolute pin-l pin-t bg-tertiary-300 z-60 mt-20 w-full h-full text-tertiary-800">
                <Loader message="Exporting..." />
            </div>
        );

    renderSearchModal = () => {
        if (!this.props.globalSearchView) return '';
        return <SearchModal className="h-full w-full" onClose={this.onSearchCloseHandler} />;
    };

    renderCLIDownload = () => {
        if (!this.props.cliDownloadView) return '';
        return <CLIModal className="h-full w-full" onClose={this.onCLICloseHandler} />;
    };

    renderRouter = () => (
        <section
            className={`flex-auto w-full relative ${
                this.props.pdfLoadingStatus ? '' : 'overflow-auto'
            }`}
        >
            <ErrorBoundary>
                <Switch>
                    <ProtectedRoute path={dashboardPath} component={AsyncDashboardPage} />
                    <ProtectedRoute path={networkPath} component={AsyncNetworkPage} />
                    <ProtectedRoute path={violationsPath} component={AsyncViolationsPage} />
                    <ProtectedRoute path={compliancePath} component={AsyncCompliancePage} />
                    <ProtectedRoute path={integrationsPath} component={AsyncIntegrationsPage} />
                    <ProtectedRoute path={policiesPath} component={AsyncPoliciesPage} />
                    <ProtectedRoute path={riskPath} component={AsyncRiskPage} />
                    <ProtectedRoute path={imagesPath} component={AsyncImagesPage} />
                    <ProtectedRoute path={secretsPath} component={AsyncSecretsPage} />
                    <ProtectedRoute path={accessControlPath} component={AsyncAccessControlPage} />
                    <ProtectedRoute path={apidocsPath} component={AsyncApiDocsPage} />
                    {process.env.REACT_APP_ROX_LICENSE_ENFORCEMENT === 'true' && (
                        <ProtectedRoute path={licensePath} component={AsyncLicensePage} />
                    )}
                    <Redirect from={mainPath} to={dashboardPath} />
                </Switch>
                {this.renderPDFLoader()}
            </ErrorBoundary>
        </section>
    );

    windowReloadHandler = () => {
        window.location.reload();
    };

    renderVersionOutOfDate = () => {
        if (!this.props.metadata.stale) return null;
        return (
            <div className="flex w-full items-center p-3 bg-warning-200 text-warning-800 border-b border-base-400 justify-center font-700">
                <span>
                    It looks like this page is out of date and may not behave properly. Please{' '}
                    <Button
                        text="refresh this page"
                        className="text-tertiary-700 hover:text-tertiary-800 underline font-700 justify-center"
                        onClick={this.windowReloadHandler}
                    />{' '}
                    to correct any issues.
                </span>
            </div>
        );
    };

    render() {
        return (
            <section className="flex flex-1 flex-col h-full relative">
                <Notifications />
                {process.env.REACT_APP_ROX_LICENSE_ENFORCEMENT === 'true' && <LicenseReminder />}
                <div className="navigation-gradient" />
                {this.renderVersionOutOfDate()}
                <UnreachableWarning />
                <header className="flex z-20">
                    <TopNavigation />
                </header>
                <section className="flex flex-1 flex-row">
                    <LeftNavigation />
                    {this.renderRouter()}
                </section>
                {this.renderSearchModal()}
                {this.renderCLIDownload()}
            </section>
        );
    }
}

const mapStateToProps = createStructuredSelector({
    globalSearchView: selectors.getGlobalSearchView,
    cliDownloadView: selectors.getCLIDownloadView,
    metadata: selectors.getMetadata,
    pdfLoadingStatus: selectors.getPdfLoadingStatus
});

const mapDispatchToProps = dispatch => ({
    toggleGlobalSearchView: () => dispatch(globalSearchActions.toggleGlobalSearchView()),
    toggleCLIDownloadView: () => dispatch(cliSearchActions.toggleCLIDownloadView())
});

export default connect(
    mapStateToProps,
    mapDispatchToProps
)(MainPage);
