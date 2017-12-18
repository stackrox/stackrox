import React from 'react';
import qualys from 'images/qualys.svg';
import artifactory from 'images/artifactory.svg';
import azure from 'images/azure.svg';
import dockerEnt from 'images/docker-ent.svg';
import docker from 'images/docker.svg';
import google from 'images/google.svg';
import grafeas from 'images/grafeas.svg';
import jira from 'images/jira.svg';
import kubernetes from 'images/kubernetes.svg';
import openshift from 'images/openshift.svg';
import pagerduty from 'images/pagerduty.svg';
import slack from 'images/slack.svg';
import tenable from 'images/tenable.svg';
import servicenow from 'images/servicenow.svg';
import splunk from 'images/splunk.svg';

const IntegrationsContainer = () => (
    <section className="flex">
        <div className="md:w-2/3 border-r border-primary-300 pt-4">
            <h1 className="font-500 mx-3 border-b border-primary-300 pb-4 uppercase text-xl font-800 text-primary-600 tracking-wide">Data sources</h1>
            <div>
                <h2 className="mx-3 mt-8 text-xl text-base text-primary-500 pb-3">Registries</h2>
                <div className="flex flex-wrap">
                    <div className="p-3 w-1/3">
                        <div className="p-4 bg-white rounded-sm shadow text-center ">
                            <img className="w-24 h-24 mb-4" src={docker} alt="Docker Trusted Registry" />
                            <div className="font-bold text-xl pt-4  border-t border-base-200">
                                Docker Trusted Registry
                            </div>
                        </div>
                    </div>

                    <div className="p-3 w-1/3">
                        <div className="p-4 bg-white rounded-sm shadow text-center ">
                            <img className="w-24 h-24 mb-4" src={docker} alt="Docker Hub" />
                            <div className="font-bold text-xl pt-4  border-t border-base-200">
                                Docker Hub
                            </div>
                        </div>
                    </div>

                    <div className="p-3 w-1/3">
                        <div className="p-4 bg-white rounded-sm shadow disabled text-center ">
                            <img className="w-24 h-24 mb-4" src={openshift} alt="Openshift" />
                            <div className="font-bold text-xl pt-4  border-t border-base-200">
                                Openshift Container Registry
                            </div>
                        </div>
                    </div>

                    <div className="p-3 w-1/3">
                        <div className="p-4 bg-white rounded-sm shadow disabled text-center ">
                            <img className="w-24 h-24 mb-4" src={google} alt="Google Container Registry" />
                            <div className="font-bold text-xl pt-4  border-t border-base-200">
                                Google Container Registry
                            </div>
                        </div>
                    </div>

                    <div className="p-3 w-1/3">
                        <div className="p-4 bg-white rounded-sm shadow disabled text-center ">
                            <img className="w-24 h-24 mb-4" src={azure} alt="Azure Container Registry" />
                            <div className="font-bold text-xl pt-4  border-t border-base-200">
                                Azure Container Registry
                            </div>
                        </div>
                    </div>

                    <div className="p-3 w-1/3">
                        <div className="p-4 bg-white rounded-sm shadow disabled text-center ">
                            <img className="w-24 h-24 mb-4" src={artifactory} alt="Artifactory" />
                            <div className="font-bold text-xl pt-4  border-t border-base-200">
                                Artifactory
                            </div>
                        </div>
                    </div>
                </div>
            </div>
            <div>
                <h2 className="mx-3 mt-8 text-xl text-base text-primary-500 border-t border-primary-300 pt-6 pb-3">
                    Orchestrators & Container Platforms
                </h2>
                <div className="flex">

                    <div className="p-3 flex-1">
                        <div className="p-4 bg-white rounded-sm shadow text-center ">
                            <img className="w-24 h-24 mb-4" src={dockerEnt} alt="Docker Enterprise Edition" />
                            <div className="font-bold text-xl pt-4  border-t border-base-200">
                                Docker Enterprise Edition
                            </div>
                        </div>
                    </div>

                    <div className="p-3 flex-1">
                        <div className="p-4 bg-white rounded-sm shadow disabled text-center ">
                            <img className="w-24 h-24 mb-4" src={kubernetes} alt="Kubernetes" />
                            <div className="font-bold text-xl pt-4  border-t border-base-200">
                                Kubernetes
                            </div>
                        </div>
                    </div>

                    <div className="p-3 flex-1">
                        <div className="p-4 bg-white rounded-sm shadow disabled text-center ">
                            <img className="w-24 h-24 mb-4" src={docker} alt="Docker Swarm" />
                            <div className="font-bold text-xl pt-4  border-t border-base-200">
                                Docker Swarm
                            </div>
                        </div>
                    </div>

                    <div className="p-3 flex-1">
                        <div className="p-4 bg-white rounded-sm shadow disabled text-center ">
                            <img className="w-24 h-24 mb-4" src={openshift} alt="Openshift" />
                            <div className="font-bold text-xl pt-4  border-t border-base-200">
                                Red Hat Openshift
                            </div>
                        </div>
                    </div>
                </div>
            </div>

            <div className="mb-6">
                <h2 className="mx-3 mt-8 text-xl text-base text-primary-500 border-t border-primary-300 pt-6 pb-3">
                    Scanning & Governance Tools
                </h2>
                <div className="flex flex-wrap">
                    <div className="p-3 w-1/3">
                        <div className="p-4 bg-white rounded-sm shadow disabled text-center ">
                            <img className="w-24 h-24 mb-4" src={tenable} alt="Tenable" />
                            <div className="font-bold text-xl pt-4  border-t border-base-200">
                                Tenable
                            </div>
                        </div>
                    </div>

                    <div className="p-3 w-1/3">
                        <div className="p-4 bg-white rounded-sm shadow disabled text-center ">
                            <img className="w-24 h-24 mb-4" src={qualys} alt="Aqua" />
                            <div className="font-bold text-xl pt-4  border-t border-base-200">
                                qualys
                            </div>
                        </div>
                    </div>

                    <div className="p-3 w-1/3">
                        <div className="p-4 bg-white rounded-sm shadow disabled text-center ">
                            <img className="w-24 h-24 mb-4" src={grafeas} alt="Grafeas" />
                            <div className="font-bold text-xl pt-4  border-t border-base-200">
                                Grafeas
                            </div>
                        </div>
                    </div>
                </div>
            </div>

            <div />
        </div>
        <aside className="flex-col h-full bg-primary-100 md:w-1/3 pt-4">
            <h1 className="font-500 mx-3 border-b border-primary-300 pb-4 uppercase text-xl font-800 text-primary-600 tracking-wide">Workflow</h1>
            <h2 className="mx-3 mt-8 text-xl text-base text-primary-500 pb-3">
                Plugins
            </h2>
            <div>
                <div className="flex flex-wrap">
                    <div className="p-3 w-1/2">
                        <div className="p-4 bg-white rounded-sm shadow disabled text-center ">
                            <img className="w-24 h-24 mb-4" src={slack} alt="Slack" />
                            <div className="font-bold text-xl pt-4  border-t border-base-200">
                                Slack
                            </div>
                        </div>
                    </div>

                    <div className="p-3 w-1/2">
                        <div className="p-4 bg-white rounded-sm shadow disabled text-center ">
                            <img className="w-24 h-24 mb-4" src={pagerduty} alt="Pagerduty" />
                            <div className="font-bold text-xl pt-4  border-t border-base-200">
                                Pagerduty
                            </div>
                        </div>
                    </div>

                    <div className="p-3 w-1/2">
                        <div className="p-4 bg-white rounded-sm shadow disabled text-center ">
                            <img className="w-24 h-24 mb-4" src={jira} alt="Jira" />
                            <div className="font-bold text-xl pt-4  border-t border-base-200">
                                Jira
                            </div>
                        </div>
                    </div>
                    <div className="p-3 w-1/2">
                        <div className="p-4 bg-white rounded-sm shadow disabled text-center ">
                            <img className="w-24 h-24 mb-4" src={splunk} alt="Splunk" />
                            <div className="font-bold text-xl pt-4  border-t border-base-200">
                                Splunk
                            </div>
                        </div>
                    </div>
                    <div className="p-3 w-1/2">
                        <div className="p-4 bg-white rounded-sm shadow disabled text-center ">
                            <img className="w-24 h-24 mb-4" src={servicenow} alt="ServiceNow" />
                            <div className="font-bold text-xl pt-4  border-t border-base-200">
                                ServiceNow
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </aside>
    </section>
);

export default IntegrationsContainer;
