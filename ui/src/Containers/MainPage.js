import React, { Component } from 'react';
import { Redirect, Switch } from 'react-router-dom';
import ProtectedRoute from 'Components/ProtectedRoute';
import DashboardPage from 'Containers/Dashboard/DashboardPage';
import IntegrationsPage from 'Containers/Integrations/IntegrationsPage';
import ViolationsPage from 'Containers/Violations/ViolationsPage';
import PoliciesPage from 'Containers/Policies/PoliciesPage';
import ImagesPage from 'Containers/Images/ImagesPage';
import CompliancePage from 'Containers/Compliance/CompliancePage';
import RiskPage from 'Containers/Risk/RiskPage';
import TopNavigation from 'Containers/Navigation/TopNavigation';
import LeftNavigation from 'Containers/Navigation/LeftNavigation';

class MainPage extends Component {
    renderRouter = () => (
        <section className="flex-auto overflow-auto">
            <Switch>
                <ProtectedRoute path="/main/dashboard" component={DashboardPage} />
                <ProtectedRoute path="/main/violations/:alertId?" component={ViolationsPage} />
                <ProtectedRoute path="/main/compliance/:clusterId?" component={CompliancePage} />
                <ProtectedRoute path="/main/imageintegrations" component={IntegrationsPage} />
                <ProtectedRoute path="/main/policies" component={PoliciesPage} />
                <ProtectedRoute path="/main/risk" component={RiskPage} />
                <ProtectedRoute path="/main/images" component={ImagesPage} />
                <Redirect from="/main" to="/main/dashboard" />
            </Switch>
        </section>
    );

    render() {
        return (
            <section className="flex flex-1 flex-col h-full">
                <div className="navigation-gradient" />
                <header className="flex">
                    <TopNavigation />
                </header>
                <section className="flex flex-1 flex-row">
                    <LeftNavigation />
                    {this.renderRouter()}
                </section>
            </section>
        );
    }
}

export default MainPage;
