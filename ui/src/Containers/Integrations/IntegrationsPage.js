import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import PageHeader from 'Components/PageHeader';

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
                    client_id: PropTypes.string.isRequired,
                    issuer: PropTypes.string.isRequired,
                    mode: PropTypes.string.isRequired
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
        notifiers: PropTypes.arrayOf(PropTypes.object).isRequired,
        imageIntegrations: PropTypes.arrayOf(PropTypes.object).isRequired,
        fetchAuthProviders: PropTypes.func.isRequired,
        fetchAPITokens: PropTypes.func.isRequired,
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
        const imageIntegrations = this.renderIntegrationTiles('imageIntegrations');
        const orchestrators = this.renderIntegrationTiles('orchestrators');
        const plugins = this.renderIntegrationTiles('plugins');
        const authProviders = this.renderIntegrationTiles('authProviders');
        return (
            <div className="h-full flex flex-col md:w-full bg-base-200">
                <div className="flex flex-no-shrink">
                    <PageHeader header="Integrations" subHeader="Setup & Configuration" />
                </div>
                <div className="w-full h-full overflow-auto">
                    <section className="mb-6">
                        <h2 className="bg-base-200 border-b border-primary-400 font-700 mx-4 pin-t px-3 py-4 sticky text-base text-base-600 tracking-wide  uppercase z-1">
                            Images
                        </h2>
                        <div className="flex flex-col items-center w-full">
                            <div className="flex flex-wrap w-full -mx-6 p-3">
                                {imageIntegrations}
                            </div>
                        </div>
                    </section>

                    <section className="mb-6">
                        <h2 className="bg-base-200 border-b border-primary-400 font-700 mx-4 pin-t px-3 py-4 sticky text-base text-base-600 tracking-wide  uppercase z-1">
                            Orchestrators &amp; Container Platforms
                        </h2>
                        <div className="flex flex-col items-center w-full">
                            <div className="flex flex-wrap w-full -mx-6 p-3">{orchestrators}</div>
                        </div>
                    </section>

                    <section className="mb-6">
                        <h2 className="bg-base-200 border-b border-primary-400 font-700 mx-4 pin-t px-3 py-4 sticky text-base text-base-600 tracking-wide  uppercase z-1">
                            Plugins
                        </h2>
                        <div className="flex flex-col items-center w-full">
                            <div className="flex flex-wrap w-full -mx-6 p-3">{plugins}</div>
                        </div>
                    </section>

                    <section className="mb-6">
                        <h2 className="bg-base-200 border-b border-primary-400 font-700 mx-4 pin-t px-3 py-4 sticky text-base text-base-600 tracking-wide  uppercase z-1">
                            Authentication Providers
                        </h2>
                        <div className="flex flex-col items-center w-full">
                            <div className="flex flex-wrap w-full -mx-6 p-3">{authProviders}</div>
                        </div>
                    </section>
                </div>
                {this.renderIntegrationModal()}
            </div>
        );
    }
}

const mapStateToProps = createStructuredSelector({
    authProviders: selectors.getAuthProviders,
    apiTokens: selectors.getAPITokens,
    clusters: selectors.getClusters,
    notifiers: selectors.getNotifiers,
    imageIntegrations: selectors.getImageIntegrations
});

const mapDispatchToProps = dispatch => ({
    fetchAuthProviders: () => dispatch(authActions.fetchAuthProviders.request()),
    fetchAPITokens: () => dispatch(apiTokenActions.fetchAPITokens.request()),
    fetchNotifiers: () => dispatch(integrationActions.fetchNotifiers.request()),
    fetchImageIntegrations: () => dispatch(integrationActions.fetchImageIntegrations.request()),
    fetchRegistries: () => dispatch(integrationActions.fetchRegistries.request()),
    fetchScanners: () => dispatch(integrationActions.fetchScanners.request()),
    fetchClusters: () => dispatch(clusterActions.fetchClusters.request())
});

export default connect(
    mapStateToProps,
    mapDispatchToProps
)(IntegrationsPage);
