import React, { Component } from 'react';
import {
    BrowserRouter as Router,
    Route,
    Redirect,
    Link
} from 'react-router-dom';

import logo from 'images/logo.svg';

import ViolationsPage from 'Containers/ViolationsPage';

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
                    <section className="flex flex-1 text-grey-dark">
                        <nav className="flex w-1 bg-blue-lightest md:w-1/6 border-r border-gray-light">
                            <ul className="flex flex-col list-reset p-0 w-full font-mono font-bold">
                                <li className="flex">
                                    <a className="flex p-6 w-full">
                                        <Link to="/dashboard" className="no-underline hover:underline text-grey">Dashboard</Link> 
                                    </a>
                                </li>
                                <li className="flex">
                                    <a className="flex p-6 w-full">
                                        <Link to="/violations" className="no-underline hover:underline text-grey">Violations</Link>
                                    </a>
                                </li>
                                <li className="flex">
                                    <a className="flex p-6 w-full">
                                        <Link to="/compliance" className="no-underline hover:underline text-grey">Compliance</Link>
                                    </a>
                                </li>
                                <li className="flex">
                                    <a className="flex p-6 w-full">
                                        <Link to="/policies" className="no-underline hover:underline text-grey">Policies</Link>
                                    </a>
                                </li>
                                <li className="flex">
                                    <a className="flex p-6 w-full">
                                        <Link to="/integrations" className="no-underline hover:underline text-grey">Integrations</Link>
                                    </a>
                                </li>
                            </ul>
                        </nav>
                        <main className="flex flex-1 flex-col bg-white md:w-5/6">
                            {/* Redirects to a default path */}
                            <Redirect from="/" to="/violations" />
                            <Route exact path="/violations" component={ViolationsPage} />
                        </main>
                    </section>
                </Router>
            </section>
        );
    }
}

export default Main;
