import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createSelector, createStructuredSelector } from 'reselect';
import { selectors } from 'reducers';
import { actions as environmentActions, networkGraphClusters } from 'reducers/environment';
import { actions as deploymentActions, types as deploymentTypes } from 'reducers/deployments';
import { actions as clusterActions } from 'reducers/clusters';

import dateFns from 'date-fns';
import Select from 'Components/ReactSelect';
import PageHeader from 'Components/PageHeader';
import SearchInput from 'Components/SearchInput';
import NetworkGraph from 'Components/EnvironmentGraph/webgl/NetworkGraph';
import * as Icon from 'react-feather';
import Panel from 'Components/Panel';
import Tabs from 'Components/Tabs';
import Loader from 'Components/Loader';
import TabContent from 'Components/TabContent';
import DeploymentDetails from 'Containers/Risk/DeploymentDetails';
import NetworkPoliciesDetails from 'Containers/Network/NetworkPoliciesDetails';
import NetworkGraphLegend from 'Containers/Network/NetworkGraphLegend';

class NetworkPage extends Component {
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
        selectClusterId: PropTypes.func.isRequired,
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
        selectedClusterId: PropTypes.string,
        isFetchingNode: PropTypes.bool,
        nodeUpdatesEpoch: PropTypes.number,
        environmentGraphUpdateKey: PropTypes.number.isRequired,
        incrementEnvironmentGraphUpdateKey: PropTypes.func.isRequired,
        onNodesUpdate: PropTypes.func
    };

    static defaultProps = {
        isFetchingNode: false,
        selectedNodeId: null,
        networkPolicies: [],
        deployment: {},
        nodeUpdatesEpoch: null,
        selectedClusterId: '',
        onNodesUpdate: null
    };

    onSearch = searchOptions => {
        if (searchOptions.length && !searchOptions[searchOptions.length - 1].type) {
            this.closeSidePanel();
        }
    };

    onNodeClick = node => {
        this.props.setSelectedNodeId(node.id);
        this.props.fetchDeployment(node.id);
        this.props.fetchNetworkPolicies([...node.policyIds]);
    };

    onUpdateGraph = () => {
        if (this.props.onNodesUpdate) this.props.onNodesUpdate();
        this.props.incrementEnvironmentGraphUpdateKey();
    };

    getNodeUpdates = () => {
        const { environmentGraph, nodeUpdatesEpoch } = this.props;
        return nodeUpdatesEpoch - environmentGraph.epoch;
    };

    closeSidePanel = () => {
        this.props.setSelectedNodeId(null);
    };

    changeCluster = clusterId => {
        this.props.selectClusterId(clusterId);
        this.closeSidePanel();
    };

    renderGraph = () => (
        <NetworkGraph
            updateKey={this.props.environmentGraphUpdateKey}
            nodes={this.props.environmentGraph.nodes}
            links={this.props.environmentGraph.edges}
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
                    <div className="flex flex-1 flex-col h-full">
                        {deployment.id && <DeploymentDetails deployment={deployment} />}
                    </div>
                </TabContent>
                <TabContent>
                    <div className="flex flex-1 flex-col h-full">
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
        // network policies are only applicable on k8s-based clusters
        const options = this.props.clusters
            .filter(cluster => networkGraphClusters[cluster.type])
            .map(cluster => ({
                value: cluster.id,
                label: cluster.name
            }));
        const clustersProps = {
            className: 'min-w-64 ml-5',
            options,
            value: this.props.selectedClusterId,
            placeholder: 'Select a cluster',
            onChange: this.changeCluster,
            autoFocus: true
        };
        return <Select {...clustersProps} />;
    };

    renderNodesUpdateButton = () => {
        const nodeUpdatesCount = this.getNodeUpdates();
        if (Number.isNaN(nodeUpdatesCount) || nodeUpdatesCount <= 0) return null;
        return (
            <button
                type="button"
                className="btn-graph-refresh p-2 bg-primary-500 hover:bg-primary-400 rounded-sm text-sm text-base-100 mt-2 w-full"
                onClick={this.onUpdateGraph}
            >
                <Icon.Circle className="h-2 w-2 border-primary-300" />
                <span className="pl-1">
                    {`${nodeUpdatesCount} update${nodeUpdatesCount === 1 ? '' : 's'} available`}
                </span>
            </button>
        );
    };

    renderNodesUpdateSection = () => {
        if (!this.props.lastUpdatedTimestamp) return null;
        return (
            <div className="absolute pin-t pin-r mt-2 mr-2 p-2 bg-base-100 z-10 rounded-sm shadow-outline">
                <div className="uppercase">{`Last Updated: ${dateFns.format(
                    this.props.lastUpdatedTimestamp,
                    'hh:mm:ssA'
                )}`}</div>
                {this.renderNodesUpdateButton()}
            </div>
        );
    };

    render() {
        const subHeader = this.props.isViewFiltered ? 'Filtered view' : 'Default view';
        return (
            <section className="flex flex-1 h-full w-full">
                <div className="flex flex-1 flex-col w-full">
                    <div className="flex">
                        <PageHeader
                            header="Environment"
                            subHeader={subHeader}
                            className="w-1/2 bg-primary-200"
                        >
                            <SearchInput
                                id="environment"
                                className="w-full"
                                searchOptions={this.props.searchOptions}
                                searchModifiers={this.props.searchModifiers}
                                searchSuggestions={this.props.searchSuggestions}
                                setSearchOptions={this.props.setSearchOptions}
                                setSearchModifiers={this.props.setSearchModifiers}
                                setSearchSuggestions={this.props.setSearchSuggestions}
                                onSearch={this.onSearch}
                            />
                            {this.renderClustersSelect()}
                        </PageHeader>
                    </div>
                    <section className="environment-grid-bg flex flex-1 relative">
                        <NetworkGraphLegend />
                        {this.renderGraph()}
                        {this.renderNodesUpdateSection()}
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
    clusters: selectors.getClusters,
    selectedClusterId: selectors.getSelectedEnvironmentClusterId,
    environmentGraph: selectors.getEnvironmentGraph,
    searchOptions: selectors.getEnvironmentSearchOptions,
    searchModifiers: selectors.getEnvironmentSearchModifiers,
    searchSuggestions: selectors.getEnvironmentSearchSuggestions,
    nodeUpdatesEpoch: selectors.getNodeUpdatesEpoch,
    isViewFiltered,
    selectedNodeId: selectors.getSelectedNodeId,
    deployment: selectors.getDeployment,
    networkPolicies: selectors.getNetworkPolicies,
    environmentGraphUpdateKey: selectors.getEnvironmentGraphUpdateKey,
    isFetchingNode: state => selectors.getLoadingStatus(state, deploymentTypes.FETCH_DEPLOYMENT),
    lastUpdatedTimestamp: selectors.getLastUpdatedTimestamp
});

const mapDispatchToProps = {
    fetchEnvironmentGraph: environmentActions.fetchEnvironmentGraph.request,
    fetchNetworkPolicies: environmentActions.fetchNetworkPolicies.request,
    fetchDeployment: deploymentActions.fetchDeployment.request,
    fetchClusters: clusterActions.fetchClusters.request,
    setSelectedNodeId: environmentActions.setSelectedNodeId,
    setSearchOptions: environmentActions.setEnvironmentSearchOptions,
    setSearchModifiers: environmentActions.setEnvironmentSearchModifiers,
    setSearchSuggestions: environmentActions.setEnvironmentSearchSuggestions,
    selectClusterId: environmentActions.selectEnvironmentClusterId,
    incrementEnvironmentGraphUpdateKey: environmentActions.incrementEnvironmentGraphUpdateKey,
    onNodesUpdate: environmentActions.networkNodesUpdate
};

export default connect(
    mapStateToProps,
    mapDispatchToProps
)(NetworkPage);
