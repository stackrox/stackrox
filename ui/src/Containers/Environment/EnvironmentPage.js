import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createSelector, createStructuredSelector } from 'reselect';
import { selectors } from 'reducers';
import { actions as environmentActions, networkGraphClusters } from 'reducers/environment';
import { actions as deploymentActions, types as deploymentTypes } from 'reducers/deployments';
import { actions as clusterActions } from 'reducers/clusters';

import Select from 'react-select';
import PageHeader from 'Components/PageHeader';
import SearchInput from 'Components/SearchInput';
import EnvironmentGraph from 'Components/EnvironmentGraph';
import NetworkGraphZoom from 'Components/NetworkGraphZoom';
import * as Icon from 'react-feather';
import Panel from 'Components/Panel';
import Tabs from 'Components/Tabs';
import Loader from 'Components/Loader';
import TabContent from 'Components/TabContent';
import featureFlag from 'utils/featureFlag';
import DeploymentDetails from '../Risk/DeploymentDetails';
import NetworkPoliciesDetails from './NetworkPoliciesDetails';
import EnvironmentGraphLegend from './EnvironmentGraphLegend';
import NetworkPolicySimulator from './NetworkPolicySimulator';

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
        incrementEnvironmentGraphUpdateKey: PropTypes.func.isRequired
    };

    static defaultProps = {
        isFetchingNode: false,
        selectedNodeId: null,
        networkPolicies: [],
        deployment: {},
        nodeUpdatesEpoch: null,
        selectedClusterId: ''
    };

    state = {
        showNetworkPolicySimulator: false
    };

    componentDidMount() {
        this.props.fetchClusters();
    }

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
        this.props.incrementEnvironmentGraphUpdateKey();
        this.props.fetchEnvironmentGraph();
    };

    onYamlUpload = () => {}; // placeholder function for updating graph on yaml upload

    getNodeUpdates = () => {
        const { environmentGraph, nodeUpdatesEpoch } = this.props;
        return nodeUpdatesEpoch - environmentGraph.epoch;
    };

    closeSidePanel = () => {
        this.props.setSelectedNodeId(null);
    };

    changeCluster = option => {
        if (option) this.props.selectClusterId(option.value);
        this.closeSidePanel();
    };

    showNetworkPolicySimulator = () => this.setState({ showNetworkPolicySimulator: true });

    hideNetworkPolicySimulator = () => this.setState({ showNetworkPolicySimulator: false });

    renderGraph = () => {
        const className = this.state.showNetworkPolicySimulator
            ? 'border-4 border-success-500'
            : '';
        return (
            <div className={` w-full h-full border-box ${className}`}>
                {this.state.showNetworkPolicySimulator && (
                    <div className="absolute pin-t pin-l bg-success-500 text-white uppercase p-2 z-10">
                        Simulation Mode
                    </div>
                )}
                <EnvironmentGraph
                    updateKey={this.props.environmentGraphUpdateKey}
                    nodes={this.props.environmentGraph.nodes}
                    edges={this.props.environmentGraph.edges}
                    onNodeClick={this.onNodeClick}
                />
            </div>
        );
    };

    renderSidePanel = () => {
        const { selectedNodeId, deployment, networkPolicies } = this.props;
        if (!selectedNodeId || this.state.showNetworkPolicySimulator) return null;
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
            <div className="w-2/5 h-full absolute pin-t pin-r z-20">
                <Panel header={deployment.name} onClose={this.closeSidePanel}>
                    {content}
                </Panel>
            </div>
        );
    };

    renderPageHeader = () => {
        const subHeader = this.props.isViewFiltered ? 'Filtered view' : 'Default view';
        return (
            <PageHeader header="Network Graph" subHeader={subHeader} className="w-2/3">
                <SearchInput
                    id="environment"
                    className="flex flex-1"
                    searchOptions={this.props.searchOptions}
                    searchModifiers={this.props.searchModifiers}
                    searchSuggestions={this.props.searchSuggestions}
                    setSearchOptions={this.props.setSearchOptions}
                    setSearchModifiers={this.props.setSearchModifiers}
                    setSearchSuggestions={this.props.setSearchSuggestions}
                    onSearch={this.onSearch}
                />
                {this.renderClustersSelect()}
                {featureFlag.networkPolicySimulator && this.renderNetworkPolicySimulatorButton()}
            </PageHeader>
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

    renderNetworkGraphZoom = () => {
        const positionStyle = this.state.showNetworkPolicySimulator
            ? { right: '40%' }
            : { right: '0' };
        return (
            <div className="absolute pin-b z-10" style={positionStyle}>
                <NetworkGraphZoom />
            </div>
        );
    };

    renderNetworkPolicySimulatorButton = () => {
        const className = this.state.showNetworkPolicySimulator
            ? 'bg-success-200 border-success-500 hover:border-success-600 hover:text-success-600 text-success-500'
            : 'bg-base-100 hover:border-base-300 hover:text-base-500 border-base-200 text-base-400';
        const style = { height: '36px' };
        const iconColor = this.state.showNetworkPolicySimulator ? '#53c6a9' : '#d2d5ed';
        return (
            <button
                className={`border-2 rounded-sm text-sm ml-2 pl-2 pr-2 ${className}`}
                style={style}
                onClick={this.showNetworkPolicySimulator}
            >
                <span className="pr-1">Simulate Network Policy</span>
                <Icon.Circle className="h-2 w-2" fill={iconColor} stroke={iconColor} />
            </button>
        );
    };

    renderNodesUpdateButton = () => {
        const nodeUpdatesCount = this.getNodeUpdates();
        if (nodeUpdatesCount !== 'NaN' || nodeUpdatesCount <= 0) return null;
        return (
            <button
                className="btn-graph-refresh absolute pin-t pin-r mt-2 mr-2 p-2 bg-primary-500 hover:bg-primary-400 rounded-sm text-sm text-white"
                onClick={this.onUpdateGraph}
            >
                <Icon.Circle className="h-2 w-2 border-primary-300" />
                <span className="pl-1">{`${nodeUpdatesCount} ${
                    nodeUpdatesCount === 1 ? 'update' : 'updates'
                } available`}</span>
            </button>
        );
    };

    renderNetworkPolicySimulator() {
        if (!this.state.showNetworkPolicySimulator) return null;
        return (
            <NetworkPolicySimulator
                onClose={this.hideNetworkPolicySimulator}
                onYamlUpload={this.onYamlUpload}
            />
        );
    }

    render() {
        return (
            <section className="flex flex-1 h-full w-full">
                <div className="flex flex-1 flex-col w-full">
                    <div className="flex flex-row">{this.renderPageHeader()}</div>
                    <section className="environment-grid-bg flex flex-1 relative">
                        <EnvironmentGraphLegend />
                        {this.renderGraph()}
                        {this.renderNodesUpdateButton()}
                        {this.renderSidePanel()}
                        {this.renderNetworkPolicySimulator()}
                        {this.renderNetworkGraphZoom()}
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
    setSearchSuggestions: environmentActions.setEnvironmentSearchSuggestions,
    selectClusterId: environmentActions.selectEnvironmentClusterId,
    incrementEnvironmentGraphUpdateKey: environmentActions.incrementEnvironmentGraphUpdateKey
};

export default connect(mapStateToProps, mapDispatchToProps)(EnvironmentPage);
