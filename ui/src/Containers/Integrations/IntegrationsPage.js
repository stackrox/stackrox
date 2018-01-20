import React, { Component } from 'react';
import { ToastContainer, toast } from 'react-toastify';

import IntegrationModal from 'Containers/Integrations/IntegrationModal';
import IntegrationTile from 'Containers/Integrations/IntegrationTile';
import ClustersModal from 'Containers/Integrations/ClustersModal';

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
        },
        {
            label: 'Openshift Container Registry',
            type: 'openshift',
            source: 'registries',
            image: openshift,
            disabled: true
        },
        {
            label: 'Google Container Registry',
            type: 'google',
            source: 'registries',
            image: google,
            disabled: true
        },
        {
            label: 'Azure Container Registry',
            type: 'azure',
            source: 'registries',
            image: azure,
            disabled: true
        },
        {
            label: 'Artifactory',
            type: 'artifactory',
            source: 'registries',
            image: artifactory,
            disabled: true
        }
    ],
    orchestratorsAndContainerPlatforms: [
        {
            label: 'Docker Enterprise Edition',
            image: dockerEnt,
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
        },
        {
            label: 'Red Hat OpenShift',
            image: openshift,
            disabled: false,
            clusterType: 'OPENSHIFT_CLUSTER'
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
        },
        {
            label: 'Qualys',
            type: 'qualys',
            source: 'scanners',
            image: qualys,
            disabled: true
        },
        {
            label: 'Grafeas',
            type: 'grafeas',
            source: 'scanners',
            image: grafeas,
            disabled: true
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
        },
        {
            label: 'Pagerduty',
            type: 'pagerduty',
            source: 'notifiers',
            image: pagerduty,
            disabled: true
        },
        {
            label: 'Splunk',
            type: 'splunk',
            source: 'notifiers',
            image: splunk,
            disabled: true
        },
        {
            label: 'ServiceNow',
            type: 'servicenow',
            source: 'notifiers',
            image: servicenow,
            disabled: true
        }
    ]
};


const reducer = (action, prevState, nextState) => {
    switch (action) {
        case 'OPEN_INTEGRATION_MODAL':
            return {
                integrationModal: {
                    open: true,
                    integration: nextState.integration,
                    source: nextState.source
                }
            };
        case 'CLOSE_INTEGRATION_MODAL':
            return {
                integrationModal: {
                    open: false,
                    integration: {},
                    source: {}
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
                integration: {},
                source: {}
            },
            clustersModal: {
                open: false,
                clusters: []
            },
            notifiers: [],
            clusters: [],
            scanners: [],
            registries: []
        };
    }

    componentDidMount() {
        this.getEntities('notifiers');
        this.getEntities('scanners');
        this.getEntities('registries');
        this.getEntities('clusters');
    }

    getEntities(source) {
        axios.get(`/v1/${source}`).then((response) => {
            const { [source]: entities } = response.data;
            this.update('UPDATE_ENTITIES', { entities, source });
        }).catch((error) => {
            console.error(error.response);
        });
    }

    openIntegrationModal = (integrationCategory) => {
        const { source, type } = integrationCategory;
        let integration = this.state[source].find(i => i.type === type.toLowerCase());
        if (!integration) integration = { type: type.toLowerCase(), source };
        this.update('OPEN_INTEGRATION_MODAL', { integration, source });
    }

    closeIntegrationModal = (isSuccessful) => {
        if (isSuccessful === true) {
            const {
                integrationModal: {
                    integration,
                    source
                }
            } = this.state;
            toast(`Successfully integrated ${integration.type}`);
            this.getEntities(source);
        }
        this.update('CLOSE_INTEGRATION_MODAL');
    }

    openClustersModal = (orchestrator) => {
        const { clusterType } = orchestrator;
        const clusters = this.state.clusters.filter(cluster => cluster.type === clusterType);
        this.update('OPEN_CLUSTERS_MODAL', { clusters, clusterType });
    }

    closeClustersModal = () => {
        this.update('CLOSE_CLUSTERS_MODAL');
    }

    update = (action, nextState) => {
        this.setState(prevState => reducer(action, prevState, nextState));
    }

    renderIntegrationModal() {
        const {
            integrationModal: {
                integration,
                source,
                open
            }
        } = this.state;
        if (!open) return null;
        return (
            <IntegrationModal
                integration={integration}
                source={source}
                onRequestClose={this.closeIntegrationModal}
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
                            {
                                dataSources.registries.map(registry => (
                                    <IntegrationTile
                                        key={registry.label}
                                        integration={registry}
                                        onClick={this.openIntegrationModal}
                                    />))
                            }
                        </div>
                    </div>
                    <div>
                        <h2 className="mx-3 mt-8 text-xl text-base text-primary-500 border-t border-primary-300 pt-6 pb-3">
                            Orchestrators &amp; Container Platforms
                        </h2>
                        <div className="flex">
                            {
                                dataSources.orchestratorsAndContainerPlatforms.map(orchestrator => (
                                    <IntegrationTile
                                        key={orchestrator.label}
                                        integration={orchestrator}
                                        onClick={this.openClustersModal}
                                    />))
                            }
                        </div>
                    </div>

                    <div className="mb-6">
                        <h2 className="mx-3 mt-8 text-xl text-base text-primary-500 border-t border-primary-300 pt-6 pb-3">
                            Scanning &amp; Governance Tools
                        </h2>
                        <div className="flex flex-wrap">
                            {
                                dataSources.scanningAndGovernanceTools.map(tool => (
                                    <IntegrationTile
                                        key={tool.label}
                                        integration={tool}
                                        onClick={this.openIntegrationModal}
                                    />))
                            }
                        </div>
                    </div>

                    <div className="mb-6">
                        <h2 className="mx-3 mt-8 text-xl text-base text-primary-500 border-t border-primary-300 pt-6 pb-3">
                            Plugins
                        </h2>
                        <div className="flex flex-wrap">
                            {
                                dataSources.plugins.map(plugin => (
                                    <IntegrationTile
                                        key={plugin.label}
                                        integration={plugin}
                                        onClick={this.openIntegrationModal}
                                    />))
                            }
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
