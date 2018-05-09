import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { ToastContainer, toast } from 'react-toastify';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';

import ClustersModal from 'Containers/Integrations/ClustersModal';
import IntegrationModal from 'Containers/Integrations/IntegrationModal';
import IntegrationTile from 'Containers/Integrations/IntegrationTile';
import { actions as authActions } from 'reducers/auth';
import { actions as integrationActions } from 'reducers/integrations';
import { actions as clusterActions } from 'reducers/clusters';
import { selectors } from 'reducers';

import artifactory from 'images/artifactory.svg';
import auth0 from 'images/auth0.svg';
import clair from 'images/clair.svg';
import docker from 'images/docker.svg';
import email from 'images/email.svg';
import google from 'images/google-cloud.svg';
import jira from 'images/jira.svg';
import kubernetes from 'images/kubernetes.svg';
import openshift from 'images/openshift.svg';
import quay from 'images/quay.svg';
import slack from 'images/slack.svg';
import tenable from 'images/tenable.svg';

const integrationsList = {
    authProviders: [
        {
            label: 'Auth0',
            type: 'auth0',
            source: 'authProviders',
            image: auth0
        }
    ],
    imageIntegrations: [
        {
            label: 'Generic Docker Registry',
            type: 'docker',
            categories: 'Registry',
            source: 'imageIntegrations',
            image: docker,
            disabled: false
        },
        {
            label: 'Docker Trusted Registry',
            type: 'dtr',
            categories: 'Registry + Scanner',
            source: 'imageIntegrations',
            image: docker,
            disabled: false
        },
        {
            label: 'Tenable.io',
            type: 'tenable',
            categories: 'Registry + Scanner',
            source: 'imageIntegrations',
            image: tenable,
            disabled: false
        },
        {
            label: 'JFrog Artifactory',
            type: 'artifactory',
            categories: 'Registry',
            source: 'imageIntegrations',
            image: artifactory,
            disabled: false
        },
        {
            label: 'Quay.io',
            type: 'quay',
            categories: 'Registry + Scanner',
            source: 'imageIntegrations',
            image: quay,
            disabled: false
        },
        {
            label: 'CoreOS Clair',
            type: 'clair',
            categories: 'Scanner',
            source: 'imageIntegrations',
            image: clair,
            disabled: false
        },
        {
            label: 'Google Cloud',
            type: 'google',
            categories: 'Registry + Scanner',
            source: 'imageIntegrations',
            image: google,
            disabled: false
        }
    ],
    orchestrators: [
        {
            label: 'Kubernetes',
            image: kubernetes,
            source: 'clusters',
            type: 'KUBERNETES_CLUSTER'
        },
        {
            label: 'OpenShift',
            image: openshift,
            source: 'clusters',
            type: 'OPENSHIFT_CLUSTER'
        },
        {
            label: 'Docker Enterprise Edition',
            image: docker,
            source: 'clusters',
            type: 'DOCKER_EE_CLUSTER'
        },
        {
            label: 'Docker Swarm',
            image: docker,
            source: 'clusters',
            type: 'SWARM_CLUSTER'
        }
    ],
    plugins: [
        {
            label: 'Slack',
            type: 'slack',
            source: 'notifiers',
            image: slack
        },
        {
            label: 'Jira',
            type: 'jira',
            source: 'notifiers',
            image: jira
        },
        {
            label: 'Email',
            type: 'email',
            source: 'notifiers',
            image: email
        },
        {
            label: 'Google Cloud SCC',
            type: 'cscc',
            source: 'notifiers',
            image: google
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
        default:
            return prevState;
    }
};

class IntegrationsPage extends Component {
    static propTypes = {
        /* eslint-disable */
        authProviders: PropTypes.arrayOf(
            PropTypes.shape({
                config: PropTypes.shape({
                    audience: PropTypes.string.isRequired,
                    client_id: PropTypes.string.isRequired,
                    domain: PropTypes.string.isRequired
                }),
                name: PropTypes.string.isRequired
            })
        ).isRequired,
        clusters: PropTypes.arrayOf(PropTypes.object).isRequired,
        notifiers: PropTypes.arrayOf(PropTypes.object).isRequired,
        imageIntegrations: PropTypes.arrayOf(PropTypes.object).isRequired,
        /* eslint-enable */
        selectedClusterType: PropTypes.string,
        selectClusterType: PropTypes.func.isRequired,
        closeClustersModal: PropTypes.func.isRequired,
        fetchAuthProviders: PropTypes.func.isRequired,
        fetchNotifiers: PropTypes.func.isRequired,
        fetchImageIntegrations: PropTypes.func.isRequired,
        fetchClusters: PropTypes.func.isRequired
    };

    static defaultProps = {
        selectedClusterType: null
    };

    constructor(props) {
        super(props);

        this.state = {
            integrationModal: {
                open: false,
                integrations: [],
                source: '',
                type: ''
            }
        };
    }

    getEntities = source => {
        switch (source) {
            case 'authProviders':
                this.props.fetchAuthProviders();
                break;
            case 'imageIntegrations':
                this.props.fetchImageIntegrations();
                break;
            case 'notifiers':
                this.props.fetchNotifiers();
                break;
            case 'clusters':
                this.props.fetchClusters();
                break;
            default:
                throw new Error(`Unknown source ${source}`);
        }
    };

    getClustersForOrchestrator = orchestrator => {
        const { type } = orchestrator;
        const clusters = this.props.clusters.filter(cluster => cluster.type === type);
        return clusters;
    };

    openIntegrationModal = integrationCategory => {
        if (integrationCategory.source === 'clusters') {
            this.props.selectClusterType(integrationCategory.type);
        } else {
            const { source, type } = integrationCategory;
            const integrations =
                source !== 'clusters'
                    ? this.props[source].filter(i => i.type === type.toLowerCase())
                    : this.props.clusters.filter(cluster => cluster.type === type);
            this.update('OPEN_INTEGRATION_MODAL', { integrations, source, type });
        }
    };

    closeIntegrationModal = isSuccessful => {
        if (isSuccessful === true) {
            const { integrationModal: { source, type } } = this.state;
            toast(`Successfully integrated ${type}`);
            this.getEntities(source);
        }
        this.update('CLOSE_INTEGRATION_MODAL');
    };

    findIntegrations = (source, type) => {
        const integrations = this.props[source].filter(i => i.type === type.toLowerCase());
        return integrations.filter(obj => obj.type === type);
    };

    update = (action, nextState) => {
        this.setState(prevState => reducer(action, prevState, nextState));
    };

    renderClustersModal() {
        const { selectedClusterType } = this.props;
        if (!selectedClusterType) return null;
        const clusters = this.props.clusters.filter(
            cluster => cluster.type === selectedClusterType
        );
        return (
            <ClustersModal
                clusters={clusters}
                selectedClusterType={selectedClusterType}
                onRequestClose={this.props.closeClustersModal}
            />
        );
    }

    renderIntegrationModal() {
        const { integrationModal: { source, type, open } } = this.state;
        if (!open) return null;
        const integrations =
            source !== 'clusters'
                ? this.props[source].filter(i => i.type === type.toLowerCase())
                : this.props.clusters.filter(cluster => cluster.type === type);
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

    renderIntegrationTiles = source =>
        integrationsList[source].map(tile => (
            <IntegrationTile
                key={tile.label}
                integration={tile}
                onClick={this.openIntegrationModal}
                numIntegrations={
                    source !== 'orchestrators'
                        ? this.findIntegrations(tile.source, tile.type).length
                        : this.getClustersForOrchestrator(tile).length
                }
            />
        ));

    render() {
        const imageIntegrations = this.renderIntegrationTiles('imageIntegrations');
        const orchestrators = this.renderIntegrationTiles('orchestrators');
        const plugins = this.renderIntegrationTiles('plugins');
        const authProviders = this.renderIntegrationTiles('authProviders');
        return (
            <section className="flex">
                <ToastContainer
                    toastClassName="font-sans text-base-600 text-white font-600 bg-black"
                    hideProgressBar
                    autoClose={3000}
                />
                <div className="md:w-full border-r border-primary-300 pt-4 bg-base-100">
                    <h1 className="font-500 mx-3 border-b border-primary-300 pb-4 uppercase text-xl font-800 text-primary-600 tracking-wide">
                        Integrations
                    </h1>
                    <div>
                        <h2 className="mx-3 mt-8 text-xl text-base text-primary-500 pb-3">
                            Image Integrations
                        </h2>
                        <div className="flex flex-wrap">{imageIntegrations}</div>
                    </div>
                    <div>
                        <h2 className="mx-3 mt-8 text-xl text-base text-primary-500 border-t border-primary-300 pt-6 pb-3">
                            Orchestrators &amp; Container Platforms
                        </h2>
                        <div className="flex flex-wrap">{orchestrators}</div>
                    </div>
                    <div className="mb-6">
                        <h2 className="mx-3 mt-8 text-xl text-base text-primary-500 border-t border-primary-300 pt-6 pb-3">
                            Plugins
                        </h2>
                        <div className="flex flex-wrap">{plugins}</div>
                    </div>
                    <div className="mb-6">
                        <h2 className="mx-3 mt-8 text-xl text-base text-primary-500 border-t border-primary-300 pt-6 pb-3">
                            Authentication Providers
                        </h2>
                        <div className="flex flex-wrap">{authProviders}</div>
                    </div>
                </div>
                {this.renderIntegrationModal()}
                {this.renderClustersModal()}
            </section>
        );
    }
}

const mapStateToProps = createStructuredSelector({
    authProviders: selectors.getAuthProviders,
    clusters: selectors.getClusters,
    selectedClusterType: selectors.getSelectedClusterType,
    notifiers: selectors.getNotifiers,
    imageIntegrations: selectors.getImageIntegrations
});

const mapDispatchToProps = dispatch => ({
    fetchAuthProviders: () => dispatch(authActions.fetchAuthProviders.request()),
    fetchNotifiers: () => dispatch(integrationActions.fetchNotifiers.request()),
    fetchImageIntegrations: () => dispatch(integrationActions.fetchImageIntegrations.request()),
    fetchRegistries: () => dispatch(integrationActions.fetchRegistries.request()),
    fetchScanners: () => dispatch(integrationActions.fetchScanners.request()),
    fetchClusters: () => dispatch(clusterActions.fetchClusters.request()),
    closeClustersModal: () => dispatch(clusterActions.selectClusterType(null)),
    selectClusterType: clusterType => dispatch(clusterActions.selectClusterType(clusterType))
});

export default connect(mapStateToProps, mapDispatchToProps)(IntegrationsPage);
