import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createSelector, createStructuredSelector } from 'reselect';
import Select from 'react-select';

import { selectors } from 'reducers';
import { actions as environmentActions } from 'reducers/environment';
import { actions as deploymentActions, types as deploymentTypes } from 'reducers/deployments';
import { actions as clusterActions } from 'reducers/clusters';

import PageHeader from 'Components/PageHeader';
import SearchInput from 'Components/SearchInput';
import EnvironmentGraph from 'Components/EnvironmentGraph';
import * as Icon from 'react-feather';
import Panel from 'Components/Panel';
import Tabs from 'Components/Tabs';
import Loader from 'Components/Loader';
import TabContent from 'Components/TabContent';
import DeploymentDetails from '../Risk/DeploymentDetails';
import NetworkPoliciesDetails from './NetworkPoliciesDetails';

class EnvironmentPage extends Component {
    static propTypes = {
        selectedNodeId: PropTypes.string,
        searchOptions: PropTypes.arrayOf(PropTypes.object).isRequired,
        searchModifiers: PropTypes.arrayOf(PropTypes.object).isRequired,
        searchSuggestions: PropTypes.arrayOf(PropTypes.object).isRequired,
        setSearchOptions: PropTypes.func.isRequired,
        setSearchModifiers: PropTypes.func.isRequired,
        setSearchSuggestions: PropTypes.func.isRequired,
        setSelectedNodeId: PropTypes.func.isRequired,
        isViewFiltered: PropTypes.bool.isRequired,
        fetchEnvironmentGraph: PropTypes.func.isRequired,
        fetchNetworkPolicies: PropTypes.func.isRequired,
        fetchDeployment: PropTypes.func.isRequired,
        fetchClusters: PropTypes.func.isRequired,
        deployment: PropTypes.shape({}),
        networkPolicies: PropTypes.arrayOf(PropTypes.object),
        environmentGraph: PropTypes.shape({
            nodes: PropTypes.arrayOf(
                PropTypes.shape({
                    id: PropTypes.string.isRequired
                })
            ),
            edges: PropTypes.arrayOf(
                PropTypes.shape({
                    source: PropTypes.string.isRequired,
                    target: PropTypes.string.isRequired
                })
            ),
            epoch: PropTypes.number
        }).isRequired,
        clusters: PropTypes.arrayOf(PropTypes.object).isRequired,
        isFetchingNode: PropTypes.bool,
        nodeUpdatesEpoch: PropTypes.number
    };

    static defaultProps = {
        isFetchingNode: false,
        selectedNodeId: null,
        networkPolicies: [],
        deployment: {},
        nodeUpdatesEpoch: null
    };

    state = {
        selectedCluster: null
    };

    componentDidMount() {
        this.props.fetchClusters();
    }

    onSearch = searchOptions => {
        if (searchOptions.length && !searchOptions[searchOptions.length - 1].type) {
            this.closeSidePanel();
        }
    };

    onClusterSelect = selectedCluster => {
        this.setState({ selectedCluster });
    };

    onNodeClick = node => {
        this.props.setSelectedNodeId(node.id);
        this.props.fetchDeployment(node.id);
        this.props.fetchNetworkPolicies([...node.policyIds]);
    };

    getNodeUpdates = () => {
        const { environmentGraph, nodeUpdatesEpoch } = this.props;
        return nodeUpdatesEpoch - environmentGraph.epoch;
    };

    closeSidePanel = () => {
        this.props.setSelectedNodeId(null);
    };

    renderGraph = () => (
        <EnvironmentGraph
            nodes={this.props.environmentGraph.nodes}
            edges={this.props.environmentGraph.edges}
            onNodeClick={this.onNodeClick}
        />
    );

    renderSidePanel = () => {
        const { selectedNodeId, deployment, networkPolicies } = this.props;
        if (!selectedNodeId) return null;
        const envGraphPanelTabs = [{ text: 'Deployment Details' }, { text: 'Network Policies' }];
        const content = this.props.isFetchingNode ? (
            <Loader />
        ) : (
            <Tabs headers={envGraphPanelTabs}>
                <TabContent>
                    <div className="flex flex-1 flex-col">
                        {deployment.id && <DeploymentDetails deployment={deployment} />}
                    </div>
                </TabContent>
                <TabContent>
                    <div className="flex flex-1 flex-col">
                        <NetworkPoliciesDetails networkPolicies={networkPolicies} />
                    </div>
                </TabContent>
            </Tabs>
        );

        return (
            <div className="w-2/5 h-full absolute pin-t pin-r">
                <Panel header={deployment.name} onClose={this.closeSidePanel}>
                    {content}
                </Panel>
            </div>
        );
    };

