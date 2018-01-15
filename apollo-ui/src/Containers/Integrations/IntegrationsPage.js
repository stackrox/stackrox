import React, { Component } from 'react';
import { ToastContainer, toast } from 'react-toastify';

import IntegrationModal from 'Containers/Integrations/IntegrationModal';

import axios from 'axios';

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
import email from 'images/email.svg';

const dataSources = {
    registries: [
        {
            label: 'Docker Registry',
            type: 'docker',
            source: 'registry',
            image: docker,
            disabled: false
        },
        {
            label: 'Tenable Registry',
            type: 'tenable',
            source: 'registry',
            image: tenable,
            disabled: false
        },
        {
            label: 'Openshift Container Registry',
            type: 'openshift',
            source: 'registry',
            image: openshift,
            disabled: true
        },
        {
            label: 'Google Container Registry',
            type: 'google',
            source: 'registry',
            image: google,
            disabled: true
        },
        {
            label: 'Azure Container Registry',
            type: 'azure',
            source: 'registry',
            image: azure,
            disabled: true
        },
        {
            label: 'Artifactory',
            type: 'artifactory',
            source: 'registry',
            image: artifactory,
            disabled: true
        }
    ],
    orchestratorsAndContainerPlatforms: [
        {
            label: 'Docker Enterprise Edition',
            image: dockerEnt,
            disabled: false
        },
        {
            label: 'Kubernetes',
            image: kubernetes,
            disabled: true
        },
        {
            label: 'Docker Swarm',
            image: docker,
            disabled: true
        },
        {
            label: 'Red Hat Openshift',
            image: openshift,
            disabled: true
        }
    ],
    scanningAndGovernanceTools: [
        {
            label: 'Docker Trusted Registry',
            type: 'dtr',
            source: 'scanner',
            image: docker,
            disabled: false
        },
        {
            label: 'Tenable',
            type: 'tenable',
            source: 'scanner',
            image: tenable,
            disabled: false
        },
        {
            label: 'Qualys',
            type: 'qualys',
            source: 'scanner',
            image: qualys,
            disabled: true
        },
        {
            label: 'Grafeas',
            type: 'grafeas',
            source: 'scanner',
            image: grafeas,
            disabled: true
        }
    ],
    plugins: [
        {
            label: 'Slack',
            type: 'slack',
            source: 'plugin',
            image: slack,
            disabled: false
        },
        {
            label: 'Jira',
            type: 'jira',
            source: 'plugin',
            image: jira,
            disabled: false
        },
        {
            label: 'Email',
            type: 'email',
            source: 'plugin',
            image: email,
            disabled: false
        },
        {
            label: 'Pagerduty',
            type: 'pagerduty',
            source: 'plugin',
            image: pagerduty,
            disabled: true
        },
        {
            label: 'Splunk',
            type: 'splunk',
            source: 'splunk',
            image: splunk,
            disabled: true
        },
        {
            label: 'ServiceNow',
            type: 'servicenow',
            source: 'plugin',
            image: servicenow,
            disabled: true
        }
    ]
};

const reducer = (action, prevState, nextState) => {
    switch (action) {
        case 'OPEN_MODAL':
            return {
                isModalOpen: true,
                integration: nextState.integration,
                source: nextState.source
            };
        case 'CLOSE_MODAL':
            return { isModalOpen: false, integration: {}, source: '' };
        case 'UPDATE_NOTIFIERS':
            return { notifiers: nextState.notifiers };
        default:
            return prevState;
    }
};

class IntegrationsPage extends Component {
    constructor(props) {
        super(props);

        this.state = {
            isModalOpen: false,
            integration: {},
            source: '',
            notifiers: []
        };
    }

    componentDidMount() {
        this.getNotifiers();
    }

    getNotifiers = () => {
        axios.get('/v1/notifiers').then((response) => {
            const { notifiers } = response.data;
            this.update('UPDATE_NOTIFIERS', { notifiers });
        }).catch((error) => {
            console.error(error.response);
        });
    }

    openModal = (type, source) => {
        let integration = this.state.notifiers.find(i => i.type === type.toLowerCase());
        if (!integration) integration = { type: type.toLowerCase(), source };
        this.update('OPEN_MODAL', { integration, source });
    }

    closeModal = (isSuccessful) => {
        if (isSuccessful === true) {
            toast(`Successfully integrated ${this.state.integration.type}`);
            this.getNotifiers();
        }
        this.update('CLOSE_MODAL');
    }

    update = (action, nextState) => {
        this.setState(prevState => reducer(action, prevState, nextState));
    }

