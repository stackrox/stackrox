import React, { Component } from 'react';
import Tabs from '../Components/Tabs';
import Pills from '../Components/Pills';
import TabContent from '../Components/TabContent';
import * as Icon from "react-feather";
import Logo from "../Components/icons/logo";



class Main extends Component {
    render() {
        return <section className="flex h-full flex-col">
            <header className="flex bg-primary-500 justify-between px-4 py-2">
              <div className="flex self-center">
                <Logo className="fill-current text-white h-10 w-10" />
              </div>
              <div className="flex self-center">
                <img className="block h-8 rounded-full" src="https://loremflickr.com/320/320?lock=4" alt="" />
              </div>
            </header>
            <section className="flex flex-1">
              <nav id="mainNav" className="flex z-10 absolute pin md:overflow-hidden md:overflow-y-scroll md:relative bg-primary-100 w-full md:w-1/4 lg:w-1/6 border-r">
                <ul className="p-0 w-full uppercase text-xs font-700">
                  <li>
                    <a className="no-underline block p-4 w-full" href="#dashboard">
                      Dashboard
                    </a>
                  </li>
                  <li>
                    <a className="no-underline block p-4 w-full" href="#violations">
                      Violations
                    </a>
                  </li>
                  <li>
                    <a className="no-underline block p-4 w-full" href="#policies">
                      Policies
                    </a>
                  </li>
                  <li>
                    <a className="no-underline block p-4 w-full" href="#integrations">
                      Integrations
                    </a>
                  </li>
                </ul>
              </nav>
              <main className="w-full md:w-3/4 lg:w-5/6 md:overflow-hidden md:overflow-y-scroll">
                <section className="flex p-3">
                  <Tabs header="Policies,Compliance">
                    <TabContent name="Policies">
                      <header className="flex w-full my-3">
                        <div className="flex-auto">
                          <input className="border rounded w-full p-2" placeholder="Scope by resource type:Registry" />
                        </div>
                        <div className="">
                          <div className="relative ml-3">
                            <select className="block appearance-none w-full border py-2 px-4 pr-8 rounded">
                              <option>Last 24 Hours</option>
                              <option>Last Week</option>
                              <option>Last Month</option>
                              <option>Last Year</option>
                            </select>
                            <div className="pointer-events-none absolute pin-y pin-r flex items-center px-2">
                              <Icon.ChevronDown className="h-5 w-5" />
                            </div>
                          </div>
                        </div>
                      </header>
                      <Pills data={[{ text: "All" }, { text: "Image Assurance" }, { text: "Configurations" }, { text: "Orchestrator Target" }, { text: "Denial of Policy" }, { text: "Privileges & Capabilities" }, { text: "Account Authorization" }]} />
                      <p>hello</p>
                      <p>hello</p>
                      <p>
                        hello
                      </p> <p>hello</p> <p>hello</p> <p>hello</p> <p>
                        hello
                      </p>
                      <p>hello</p>
                      <p>
                        <p>hello</p> <p>hello</p> <p>hello</p> <p>
                          hello
                        </p>
                        hello
                      </p> <p>hello</p> <p>hello</p> <p>hello</p> <p>
                        hello
                      </p> <p>hello</p> <p>hello</p> <p>hello</p> <p>
                        hello
                      </p> <p>hello</p> <p>hello</p> <p>hello</p> <p>
                        hello
                      </p> <p>hello</p> <p>hello</p> <p>hello</p> <p>
                        hello
                      </p> <p>hello</p> <p>hello</p> <p>hello</p> <p>
                        hello
                      </p> <p>hello</p> <p>hello</p>
                    </TabContent>
                    <TabContent name="Compliance" />
                  </Tabs>
                </section>
              </main>
            </section>
          </section>;
    }
}

export default Main;
