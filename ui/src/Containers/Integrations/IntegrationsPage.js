import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';

import ClustersModal from 'Containers/Integrations/Clusters/ClustersModal';
import integrationsList from 'Containers/Integrations/integrationsList';
import IntegrationModal from 'Containers/Integrations/IntegrationModal';
import IntegrationTile from 'Containers/Integrations/IntegrationTile';
import { actions as authActions } from 'reducers/auth';
import { actions as apiTokenActions } from 'reducers/apitokens';
import { actions as integrationActions } from 'reducers/integrations';
import { actions as clusterActions } from 'reducers/clusters';
import { selectors } from 'reducers';
import APITokensModal from './APITokens/APITokensModal';

class IntegrationsPage extends Component {
    static propTypes = {
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
        apiTokens: PropTypes.arrayOf(
            PropTypes.shape({
                name: PropTypes.string.isRequired,
                role: PropTypes.string.isRequired
            })
        ).isRequired,
        clusters: PropTypes.arrayOf(PropTypes.object).isRequired,
        dnrIntegrations: PropTypes.arrayOf(PropTypes.object).isRequired,
        notifiers: PropTypes.arrayOf(PropTypes.object).isRequired,
        imageIntegrations: PropTypes.arrayOf(PropTypes.object).isRequired,
        fetchAuthProviders: PropTypes.func.isRequired,
        fetchAPITokens: PropTypes.func.isRequired,
        fetchDNRIntegrations: PropTypes.func.isRequired,
        fetchNotifiers: PropTypes.func.isRequired,
        fetchImageIntegrations: PropTypes.func.isRequired,
        fetchClusters: PropTypes.func.isRequired
    };

    state = {
        modalOpen: false,
        selectedSource: '',
        selectedType: ''
    };

    getEntities = (source, type) => {
        switch (source) {
            case 'authProviders':
                if (type === 'apitoken') {
                    this.props.fetchAPITokens();
                    break;
                }
                this.props.fetchAuthProviders();
                break;
            case 'dnrIntegrations':
                this.props.fetchDNRIntegrations();
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
        this.setState({
            modalOpen: true,
            selectedSource: integrationCategory.source,
            selectedType: integrationCategory.type
        });
    };

    fetchEntitiesAndCloseModal = () => {
        this.getEntities(this.state.selectedSource, this.state.selectedType);
        this.setState({
            modalOpen: false,
            selectedSource: '',
            selectedType: ''
        });
    };

    findIntegrations = (source, type) => {
        const typeLowerMatches = integration => integration.type === type.toLowerCase();

        switch (source) {
            case 'clusters':
                return this.getClustersForOrchestrator(type);
            case 'dnrIntegrations':
                return this.props.dnrIntegrations;
            case 'authProviders':
                if (type === 'apitoken') {
                    return this.props.apiTokens;
                }
                return this.props.authProviders.filter(typeLowerMatches);
            case 'notifiers':
                return this.props.notifiers.filter(typeLowerMatches);
            case 'imageIntegrations':
                return this.props.imageIntegrations.filter(typeLowerMatches);
            default:
                throw new Error(`Unknown source ${source}`);
        }
    };

    renderClustersModal() {
        return (
            <ClustersModal
                clusterType={this.state.selectedType}
                onRequestClose={this.fetchEntitiesAndCloseModal}
            />
        );
    }

    renderAPITokensModal() {
        return (
            <APITokensModal
                tokens={this.props.apiTokens}
                onRequestClose={this.fetchEntitiesAndCloseModal}
            />
        );
    }

    renderIntegrationModal() {
        const { modalOpen, selectedSource, selectedType } = this.state;
        if (!modalOpen) return null;

        if (selectedSource === 'clusters') {
            return this.renderClustersModal();
        }

        if (selectedSource === 'authProviders' && selectedType === 'apitoken') {
            return this.renderAPITokensModal();
        }

        const integrations = this.findIntegrations(selectedSource, selectedType);
        return (
            <IntegrationModal
                integrations={integrations}
                source={selectedSource}
                type={selectedType}
                onRequestClose={this.fetchEntitiesAndCloseModal}
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
        const dnrIntegrations = this.renderIntegrationTiles('dnrIntegrations');
        const imageIntegrations = this.renderIntegrationTiles('imageIntegrations');
        const orchestrators = this.renderIntegrationTiles('orchestrators');
        const plugins = this.renderIntegrationTiles('plugins');
        const authProviders = this.renderIntegrationTiles('authProviders');
        return (
            <section className="flex">
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
                    <div>
                        <h2 className="mx-3 mt-8 text-xl text-base text-primary-500 border-t border-primary-300 pt-6 pb-3">
                            StackRox
                        </h2>
                        <div className="flex flex-wrap">{dnrIntegrations}</div>
                    </div>
                </div>
                {this.renderIntegrationModal()}
            </section>
        );
    }
}

const mapStateToProps = createStructuredSelector({
    authProviders: selectors.getAuthProviders,
    apiTokens: selectors.getAPITokens,
    clusters: selectors.getClusters,
    dnrIntegrations: selectors.getDNRIntegrations,
    notifiers: selectors.getNotifiers,
    imageIntegrations: selectors.getImageIntegrations
});

const mapDispatchToProps = dispatch => ({
    fetchAuthProviders: () => dispatch(authActions.fetchAuthProviders.request()),
    fetchAPITokens: () => dispatch(apiTokenActions.fetchAPITokens.request()),
    fetchDNRIntegrations: () => dispatch(integrationActions.fetchDNRIntegrations.request()),
    fetchNotifiers: () => dispatch(integrationActions.fetchNotifiers.request()),
    fetchImageIntegrations: () => dispatch(integrationActions.fetchImageIntegrations.request()),
    fetchRegistries: () => dispatch(integrationActions.fetchRegistries.request()),
    fetchScanners: () => dispatch(integrationActions.fetchScanners.request()),
    fetchClusters: () => dispatch(clusterActions.fetchClusters.request())
});

export default connect(mapStateToProps, mapDispatchToProps)(IntegrationsPage);
