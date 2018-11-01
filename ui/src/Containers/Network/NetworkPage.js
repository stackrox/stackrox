import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createSelector, createStructuredSelector } from 'reselect';
import { selectors } from 'reducers';
import { actions as networkActions } from 'reducers/network';
import { actions as deploymentActions, types as deploymentTypes } from 'reducers/deployments';
import { ALL_STATE, ACTIVE_STATE } from 'constants/networkGraph';

import PageHeader from 'Components/PageHeader';
import SearchInput from 'Components/SearchInput';
import NetworkGraph from 'Components/NetworkGraph';
import * as Icon from 'react-feather';
import Panel from 'Components/Panel';
import Tabs from 'Components/Tabs';
import Loader from 'Components/Loader';
import TabContent from 'Components/TabContent';
import DeploymentDetails from '../Risk/DeploymentDetails';
import NetworkPoliciesDetails from './NetworkPoliciesDetails';
import NetworkGraphLegend from './NetworkGraphLegend';
import NetworkPolicySimulator from './NetworkPolicySimulator';
import GraphFilters from './GraphFilters';
import ZoomButtons from './ZoomButtons';
import NodesUpdateSection from './NodesUpdateSection';
import ClusterSelect from './ClusterSelect';

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
        fetchNetworkPolicies: PropTypes.func.isRequired,
        fetchDeployment: PropTypes.func.isRequired,
        deployment: PropTypes.shape({}),
        networkPolicies: PropTypes.arrayOf(PropTypes.object),
        networkPolicyGraph: PropTypes.shape({
            nodes: PropTypes.arrayOf(
                PropTypes.shape({
                    deploymentId: PropTypes.string.isRequired,
                    outEdges: PropTypes.shape({})
                })
            ),
            epoch: PropTypes.number
        }).isRequired,
        networkFlowGraph: PropTypes.shape({
            nodes: PropTypes.arrayOf(
                PropTypes.shape({
                    deploymentId: PropTypes.string.isRequired,
                    outEdges: PropTypes.shape({})
                })
            ),
            epoch: PropTypes.number
        }),
        networkFlowMapping: PropTypes.shape({}).isRequired,
        isFetchingNode: PropTypes.bool,
        nodeUpdatesEpoch: PropTypes.number,
        networkGraphUpdateKey: PropTypes.number.isRequired,
        incrementNetworkGraphUpdateKey: PropTypes.func.isRequired,
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
        onNodesUpdate: PropTypes.func,
        lastUpdatedTimestamp: PropTypes.instanceOf(Date)
    };

    static defaultProps = {
        isFetchingNode: false,
        selectedNodeId: null,
        networkPolicies: [],
        deployment: {},
        nodeUpdatesEpoch: null,
        errorMessage: '',
        yamlFile: null,
        onNodesUpdate: null,
        lastUpdatedTimestamp: null,
        networkFlowGraph: null
    };

    state = {
        filterState: ALL_STATE
    };

    onSearch = searchOptions => {
        if (searchOptions.length && !searchOptions[searchOptions.length - 1].type) {
            this.closeSidePanel();
        }
    };

    onNodeClick = node => {
        this.props.setSelectedNodeId(node.deploymentId);
        this.props.fetchDeployment(node.deploymentId);
        this.props.fetchNetworkPolicies([...node.policyIds]);
    };

    onUpdateGraph = () => {
        if (this.props.onNodesUpdate) this.props.onNodesUpdate();
        this.props.incrementNetworkGraphUpdateKey();
    };

    onYamlUpload = yamlFile => {
        this.props.setYamlFile(yamlFile);
        this.props.incrementNetworkGraphUpdateKey();
    };

    onFilterButtonClick = filterState => {
        this.setState({ filterState });
    };

    getNodeUpdates = () => {
        const { networkPolicyGraph, nodeUpdatesEpoch } = this.props;
        return nodeUpdatesEpoch - networkPolicyGraph.epoch;
    };

    closeSidePanel = () => {
        this.props.setSelectedNodeId(null);
    };

    toggleNetworkPolicySimulator = () => {
        const {
            simulatorMode,
            setNetworkGraphState,
            setSimulatorMode,
            setYamlFile,
            yamlFile,
            incrementNetworkGraphUpdateKey
        } = this.props;
        setSimulatorMode(!simulatorMode);
        setYamlFile(yamlFile);
        setNetworkGraphState();
        incrementNetworkGraphUpdateKey();
    };

    renderGraph = () => {
        const colorType = this.props.networkGraphState === 'ERROR' ? 'alert' : 'success';
        const simulatorMode = this.props.simulatorMode ? 'simulator-mode' : '';
        const networkGraphState = this.props.networkGraphState === 'ERROR' ? 'error' : 'success';
        const networkGraph = {};
        const { filterState } = this.state;
        const { networkPolicyGraph, networkFlowGraph } = this.props;
        networkGraph.nodes =
            filterState === ACTIVE_STATE ? networkFlowGraph.nodes : networkPolicyGraph.nodes;
        return (
            <div className={`${simulatorMode} ${networkGraphState} w-full h-full`}>
                {this.props.simulatorMode && (
                    <div
                        className={`absolute pin-t pin-l bg-${colorType}-600 text-base-100 font-600 uppercase p-2 z-1`}
                    >
                        Simulation Mode
                    </div>
                )}
                <NetworkGraph
                    ref={instance => {
                        this.networkGraph = instance;
                    }}
                    updateKey={this.props.networkGraphUpdateKey}
                    nodes={networkGraph.nodes}
                    networkFlowMapping={this.props.networkFlowMapping}
                    onNodeClick={this.onNodeClick}
                    filterState={filterState}
                />
            </div>
        );
    };

    renderSideComponents = () => {
        const { selectedNodeId, simulatorMode } = this.props;
        const className = `${
            selectedNodeId || simulatorMode ? 'w-1/3' : 'w-0'
        } h-full absolute pin-r z-1 bg-primary-200`;
        return (
            <div className={className}>
                {this.renderSidePanel()}
                <NodesUpdateSection
                    getNodeUpdates={this.getNodeUpdates}
                    onUpdateGraph={this.onUpdateGraph}
                    lastUpdatedTimestamp={this.props.lastUpdatedTimestamp}
                />
                {this.renderNetworkPolicySimulator()}
                <ZoomButtons networkGraph={this.networkGraph} />
            </div>
        );
    };

    renderSidePanel = () => {
        const { selectedNodeId, deployment, networkPolicies } = this.props;
        if (!selectedNodeId || this.props.simulatorMode) {
            return (
                <NodesUpdateSection
                    getNodeUpdates={this.getNodeUpdates}
                    onUpdateGraph={this.onUpdateGraph}
                    lastUpdatedTimestamp={this.props.lastUpdatedTimestamp}
                />
            );
        }
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
            <Panel header={deployment.name} onClose={this.closeSidePanel}>
                {content}
            </Panel>
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
                    id="network"
                    className="w-full"
                    searchOptions={this.props.searchOptions}
                    searchModifiers={this.props.searchModifiers}
                    searchSuggestions={this.props.searchSuggestions}
                    setSearchOptions={this.props.setSearchOptions}
                    setSearchModifiers={this.props.setSearchModifiers}
                    setSearchSuggestions={this.props.setSearchSuggestions}
                    onSearch={this.onSearch}
                />
                <ClusterSelect closeSidePanel={this.closeSidePanel} />
                {this.renderNetworkPolicySimulatorButton()}
            </PageHeader>
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
                data-test-id={`simulator-button-${this.props.simulatorMode ? 'on' : 'off'}`}
                className={`flex-no-shrink border-2 rounded-sm text-sm ml-2 pl-2 pr-2 h-9 ${className}`}
                onClick={this.toggleNetworkPolicySimulator}
            >
                <span className="pr-1">Simulate Network Policy</span>
                <Icon.Circle className="h-2 w-2" fill={iconColor} stroke={iconColor} />
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
                    <section className="network-grid-bg flex flex-1 relative">
                        <GraphFilters onFilter={this.onFilterButtonClick} />
                        <NetworkGraphLegend />
                        {this.renderGraph()}
                        {this.renderSideComponents()}
                    </section>
                </div>
            </section>
        );
    }
}

