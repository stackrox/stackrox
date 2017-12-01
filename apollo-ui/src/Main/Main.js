import React, { Component } from 'react';
import Tabs from '../Components/Tabs';
import Pills from '../Components/Pills';
import TabContent from '../Components/TabContent';
import logo from '../logo.svg';

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
                <section className="flex flex-1 text-grey-dark">
                    <nav className="flex w-1 bg-blue-lightest md:w-1/6 border-r border-gray-light">
                        <ul className="flex flex-col list-reset p-0 w-full font-mono font-bold">
                            <li className="flex"><a className="flex no-underline p-6 w-full hover:underline text-grey" href="#dashboard">Dashboard</a></li>
                            <li className="flex"><a className="flex no-underline p-6 w-full hover:underline text-grey" href="#violations">Violations</a></li>
                            <li className="flex"><a className="flex no-underline p-6 w-full hover:underline text-grey" href="#compliance">Compliance</a></li>
                            <li className="flex"><a className="flex no-underline p-6 w-full hover:underline text-grey" href="#policies">Policies</a></li>
                            <li className="flex"><a className="flex no-underline p-6 w-full hover:underline text-grey" href="#integrations">Integrations</a></li>
                        </ul>
                    </nav>
                    <main className="flex flex-1 flex-col bg-white md:w-5/6">
                        <header className="flex w-full h-16 bg-white border-b border-gray-light p-4">
                            <div className="flex self-center md:w-1/6">
                                <h1 className="text-lg">Reports</h1>
                            </div>
                            <div className="flex self-center justify-start md:w-2/3">
                                <input className="appearance-none border rounded w-full py-2 px-3 border-gray-light"
                                       placeholder="Scope by resource type:Registry"/>
                            </div>
                            <div className="flex self-center justify-end md:w-1/6">
                                <div className="px-3">
                                    <div className="relative">
                                        <select className="block appearance-none w-full bg-grey-lighter border border-gray-light text-grey-darker py-2 px-4 pr-8 rounded">
                                            <option>Last 24 Hours</option>
                                            <option>Last Week</option>
                                            <option>Last Month</option>
                                            <option>Last Year</option>
                                        </select>
                                        <div className="pointer-events-none absolute pin-y pin-r flex items-center px-2 border-grey-lighter">
                                            <svg className="h-4 w-4" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20"><path d="M9.293 12.95l.707.707L15.657 8l-1.414-1.414L10 10.828 5.757 6.586 4.343 8z"/></svg>
                                        </div>
                                    </div>
                                </div>
                            </div>
                        </header>
                        <section className="flex flex-1 p-4">
                            <Tabs header="Policies,Compliance">
                                <TabContent name="Policies">
                                    <Pills data={[{ text: 'All' }, { text: 'Image Assurance' }, { text: 'Configurations' }, { text: 'Orchestrator Target' }, { text: 'Denial of Policy' }, { text: 'Privileges & Capabilities' }, { text: 'Account Authorization' }]}></Pills>
                                </TabContent>
                                <TabContent name="Compliance">
                                
                                </TabContent>
                            </Tabs>
                        </section>
                    </main>
                </section>
            </section>
        );
    }
}

export default Main;
