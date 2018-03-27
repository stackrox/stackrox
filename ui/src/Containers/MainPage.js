import React, { Component } from 'react';
import ReactRouterPropTypes from 'react-router-prop-types';
import { Redirect, Switch, NavLink, withRouter } from 'react-router-dom';
import * as Icon from 'react-feather';

import ProtectedRoute from 'Components/ProtectedRoute';
import Logo from 'Components/icons/logo';
import DashboardPage from 'Containers/Dashboard/DashboardPage';
import IntegrationsPage from 'Containers/Integrations/IntegrationsPage';
import ViolationsPage from 'Containers/Violations/ViolationsPage';
import PoliciesPage from 'Containers/Policies/PoliciesPage';
import CompliancePage from 'Containers/Compliance/CompliancePage';
import RiskPage from 'Containers/Risk/RiskPage';
import AuthService from 'services/AuthService';

const navLinks = [
    {
        text: 'Dashboard',
        align: 'left',
        to: '/main/dashboard',
        renderIcon: () => <Icon.BarChart className="h-4 w-4 mr-3" />
    },
    {
        text: 'Violations',
        align: 'left',
        to: '/main/violations',
        renderIcon: () => <Icon.AlertTriangle className="h-4 w-4 mr-3" />
    },
    {
        text: 'Compliance',
        align: 'left',
        to: '/main/compliance',
        renderIcon: () => <Icon.CheckSquare className="h-4 w-4 mr-3" />
    },
    {
        text: 'Risk',
        align: 'left',
        to: '/main/risk',
        renderIcon: () => <Icon.Shield className="h-4 w-4 mr-3" />
    },
    {
        text: 'Policies',
        align: 'left',
        to: '/main/policies',
        renderIcon: () => <Icon.FileText className="h-4 w-4 mr-3" />
    },
    {
        text: 'Integrations',
        align: 'right',
        to: '/main/integrations',
        renderIcon: () => <Icon.PlusCircle className="h-4 w-4 mr-3" />
    }
];

class MainPage extends Component {
    static propTypes = {
        history: ReactRouterPropTypes.history.isRequired
    };

    renderLeftSideNavLinks = () => (
        <ul className="flex list-reset flex-1 uppercase text-sm tracking-wide">
            {navLinks.filter(obj => obj.align === 'left').map((navLink, i, arr) => (
                <li key={navLink.text}>
                    <NavLink
                        to={navLink.to}
                        className={`flex border-primary-400 px-4 no-underline py-5 pb-4 text-base-600 hover:text-primary-200 text-white items-center ${
                            i === arr.length - 1 ? 'border-l border-r' : 'border-l'
                        }`}
                        activeClassName="bg-primary-600"
                    >
                        <span>{navLink.renderIcon()}</span>
                        <span>{navLink.text}</span>
                    </NavLink>
                </li>
            ))}
        </ul>
    );

    renderRightSideNavLinks = () => (
        <ul className="flex list-reset flex-1 uppercase text-sm tracking-wide justify-end">
            {navLinks.filter(obj => obj.align === 'right').map(navLink => (
                <li key={navLink.text}>
                    <NavLink
                        to={navLink.to}
                        className="flex border-l border-primary-400 px-4 no-underline py-5 pb-4 text-base-600 hover:text-primary-200 text-white items-center"
                        activeClassName="bg-primary-600"
                    >
                        <span>{navLink.renderIcon()}</span>
                        <span>{navLink.text}</span>
                    </NavLink>
                </li>
            ))}
        </ul>
    );

    renderLogoutButton = () => {
        if (!AuthService.isLoggedIn()) return '';
        const logout = () => () => {
            AuthService.logout();
            this.props.history.push('/login');
        };
        return (
            <button
                onClick={logout()}
                className="flex border-l border-r border-primary-400 px-4 no-underline py-5 pb-4 text-base-600 hover:text-primary-200 text-white items-center"
            >
                <span>
                    <Icon.LogOut className="h-4 w-4 mr-3" />
                </span>
                <span>Logout</span>
            </button>
        );
    };

    render() {
        return (
            <section className="flex flex-1 flex-col h-full">
                <header className="flex bg-primary-500 justify-between">
                    <div className="flex flex-1">
                        <nav className="flex flex-row flex-1">
                            <div className="flex self-center">
                                <Logo className="fill-current text-white h-10 w-10 mx-3" />
                            </div>
                            {this.renderLeftSideNavLinks()}
                            {this.renderRightSideNavLinks()}
                            {this.renderLogoutButton()}
                        </nav>
                    </div>
                </header>
                <section className="flex flex-1 bg-base-100">
                    <main className="overflow-y-scroll w-full">
                        {/* Redirects to a default path */}
                        <Switch>
                            <ProtectedRoute path="/main/dashboard" component={DashboardPage} />
                            <ProtectedRoute
                                path="/main/violations/:alertId?"
                                component={ViolationsPage}
                            />
                            <ProtectedRoute path="/main/compliance" component={CompliancePage} />
                            <ProtectedRoute path="/main/risk" component={RiskPage} />
                            <ProtectedRoute
                                path="/main/integrations"
                                component={IntegrationsPage}
                            />
                            <ProtectedRoute path="/main/policies" component={PoliciesPage} />
                            <Redirect from="/main" to="/main/dashboard" />
                        </Switch>
                    </main>
                </section>
            </section>
        );
    }
}

export default withRouter(MainPage);