    renderModal = () => {
        if (!this.state.isModalOpen) return '';
        return (
            <IntegrationModal integration={this.state.integration} source={this.state.source} isOpen={this.state.isModalOpen} onRequestClose={this.closeModal} />
        );
    }

    render() {
        return (
            <section className="flex">
                <ToastContainer toastClassName="font-sans text-base-600 text-white font-600 bg-black" hideProgressBar autoClose={3000} />
                <div className="md:w-full border-r border-primary-300 pt-4">
                    <h1 className="font-500 mx-3 border-b border-primary-300 pb-4 uppercase text-xl font-800 text-primary-600 tracking-wide">Data sources</h1>
                    <div>
                        <h2 className="mx-3 mt-8 text-xl text-base text-primary-500 pb-3">Registries</h2>
                        <div className="flex flex-wrap">
                            {
                                dataSources.registries.map((registry) => {
                                    this.func = () =>
                                        this.openModal(registry.type, registry.source);
                                    return (
                                        <div className="p-3 w-1/4" key={registry.label}>
                                            <button
                                                className={`w-full p-4 bg-white rounded-sm shadow text-center ${(registry.disabled) ? 'disabled' : ''}`}
                                                onClick={this.func}
                                            >
                                                <img className="w-24 h-24 mb-4" src={registry.image} alt={registry.label} />
                                                <div className="font-bold text-xl pt-4  border-t border-base-200">
                                                    {registry.label}
                                                </div>
                                            </button>
                                        </div>
                                    );
                                })
                            }
                        </div>
                    </div>
                    <div>
                        <h2 className="mx-3 mt-8 text-xl text-base text-primary-500 border-t border-primary-300 pt-6 pb-3">
                            Orchestrators & Container Platforms
                        </h2>
                        <div className="flex">
                            {
                                dataSources.orchestratorsAndContainerPlatforms.map(orchestrator => (
                                    <div className="p-3 w-1/4" key={orchestrator.label}>
                                        <div className={`p-4 bg-white rounded-sm shadow text-center ${(orchestrator.disabled) ? 'disabled' : ''}`}>
                                            <img className="w-24 h-24 mb-4" src={orchestrator.image} alt={orchestrator.label} />
                                            <div className="font-bold text-xl pt-4  border-t border-base-200">
                                                {orchestrator.label}
                                            </div>
                                        </div>
                                    </div>
                                ))
                            }
                        </div>
                    </div>

                    <div className="mb-6">
                        <h2 className="mx-3 mt-8 text-xl text-base text-primary-500 border-t border-primary-300 pt-6 pb-3">
                            Scanning & Governance Tools
                        </h2>
                        <div className="flex flex-wrap">
                            {
                                dataSources.scanningAndGovernanceTools.map((tool) => {
                                    this.func = () => this.openModal(tool.type, tool.source);
                                    return (
                                        <div className="p-3 w-1/4" key={tool.label}>
                                            <button
                                                className={`w-full p-4 bg-white rounded-sm shadow text-center ${(tool.disabled) ? 'disabled' : ''}`}
                                                onClick={this.func}
                                            >
                                                <img className="w-24 h-24 mb-4" src={tool.image} alt={tool.label} />
                                                <div className="font-bold text-xl pt-4  border-t border-base-200">
                                                    {tool.label}
                                                </div>
                                            </button>
                                        </div>
                                    );
                                })
                            }
                        </div>
                    </div>

                    <div className="mb-6">
                        <h2 className="mx-3 mt-8 text-xl text-base text-primary-500 border-t border-primary-300 pt-6 pb-3">
                            Plugins
                        </h2>
                        <div className="flex flex-wrap">
                            {
                                dataSources.plugins.map((plugin) => {
                                    this.func = () => this.openModal(plugin.type, plugin.source);
                                    return (
                                        <div className="p-3 w-1/4" key={plugin.label}>
                                            <button
                                                className={`w-full p-4 bg-white rounded-sm hover:shadow-lg shadow text-center ${(plugin.disabled) ? 'disabled' : ''}`}
                                                onClick={this.func}
                                            >
                                                <img className="w-24 h-24 mb-4" src={plugin.image} alt={plugin.label} />
                                                <div className="font-bold text-xl pt-4  border-t border-base-200">
                                                    {plugin.label}
                                                </div>
                                            </button>
                                        </div>
                                    );
                                })
                            }
                        </div>
                    </div>
                </div>
                {this.renderModal()}
            </section>
        );
    }
}

export default IntegrationsPage;
