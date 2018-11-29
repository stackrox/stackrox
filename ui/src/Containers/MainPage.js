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
    apidocsPath
} from 'routePaths';
import { selectors } from 'reducers';
import { actions as globalSearchActions } from 'reducers/globalSearch';

import asyncComponent from 'Components/AsyncComponent';
import ProtectedRoute from 'Components/ProtectedRoute';
import Notifications from 'Containers/Notifications';
import TopNavigation from 'Containers/Navigation/TopNavigation';
import LeftNavigation from 'Containers/Navigation/LeftNavigation';
import SearchModal from 'Containers/Search/SearchModal';
import ErrorBoundary from 'Containers/ErrorBoundary';

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
const AsyncRiskPage = asyncComponent(() => import('Containers/Risk/RiskPage'));
const AsyncSecretsPage = asyncComponent(() => import('Containers/Secrets/SecretsPage'));

class MainPage extends Component {
    static propTypes = {
        history: ReactRouterPropTypes.history.isRequired,
        toggleGlobalSearchView: PropTypes.func.isRequired,
        globalSearchView: PropTypes.bool.isRequired
    };

    onCloseHandler = toURL => {
        this.props.toggleGlobalSearchView();
        if (toURL && typeof toURL === 'string') this.props.history.push(toURL);
    };

    renderSearchModal = () => {
        if (!this.props.globalSearchView) return '';
        return <SearchModal className="h-full w-full" onClose={this.onCloseHandler} />;
    };

    renderRouter = () => (
        <section className="flex-auto w-full overflow-hidden">
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
                    <ProtectedRoute path={apidocsPath} component={AsyncApiDocsPage} />
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
            </section>
        );
    }
}

const mapStateToProps = createStructuredSelector({
    globalSearchView: selectors.getGlobalSearchView
});

const mapDispatchToProps = dispatch => ({
    toggleGlobalSearchView: () => dispatch(globalSearchActions.toggleGlobalSearchView())
});

export default connect(
    mapStateToProps,
    mapDispatchToProps
)(MainPage);
