import React from 'react';
import {
    BrowserRouter as Router,
    Route,
    Redirect,
    Switch,
    NavLink
} from 'react-router-dom';
import * as Icon from 'react-feather';

import Logo from 'Components/icons/logo';
import DashboardPage from 'Containers/Dashboard/DashboardPage';
import IntegrationsPage from 'Containers/Integrations/IntegrationsPage';
import ViolationsPage from 'Containers/Violations/ViolationsPage';

const Main = () => (
    <Router>
        <section className="flex flex-1 flex-col h-full">
            <header className="flex bg-primary-600 justify-between px-3">
                <div className="flex">
                    <div className="flex self-center">
                        <Logo className="fill-current text-white h-10 w-10 mr-3" />
                    </div>
                    <nav>
                        <ul className="flex list-reset flex-1 uppercase text-sm tracking-wide">
                            <li>
                                <NavLink to="/dashboard" className="flex border-l border-primary-400 px-4 no-underline py-5 pb-4 text-base-600 hover:text-primary-200 text-white items-center" activeClassName="bg-primary-800">
                                    <span>
                                        <Icon.BarChart className="h-4 w-4 mr-3" />
                                    </span>
                                    <span>Dashboard</span>
                                </NavLink>
                            </li>
                            <li>
                                <NavLink to="/violations" className="flex border-l border-primary-400 px-4 no-underline py-5 pb-4 text-base-600 hover:text-primary-200 text-white items-center" activeClassName="bg-primary-800">
                                    <span>
                                        <Icon.AlertTriangle className="h-4 w-4 mr-3" />
                                    </span>
                                    <span>Violations</span>
                                </NavLink>
                            </li>
                            <li>
                                <NavLink to="/integrations" className="flex border-l border-r border-primary-400 px-4 no-underline py-5 pb-4 text-base-600 hover:text-primary-200 text-white items-center" activeClassName="bg-primary-800">
                                    <span>
                                        <Icon.PlusCircle className="h-4 w-4 mr-3" />
                                    </span>
                                    <span>Integrations</span>
                                </NavLink>
                            </li>
                        </ul>
                    </nav>
                </div>
            </header>
            <section className="flex flex-1 bg-base-100">
                <main className="overflow-y-scroll w-full">
                    {/* Redirects to a default path */}
                    <Switch>
                        <Route exact path="/dashboard" component={DashboardPage} />
                        <Route exact path="/violations" component={ViolationsPage} />
                        <Route exact path="/integrations" component={IntegrationsPage} />
                        <Redirect from="/" to="/dashboard" />
                    </Switch>
                </main>
            </section>
        </section>
    </Router>
);

export default Main;
