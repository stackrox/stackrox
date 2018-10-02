import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createSelector, createStructuredSelector } from 'reselect';
import { selectors } from 'reducers';
import { actions as environmentActions, networkGraphClusters } from 'reducers/environment';
import { actions as deploymentActions, types as deploymentTypes } from 'reducers/deployments';
import { actions as clusterActions } from 'reducers/clusters';

import Select from 'Components/ReactSelect';
import PageHeader from 'Components/PageHeader';
import SearchInput from 'Components/SearchInput';
import NetworkGraphZoom from 'Components/NetworkGraphZoom';
import EnvironmentGraph from 'Components/EnvironmentGraph/svg/EnvironmentGraph';
import * as Icon from 'react-feather';
import Panel from 'Components/Panel';
import Tabs from 'Components/Tabs';
import Loader from 'Components/Loader';
import TabContent from 'Components/TabContent';
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
        networkGraphState: PropTypes.string.isRequired,
        setSimulatorMode: PropTypes.func.isRequired,
        simulatorMode: PropTypes.bool.isRequired,
        setNetworkGraphState: PropTypes.func.isRequired,
        setYamlFile: PropTypes.func.isRequired,
        errorMessage: PropTypes.string,
        yamlFile: PropTypes.shape({
            content: PropTypes.string,
            name: PropTypes.string
        }),
        onNodesUpdate: PropTypes.func
    };

    static defaultProps = {
        isFetchingNode: false,
        selectedNodeId: null,
        networkPolicies: [],
        deployment: {},
        nodeUpdatesEpoch: null,
        selectedClusterId: '',
        errorMessage: '',
        yamlFile: null,
        onNodesUpdate: null
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
        if (this.props.onNodesUpdate) this.props.onNodesUpdate();
        this.props.incrementEnvironmentGraphUpdateKey();
    };

    onYamlUpload = yamlFile => {
        this.props.setYamlFile(yamlFile);
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

    toggleNetworkPolicySimulator = () => {
        const {
            simulatorMode,
            setNetworkGraphState,
            setSimulatorMode,
            setYamlFile,
            yamlFile,
            incrementEnvironmentGraphUpdateKey
        } = this.props;
        setSimulatorMode(!simulatorMode);
        setYamlFile(yamlFile);
        setNetworkGraphState();
        incrementEnvironmentGraphUpdateKey();
    };

    renderGraph = () => {
        const colorType = this.props.networkGraphState === 'ERROR' ? 'danger' : 'success';
        const simulatorMode = this.props.simulatorMode ? 'simulator-mode' : '';
        const networkGraphState = this.props.networkGraphState === 'ERROR' ? 'error' : 'success';
        return (
            <div className={`${simulatorMode} ${networkGraphState} w-full h-full`}>
                {this.props.simulatorMode && (
                    <div
                        className={`absolute pin-t pin-l bg-${colorType}-600 text-base-100 uppercase p-2 z-1`}
                    >
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
        if (!selectedNodeId || this.props.simulatorMode) return null;
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
            <div className="w-1/3 h-full absolute pin-r z-1 bg-primary-200">
                <Panel header={deployment.name} onClose={this.closeSidePanel}>
                    {content}
                </Panel>
            </div>
        );
    };

    renderPageHeader = () => {
        const subHeader = this.props.isViewFiltered ? 'Filtered view' : 'Default view';
        return (
            <PageHeader
                header="Network Graph"
                subHeader={subHeader}
                className="w-2/3 bg-primary-200 "
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
                {this.renderNetworkPolicySimulatorButton()}
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
        const positionStyle = this.props.simulatorMode ? { right: '40%' } : { right: '0' };
        return (
            <div className="absolute pin-b" style={positionStyle}>
                <NetworkGraphZoom />
            </div>
        );
    };

    renderNetworkPolicySimulatorButton = () => {
        const className = this.props.simulatorMode
            ? 'bg-success-200 border-success-500 hover:border-success-600 hover:text-success-600 text-success-500'
            : 'bg-base-200 hover:border-base-300 hover:text-base-600 border-base-200 text-base-500';
        const iconColor = this.props.simulatorMode ? '#53c6a9' : '#d2d5ed';
        return (
            <button
                type="button"
                className={`flex-no-shrink border-2 rounded-sm text-sm ml-2 pl-2 pr-2 h-9 ${className}`}
                onClick={this.toggleNetworkPolicySimulator}
            >
                <span className="pr-1">Simulate Network Policy</span>
                <Icon.Circle className="h-2 w-2" fill={iconColor} stroke={iconColor} />
            </button>
        );
    };

    renderNodesUpdateButton = () => {
        const nodeUpdatesCount = this.getNodeUpdates();
        if (Number.isNaN(nodeUpdatesCount) || nodeUpdatesCount <= 0) return null;
        return (
            <button
                type="button"
                className="btn-graph-refresh absolute pin-t pin-r mt-2 mr-2 p-2 bg-primary-500 hover:bg-primary-400 rounded-sm text-sm text-base-100"
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
        if (!this.props.simulatorMode) return null;
        return (
            <NetworkPolicySimulator
                onClose={this.toggleNetworkPolicySimulator}
                onYamlUpload={this.onYamlUpload}
                yamlUploadState={this.props.networkGraphState}
                errorMessage={this.props.errorMessage}
                yamlFile={this.props.yamlFile}
            />
        );
    }

    render() {
        return (
            <section className="flex flex-1 h-full w-full">
                <div className="flex flex-1 flex-col w-full">
                    <div className="flex">{this.renderPageHeader()}</div>
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

const getNetworkGraphState = createSelector(
    [selectors.getYamlFile, selectors.getNetworkGraphState],
    (yamlFile, networkGraphState) => {
        if (!yamlFile) {
            return 'INITIAL';
        }
        return networkGraphState;
    }
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
    networkGraphState: getNetworkGraphState,
    simulatorMode: selectors.getSimulatorMode,
    errorMessage: selectors.getNetworkGraphErrorMessage,
    yamlFile: selectors.getYamlFile
});

const mapDispatchToProps = {
    fetchNetworkPolicies: environmentActions.fetchNetworkPolicies.request,
    fetchDeployment: deploymentActions.fetchDeployment.request,
    fetchClusters: clusterActions.fetchClusters.request,
    setSelectedNodeId: environmentActions.setSelectedNodeId,
    setSearchOptions: environmentActions.setEnvironmentSearchOptions,
    setSearchModifiers: environmentActions.setEnvironmentSearchModifiers,
    setSearchSuggestions: environmentActions.setEnvironmentSearchSuggestions,
    selectClusterId: environmentActions.selectEnvironmentClusterId,
    setSimulatorMode: environmentActions.setSimulatorMode,
    setNetworkGraphState: environmentActions.setNetworkGraphState,
    setYamlFile: environmentActions.setYamlFile,
    incrementEnvironmentGraphUpdateKey: environmentActions.incrementEnvironmentGraphUpdateKey,
    onNodesUpdate: environmentActions.networkNodesUpdate
};

export default connect(
    mapStateToProps,
    mapDispatchToProps
)(EnvironmentPage);
