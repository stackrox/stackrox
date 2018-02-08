import React, { Component } from 'react';
import { ToastContainer, toast } from 'react-toastify';

import IntegrationModal from 'Containers/Integrations/IntegrationModal';
import IntegrationTile from 'Containers/Integrations/IntegrationTile';
import ClustersModal from 'Containers/Integrations/ClustersModal';

import axios from 'axios';

import auth0 from 'images/auth0.svg';
import docker from 'images/docker.svg';
import jira from 'images/jira.svg';
import kubernetes from 'images/kubernetes.svg';
import slack from 'images/slack.svg';
import tenable from 'images/tenable.svg';
import email from 'images/email.svg';

const dataSources = {
    authProviders: [
        {
            label: 'Auth0',
            type: 'auth0',
            source: 'authProviders',
            image: auth0,
            disabled: false
        }
    ],
    registries: [
        {
            label: 'Docker Registry',
            type: 'docker',
            source: 'registries',
            image: docker,
            disabled: false
        },
        {
            label: 'Tenable Registry',
            type: 'tenable',
            source: 'registries',
            image: tenable,
            disabled: false
        }
    ],
    orchestratorsAndContainerPlatforms: [
        {
            label: 'Docker Enterprise Edition',
            image: docker,
            disabled: false,
            clusterType: 'DOCKER_EE_CLUSTER'
        },
        {
            label: 'Kubernetes',
            image: kubernetes,
            disabled: false,
            clusterType: 'KUBERNETES_CLUSTER'
        },
        {
            label: 'Docker Swarm',
            image: docker,
            disabled: false,
            clusterType: 'SWARM_CLUSTER'
        }
    ],
    scanningAndGovernanceTools: [
        {
            label: 'Docker Trusted Registry',
            type: 'dtr',
            source: 'scanners',
            image: docker,
            disabled: false
        },
        {
            label: 'Tenable',
            type: 'tenable',
            source: 'scanners',
            image: tenable,
            disabled: false
        }
    ],
    plugins: [
        {
            label: 'Slack',
            type: 'slack',
            source: 'notifiers',
            image: slack,
            disabled: false
        },
        {
            label: 'Jira',
            type: 'jira',
            source: 'notifiers',
            image: jira,
            disabled: false
        },
        {
            label: 'Email',
            type: 'email',
            source: 'notifiers',
            image: email,
            disabled: false
        }
    ]
};


const reducer = (action, prevState, nextState) => {
    switch (action) {
        case 'OPEN_INTEGRATION_MODAL':
            return {
                integrationModal: {
                    open: true,
                    integrations: nextState.integrations,
                    source: nextState.source,
                    type: nextState.type
                }
            };
        case 'CLOSE_INTEGRATION_MODAL':
            return {
                integrationModal: {
                    open: false,
                    integrations: [],
                    source: '',
                    type: ''
                }
            };
        case 'OPEN_CLUSTERS_MODAL':
            return {
                clustersModal: {
                    open: true,
                    clusters: nextState.clusters,
                    clusterType: nextState.clusterType
                }
            };
        case 'CLOSE_CLUSTERS_MODAL':
            return {
                clustersModal: {
                    open: false,
                    clusters: [],
                    clusterType: null
                }
            };
        case 'UPDATE_ENTITIES': {
            const { entities, source } = nextState;
            return { [source]: entities };
        }
        default:
            return prevState;
    }
};

class IntegrationsPage extends Component {
    constructor(props) {
        super(props);

        this.state = {
            integrationModal: {
                open: false,
                integrations: [],
                source: '',
                type: ''
            },
            clustersModal: {
                open: false,
                clusters: []
            },
            authProviders: [],
            clusters: [],
            notifiers: [],
            registries: [],
            scanners: []
        };
    }

    componentDidMount() {
        this.getEntities('authProviders');
        this.getEntities('clusters');
        this.getEntities('notifiers');
        this.getEntities('registries');
        this.getEntities('scanners');
    }

    getEntities = (source) => {
        axios.get(`/v1/${source}`).then((response) => {
            const { [source]: entities } = response.data;
            this.update('UPDATE_ENTITIES', { entities, source });
        }).catch((error) => {
            console.error(error.response);
        });
    }

    getClustersForOrchestrator = (orchestrator) => {
        const { clusterType } = orchestrator;
        const clusters = this.state.clusters.filter(cluster => cluster.type === clusterType);
        return clusters;
    }

    openIntegrationModal = (integrationCategory) => {
        const { source, type } = integrationCategory;
        const integrations = this.state[source].filter(i => i.type === type.toLowerCase());
        this.update('OPEN_INTEGRATION_MODAL', { integrations, source, type });
    }

    closeIntegrationModal = (isSuccessful) => {
        if (isSuccessful === true) {
            const {
                integrationModal: {
                    source,
                    type
                }
            } = this.state;
            toast(`Successfully integrated ${type}`);
            this.getEntities(source);
        }
        this.update('CLOSE_INTEGRATION_MODAL');
    }

