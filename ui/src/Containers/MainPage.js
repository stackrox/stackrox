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
    secretsPath
} from 'routePaths';
import { selectors } from 'reducers';
import { actions as globalSearchActions } from 'reducers/globalSearch';

import ProtectedRoute from 'Components/ProtectedRoute';
import Notifications from 'Containers/Notifications';
import DashboardPage from 'Containers/Dashboard/DashboardPage';
import NetworkPage from 'Containers/Network/NetworkPage';
import IntegrationsPage from 'Containers/Integrations/IntegrationsPage';
import ViolationsPage from 'Containers/Violations/ViolationsPage';
import PoliciesPage from 'Containers/Policies/Page';
import ImagesPage from 'Containers/Images/ImagesPage';
import CompliancePage from 'Containers/Compliance/CompliancePage';
import RiskPage from 'Containers/Risk/RiskPage';
import SecretsPage from 'Containers/Secrets/SecretsPage';
import TopNavigation from 'Containers/Navigation/TopNavigation';
import LeftNavigation from 'Containers/Navigation/LeftNavigation';
import SearchModal from 'Containers/Search/SearchModal';
import ErrorBoundary from 'Containers/ErrorBoundary';

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
                    <ProtectedRoute path={dashboardPath} component={DashboardPage} />
                    <ProtectedRoute path={networkPath} component={NetworkPage} />
                    <ProtectedRoute path={violationsPath} component={ViolationsPage} />
                    <ProtectedRoute path={compliancePath} component={CompliancePage} />
                    <ProtectedRoute path={integrationsPath} component={IntegrationsPage} />
                    <ProtectedRoute path={policiesPath} component={PoliciesPage} />
                    <ProtectedRoute path={riskPath} component={RiskPage} />
                    <ProtectedRoute path={imagesPath} component={ImagesPage} />
                    <ProtectedRoute path={secretsPath} component={SecretsPage} />
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
