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
    compliance2Path,
    integrationsPath,
    policiesPath,
    riskPath,
    imagesPath,
    secretsPath,
    apidocsPath,
    accessControlPath
} from 'routePaths';
import { selectors } from 'reducers';
import { actions as globalSearchActions } from 'reducers/globalSearch';
import { actions as cliSearchActions } from 'reducers/cli';

import asyncComponent from 'Components/AsyncComponent';
import ProtectedRoute from 'Components/ProtectedRoute';
import Notifications from 'Containers/Notifications';
import TopNavigation from 'Containers/Navigation/TopNavigation';
import LeftNavigation from 'Containers/Navigation/LeftNavigation';
import SearchModal from 'Containers/Search/SearchModal';
import CLIModal from 'Containers/CLI/CLIModal';

import ErrorBoundary from 'Containers/ErrorBoundary';

import CSSGrid from 'Containers/CSSGrid';

const AsyncApiDocsPage = asyncComponent(() => import('Containers/Docs/ApiPage'));
const AsyncDashboardPage = asyncComponent(() => import('Containers/Dashboard/DashboardPage'));
const AsyncNetworkPage = asyncComponent(() => import('Containers/Network/NetworkPage'));
const AsyncIntegrationsPage = asyncComponent(() =>
    import('Containers/Integrations/IntegrationsPage')
);
const AsyncViolationsPage = asyncComponent(() => import('Containers/Violations/ViolationsPage'));
const AsyncPoliciesPage = asyncComponent(() => import('Containers/Policies/Page'));
const AsyncImagesPage = asyncComponent(() => import('Containers/Images/ImagesPage'));
const AsyncCompliancePage = asyncComponent(() => import('Containers/Compliance/CompliancePage'));
const AsyncCompliance2Page = asyncComponent(() => import('Containers/Compliance2/Page'));

const AsyncRiskPage = asyncComponent(() => import('Containers/Risk/RiskPage'));
const AsyncSecretsPage = asyncComponent(() => import('Containers/Secrets/SecretsPage'));
const AsyncAccessControlPage = asyncComponent(() => import('Containers/AccessControl/Page'));

class MainPage extends Component {
    static propTypes = {
        history: ReactRouterPropTypes.history.isRequired,
        toggleGlobalSearchView: PropTypes.func.isRequired,
        toggleCLIDownloadView: PropTypes.func.isRequired,
        globalSearchView: PropTypes.bool.isRequired,
        cliDownloadView: PropTypes.bool.isRequired
    };

    onSearchCloseHandler = toURL => {
        this.props.toggleGlobalSearchView();
        if (toURL && typeof toURL === 'string') this.props.history.push(toURL);
    };

    onCLICloseHandler = toURL => {
        this.props.toggleCLIDownloadView();
        if (toURL && typeof toURL === 'string') this.props.history.push(toURL);
    };

    renderSearchModal = () => {
        if (!this.props.globalSearchView) return '';
        return <SearchModal className="h-full w-full" onClose={this.onSearchCloseHandler} />;
    };

    renderCLIDownload = () => {
        if (!this.props.cliDownloadView) return '';
        return <CLIModal className="h-full w-full" onClose={this.onCLICloseHandler} />;
    };

    renderRouter = () => (
        <section className="flex-auto w-full overflow-hidden">
            <ErrorBoundary>
                <Switch>
                    <ProtectedRoute
                        devOnly
                        path={compliance2Path}
                        component={AsyncCompliance2Page}
                    />
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
                    <ProtectedRoute devOnly path="/main/test" component={CSSGrid} />
                    <Redirect from={mainPath} to={dashboardPath} />
                </Switch>
            </ErrorBoundary>
        </section>
    );

    render() {
        return (
            <section className="flex flex-1 flex-col h-full relative">
                <Notifications />
                <div className="navigation-gradient" />
                <header className="flex z-1">
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
    cliDownloadView: selectors.getCLIDownloadView
});

const mapDispatchToProps = dispatch => ({
    toggleGlobalSearchView: () => dispatch(globalSearchActions.toggleGlobalSearchView()),
    toggleCLIDownloadView: () => dispatch(cliSearchActions.toggleCLIDownloadView())
});

export default connect(
    mapStateToProps,
    mapDispatchToProps
)(MainPage);