    openClustersModal = (orchestrator) => {
        const clusters = this.getClustersForOrchestrator(orchestrator);
        const { clusterType } = orchestrator;
        this.update('OPEN_CLUSTERS_MODAL', { clusters, clusterType });
    }

    closeClustersModal = () => {
        this.update('CLOSE_CLUSTERS_MODAL');
    }

    findIntegrations = (source, type) => {
        const integrations = this.state[source];
        return integrations.filter(obj => obj.type === type);
    };

    update = (action, nextState) => {
        this.setState(prevState => reducer(action, prevState, nextState));
    }

    renderIntegrationModal() {
        const {
            integrationModal: {
                source,
                type,
                open
            }
        } = this.state;
        if (!open) return null;
        const integrations = this.state[source].filter(i => i.type === type.toLowerCase());
        return (
            <IntegrationModal
                integrations={integrations}
                source={source}
                type={type}
                onRequestClose={this.closeIntegrationModal}
                onIntegrationsUpdate={this.getEntities}
            />
        );
    }

    renderClustersModal() {
        const {
            clustersModal: {
                open,
                clusters,
                clusterType
            }
        } = this.state;
        if (!open) return null;
        return (
            <ClustersModal
                clusters={clusters}
                clusterType={clusterType}
                onRequestClose={this.closeClustersModal}
            />
        );
    }

    render() {
        const registries = dataSources.registries.map(registry => (
            <IntegrationTile
                key={registry.label}
                integration={registry}
                onClick={this.openIntegrationModal}
                disabled={registry.disabled}
                numIntegrations={this.findIntegrations(registry.source, registry.type).length}
            />
        ));

        const orchestrators = dataSources.orchestratorsAndContainerPlatforms.map((orchestrator) => {
            const clustersCount = this.getClustersForOrchestrator(orchestrator).length;
            return (
                <IntegrationTile
                    key={orchestrator.label}
                    integration={orchestrator}
                    onClick={this.openClustersModal}
                    disabled={clustersCount === 0}
                    numIntegrations={clustersCount}
                />
            );
        });

        const scanners = dataSources.scanningAndGovernanceTools.map(tool => (
            <IntegrationTile
                key={tool.label}
                integration={tool}
                onClick={this.openIntegrationModal}
                disabled={tool.disabled}
                numIntegrations={this.findIntegrations(tool.source, tool.type).length}
            />
        ));

        const plugins = dataSources.plugins.map(plugin => (
            <IntegrationTile
                key={plugin.label}
                integration={plugin}
                onClick={this.openIntegrationModal}
                disabled={plugin.disabled}
                numIntegrations={this.findIntegrations(plugin.source, plugin.type).length}
            />
        ));

        const authProviders = dataSources.authProviders.map(plugin => (
            <IntegrationTile
                key={plugin.label}
                integration={plugin}
                onClick={this.openIntegrationModal}
                disabled={plugin.disabled}
                numIntegrations={this.findIntegrations(plugin.source, plugin.type).length}
            />
        ));

        return (
            <section className="flex">
                <ToastContainer toastClassName="font-sans text-base-600 text-white font-600 bg-black" hideProgressBar autoClose={3000} />
                <div className="md:w-full border-r border-primary-300 pt-4">
                    <h1 className="font-500 mx-3 border-b border-primary-300 pb-4 uppercase text-xl font-800 text-primary-600 tracking-wide">Data sources</h1>
                    <div>
                        <h2 className="mx-3 mt-8 text-xl text-base text-primary-500 pb-3">
                            Registries
                        </h2>
                        <div className="flex flex-wrap">
                            {registries}
                        </div>
                    </div>
                    <div>
                        <h2 className="mx-3 mt-8 text-xl text-base text-primary-500 border-t border-primary-300 pt-6 pb-3">
                            Orchestrators &amp; Container Platforms
                        </h2>
                        <div className="flex">
                            {orchestrators}
                        </div>
                    </div>

                    <div className="mb-6">
                        <h2 className="mx-3 mt-8 text-xl text-base text-primary-500 border-t border-primary-300 pt-6 pb-3">
                            Scanning &amp; Governance Tools
                        </h2>
                        <div className="flex flex-wrap">
                            {scanners}
                        </div>
                    </div>

                    <div className="mb-6">
                        <h2 className="mx-3 mt-8 text-xl text-base text-primary-500 border-t border-primary-300 pt-6 pb-3">
                            Plugins
                        </h2>
                        <div className="flex flex-wrap">
                            {plugins}
                        </div>
                    </div>

                    <div className="mb-6">
                        <h2 className="mx-3 mt-8 text-xl text-base text-primary-500 border-t border-primary-300 pt-6 pb-3">
                            Authentication Providers
                        </h2>
                        <div className="flex flex-wrap">
                            {authProviders}
                        </div>
                    </div>
                </div>
                {this.renderIntegrationModal()}
                {this.renderClustersModal()}
            </section>
        );
    }
}

export default IntegrationsPage;
