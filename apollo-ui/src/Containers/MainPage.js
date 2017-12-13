import React, { Component } from 'react';
import {
    BrowserRouter as Router,
    Route,
    Redirect,
    Switch,
    Link
} from 'react-router-dom';
import Logo from 'Components/icons/logo';
import * as Icon from 'react-feather';

import PolicyAlertsSidePanel from 'Containers/Violations/Policies/PolicyAlertsSidePanel';
import IntegrationsPage from 'Containers/Integrations/IntegrationsPage';
import ViolationsPage from 'Containers/Violations/ViolationsPage';

class Main extends Component {
    render() {
        return <section className="flex flex-1 flex-col h-full">
              <header className="flex bg-primary-600 justify-between px-3">
                <div className="flex">
                  <div className="flex self-center">
                    <Logo className="fill-current text-white h-10 w-10 mr-3" />
                  </div>
                  <nav>
                    <ul className="flex list-reset flex-1 uppercase text-sm tracking-wide">
                      <li>
                        <Link to="/dashboard" className="flex border-l px-4 no-underline py-5 pb-4 text-base-600 text-white hover:bg-primary-700 disabled items-center">
                          <span><Icon.BarChart className="h-4 w-4 mr-3" /></span>
                          <span>Dashboard</span>
                        </Link>
                      </li>
                      <li>
                        <Link to="/violations" className="flex border-l border-primary-400 px-4 no-underline py-5 pb-4 text-base-600 hover:bg-primary-700 text-white items-center">
                          <span><Icon.AlertTriangle className="h-4 w-4 mr-3" /></span>
                          <span>Violations</span>
                        </Link>
                      </li>
                      <li>
                        <Link to="/integrations" className="flex border-l border-r border-primary-400 px-4 no-underline py-5 pb-4 text-base-600 hover:bg-primary-700 text-white items-center">
                          <span><Icon.PlusCircle className="h-4 w-4 mr-3" /></span>
                          <span>Integrations</span>
                        </Link>
                      </li>
                    </ul>
                  </nav>
                </div>
                <div className="flex self-center">
                  <img className="block h-8 rounded-full" src="https://loremflickr.com/320/320?lock=4" alt="User profile" />
                </div>
              </header>
              <Router>
              <section className="flex flex-1 bg-base-100">
                <main className="overflow-y-scroll w-full">
                  {/* Redirects to a default path */}
                  <Switch>
                    <Route exact path="/violations" component={ViolationsPage} />
                    <Route exact path="/integrations" component={IntegrationsPage} />
                    <Redirect from="/" to="/violations" />
                  </Switch>
                </main>
                <PolicyAlertsSidePanel></PolicyAlertsSidePanel>
              </section>
                </Router>
            </section>
        
    }
}

export default Main;