    renderClustersSelect = () => {
        if (!this.props.clusters.length) return null;
        const { clusters } = this.props;
        const defaultValue = {
            label: clusters[0].name,
            value: clusters[0].id
        };
        const clustersProps = {
            name: 'select-cluster',
            className: 'capitalize',
            options: clusters.map(cluster => ({
                label: cluster.name,
                value: cluster.id
            })),
            placeholder: 'Select Cluster',
            onChange: this.onClusterSelect,
            value: this.state.selectedCluster ? this.state.selectedCluster : defaultValue,
            ignoreCase: false
        };
        return (
            <div className="w-1/3 py-3 pr-4 border-b border-primary-300">
                <Select {...clustersProps} autoFocus />
            </div>
        );
    };

    render() {
        const subHeader = this.props.isViewFiltered ? 'Filtered view' : 'Default view';
        const nodeUpdatesCount = this.getNodeUpdates();
        return (
            <section className="flex flex-1 h-full w-full">
                <div className="flex flex-1 flex-col w-full">
                    <div className="flex flex-row">
                        <PageHeader header="Environment" subHeader={subHeader} className="w-2/3">
                            <SearchInput
                                id="environment"
                                searchOptions={this.props.searchOptions}
                                searchModifiers={this.props.searchModifiers}
                                searchSuggestions={this.props.searchSuggestions}
                                setSearchOptions={this.props.setSearchOptions}
                                setSearchModifiers={this.props.setSearchModifiers}
                                setSearchSuggestions={this.props.setSearchSuggestions}
                                onSearch={this.onSearch}
                            />
                        </PageHeader>
                        {this.renderClustersSelect()}
                    </div>
                    <section className="environment-grid-bg flex flex-1 relative">
                        {this.renderGraph()}
                        {nodeUpdatesCount > 0 && (
                            <button
                                className="btn-graph-refresh absolute pin-t pin-r mt-2 mr-2 p-2 bg-primary-300 hover:bg-primary-200 rounded-sm text-sm text-white"
                                onClick={this.props.fetchEnvironmentGraph}
                            >
                                <Icon.Circle className="h-2 w-2 border-primary-300" />
                                <span className="pl-1">{`${nodeUpdatesCount} Node ${
                                    nodeUpdatesCount === 1 ? 'update' : 'updates'
                                } available`}</span>
                            </button>
                        )}
                        {this.renderSidePanel()}
                    </section>
                </div>
            </section>
        );
    }
}

const isViewFiltered = createSelector(
    [selectors.getEnvironmentSearchOptions],
    searchOptions => searchOptions.length !== 0
);

const mapStateToProps = createStructuredSelector({
    environmentGraph: selectors.getEnvironmentGraph,
    clusters: selectors.getClusters,
    searchOptions: selectors.getEnvironmentSearchOptions,
    searchModifiers: selectors.getEnvironmentSearchModifiers,
    searchSuggestions: selectors.getEnvironmentSearchSuggestions,
    nodeUpdatesEpoch: selectors.getNodeUpdatesEpoch,
    isViewFiltered,
    selectedNodeId: selectors.getSelectedNodeId,
    deployment: selectors.getDeployment,
    networkPolicies: selectors.getNetworkPolicies,
    isFetchingNode: state => selectors.getLoadingStatus(state, deploymentTypes.FETCH_DEPLOYMENT)
});

const mapDispatchToProps = {
    fetchEnvironmentGraph: environmentActions.fetchEnvironmentGraph.request,
    fetchNetworkPolicies: environmentActions.fetchNetworkPolicies.request,
    fetchDeployment: deploymentActions.fetchDeployment.request,
    fetchClusters: clusterActions.fetchClusters.request,
    setSelectedNodeId: environmentActions.setSelectedNodeId,
    setSearchOptions: environmentActions.setEnvironmentSearchOptions,
    setSearchModifiers: environmentActions.setEnvironmentSearchModifiers,
    setSearchSuggestions: environmentActions.setEnvironmentSearchSuggestions
};

export default connect(mapStateToProps, mapDispatchToProps)(EnvironmentPage);