const isViewFiltered = createSelector(
    [selectors.getNetworkSearchOptions],
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
    selectedClusterId: selectors.getSelectedNetworkClusterId,
    networkPolicyGraph: selectors.getNetworkPolicyGraph,
    networkFlowGraph: selectors.getNetworkFlowGraph,
    searchOptions: selectors.getNetworkSearchOptions,
    searchModifiers: selectors.getNetworkSearchModifiers,
    searchSuggestions: selectors.getNetworkSearchSuggestions,
    nodeUpdatesEpoch: selectors.getNodeUpdatesEpoch,
    isViewFiltered,
    selectedNodeId: selectors.getSelectedNodeId,
    deployment: selectors.getDeployment,
    networkFlowMapping: selectors.getNetworkFlowMapping,
    networkPolicies: selectors.getNetworkPolicies,
    networkGraphUpdateKey: selectors.getNetworkGraphUpdateKey,
    isFetchingNode: state => selectors.getLoadingStatus(state, deploymentTypes.FETCH_DEPLOYMENT),
    lastUpdatedTimestamp: selectors.getLastUpdatedTimestamp,
    networkGraphState: getNetworkGraphState,
    simulatorMode: selectors.getSimulatorMode,
    errorMessage: selectors.getNetworkGraphErrorMessage,
    yamlFile: selectors.getYamlFile
});

const mapDispatchToProps = {
    fetchNetworkPolicies: networkActions.fetchNetworkPolicies.request,
    fetchDeployment: deploymentActions.fetchDeployment.request,
    setSelectedNodeId: networkActions.setSelectedNodeId,
    setSearchOptions: networkActions.setNetworkSearchOptions,
    setSearchModifiers: networkActions.setNetworkSearchModifiers,
    setSearchSuggestions: networkActions.setNetworkSearchSuggestions,
    setSimulatorMode: networkActions.setSimulatorMode,
    setNetworkGraphState: networkActions.setNetworkGraphState,
    setYamlFile: networkActions.setYamlFile,
    incrementNetworkGraphUpdateKey: networkActions.incrementNetworkGraphUpdateKey,
    onNodesUpdate: networkActions.networkNodesUpdate
};

export default connect(
    mapStateToProps,
    mapDispatchToProps
)(NetworkPage);
