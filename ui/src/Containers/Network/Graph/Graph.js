import React, { Component } from 'react';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import PropTypes from 'prop-types';
import { selectors } from 'reducers';
import { actions as backendActions } from 'reducers/network/backend';
import { actions as graphActions } from 'reducers/network/graph';
import { actions as pageActions } from 'reducers/network/page';
import { actions as wizardActions } from 'reducers/network/wizard';
import { actions as deploymentActions } from 'reducers/deployments';

import NoResultsMessage from 'Components/NoResultsMessage';
import wizardStages from '../Wizard/wizardStages';
import filterModes from './filterModes';
import Filters from './Overlays/Filters';
import Legend from './Overlays/Legend';
import NetworkGraph from '../../../Components/NetworkGraph';

class Graph extends Component {
    static propTypes = {
        setGraphRef: PropTypes.func.isRequired,

        wizardOpen: PropTypes.bool.isRequired,
        wizardStage: PropTypes.string.isRequired,
        filterState: PropTypes.number.isRequired,

        networkPolicyGraph: PropTypes.shape({
            nodes: PropTypes.arrayOf(PropTypes.shape({}))
        }).isRequired,
        networkPolicyGraphState: PropTypes.string.isRequired,

        networkFlowGraph: PropTypes.shape({
            nodes: PropTypes.arrayOf(PropTypes.shape({}))
        }),
        networkFlowMapping: PropTypes.shape({}).isRequired,
        networkFlowGraphUpdateKey: PropTypes.number.isRequired,
        networkFlowGraphState: PropTypes.string.isRequired,

        setSelectedNodeId: PropTypes.func.isRequired,
        fetchDeployment: PropTypes.func.isRequired,
        fetchNetworkPolicies: PropTypes.func.isRequired,

        setWizardStage: PropTypes.func.isRequired,
        openWizard: PropTypes.func.isRequired
    };

    static defaultProps = {
        networkFlowGraph: null
    };

    onNodeClick = node => {
        this.props.setSelectedNodeId(node.deploymentId);
        this.props.fetchDeployment(node.deploymentId);
        this.props.fetchNetworkPolicies([...node.policyIds]);
        this.props.setWizardStage(wizardStages.details);
        this.props.openWizard();
    };

    renderGraph = (nodes, filterState) => {
        // If we have more than 200 nodes, display a message instead of the graph.
        if (nodes.length > 600) {
            // hopefully a temporal solution
            return (
                <NoResultsMessage message="There are too many deployments to render on the graph. Please refine your search to a set of namespaces or deployments to display." />
            );
        }
        return (
            <NetworkGraph
                ref={this.props.setGraphRef}
                updateKey={this.props.networkFlowGraphUpdateKey}
                nodes={nodes}
                networkFlowMapping={this.props.networkFlowMapping}
                onNodeClick={this.onNodeClick}
                filterState={filterState}
            />
        );
    };

    render() {
        // Simulator styling.
        const simulatorOn =
            this.props.wizardOpen && this.props.wizardStage === wizardStages.simulator;
        const simulatorMode = simulatorOn ? 'simulator-mode' : '';

        // Graph nodes and styling.
        const { filterState } = this.props;
        let nodes;
        let networkGraphState;
        if (filterState === filterModes.active) {
            const { networkFlowGraphState, networkFlowGraph } = this.props;
            ({ nodes } = networkFlowGraph);
            networkGraphState = networkFlowGraphState;
        } else {
            const { networkPolicyGraphState, networkPolicyGraph } = this.props;
            ({ nodes } = networkPolicyGraph);
            networkGraphState = networkPolicyGraphState;
        }
        const networkGraphStateClass = networkGraphState === 'ERROR' ? 'error' : 'success';

        // Rendering.
        const width = this.props.wizardOpen ? 'w-2/3' : 'w-full';
        return (
            <div className={`${simulatorMode} ${networkGraphStateClass} ${width} h-full`}>
                {this.renderGraph(nodes, filterState)}
                <Filters />
                <Legend />
            </div>
        );
    }
}

const mapStateToProps = createStructuredSelector({
    wizardOpen: selectors.getNetworkWizardOpen,
    wizardStage: selectors.getNetworkWizardStage,
    filterState: selectors.getNetworkGraphFilterMode,

    networkPolicyGraph: selectors.getNetworkPolicyGraph,
    networkPolicyGraphState: selectors.getNetworkPolicyGraphState,

    networkFlowGraph: selectors.getNetworkFlowGraph,
    networkFlowMapping: selectors.getNetworkFlowMapping,
    networkFlowGraphUpdateKey: selectors.getNetworkFlowGraphUpdateKey,
    networkFlowGraphState: selectors.getNetworkFlowGraphState
});

const mapDispatchToProps = {
    setSelectedNodeId: graphActions.setSelectedNodeId,
    fetchDeployment: deploymentActions.fetchDeployment.request,
    fetchNetworkPolicies: backendActions.fetchNetworkPolicies.request,

    openWizard: pageActions.openNetworkWizard,
    setWizardStage: wizardActions.setNetworkWizardStage
};

export default connect(
    mapStateToProps,
    mapDispatchToProps
)(Graph);
