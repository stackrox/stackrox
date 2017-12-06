import React, { Component } from 'react';
import {
    BrowserRouter as Router,
    Route,
    Redirect,
    Switch,
    Link
} from 'react-router-dom';

import logo from 'images/logo.svg';

import ViolationsPage from 'Containers/ViolationsPage';
import MainSidePanel from 'Containers/MainSidePanel';

class Main extends Component {
    render() {
        return (
            <section className="flex flex-1 flex-col">
                <header className="flex h-16 bg-blue-lightest border-b border-gray-light">
                    <div className="flex p-6 self-center">
                        <img src={logo} className="h-8" alt="logo" />
                    </div>
                    <div className="flex flex-1"></div>
                    <div className="flex self-center">
                        <img className="block h-12 rounded-full mx-4" src="https://loremflickr.com/320/320?lock=4" alt="" />
                    </div>
                </header>
                <Router>
                    <section className="flex flex-1 text-grey-dark relative">
                        <nav className="flex w-1 bg-blue-lightest md:w-1/6 border-r border-gray-light">
                            <ul className="flex flex-col list-reset p-0 w-full font-mono font-bold">
                                <li className="flex">
                                    <Link to="/dashboard" className="flex p-6 w-full no-underline hover:underline text-grey">Dashboard</Link>
                                </li>
                                <li className="flex">
                                    <Link to="/violations" className="flex p-6 w-full no-underline hover:underline text-grey">Violations</Link>
                                </li>
                                <li className="flex">
                                    <Link to="/integrations" className="flex p-6 w-full no-underline hover:underline text-grey">Integrations</Link>
                                </li>
                            </ul>
                        </nav>
                        <main className="flex flex-1 flex-col bg-white md:w-5/6">
                            {/* Redirects to a default path */}
                            <Switch>
                                <Route exact path="/violations" component={ViolationsPage} />
                                <Redirect from="/" to="/violations" />
                            </Switch>
                        </main>
                        <MainSidePanel></MainSidePanel>
                    </section>
                </Router>
            </section>
        );
    }
}

export default Main;
