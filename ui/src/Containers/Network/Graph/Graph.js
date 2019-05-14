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

import NetworkGraph from 'Components/NetworkGraph';
import NoResultsMessage from 'Components/NoResultsMessage';
import wizardStages from '../Wizard/wizardStages';
import { filterModes } from './filterModes';
import Filters from './Overlays/Filters';
import Legend from './Overlays/Legend';

class Graph extends Component {
    static propTypes = {
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
        setSelectedNode: PropTypes.func.isRequired,
        setSelectedNamespace: PropTypes.func.isRequired,
        fetchDeployment: PropTypes.func.isRequired,
        fetchNetworkPolicies: PropTypes.func.isRequired,

        setWizardStage: PropTypes.func.isRequired,
        openWizard: PropTypes.func.isRequired,
        closeWizard: PropTypes.func.isRequired,

        isLoading: PropTypes.bool.isRequired
    };

    static defaultProps = {
        networkFlowGraph: null
    };

    shouldComponentUpdate(nextProps) {
        const { networkFlowGraphUpdateKey, filterState, isLoading, wizardOpen } = this.props;
        return (
            nextProps.networkFlowGraphUpdateKey !== networkFlowGraphUpdateKey ||
            nextProps.filterState !== filterState ||
            nextProps.isLoading !== isLoading ||
            nextProps.wizardOpen !== wizardOpen
        );
    }

    onNamespaceClick = namespace => {
        this.props.setSelectedNamespace(namespace);
        this.props.setWizardStage(wizardStages.namespaceDetails);
        this.props.openWizard();
    };

    onNodeClick = node => {
        this.props.setSelectedNode(node);
        this.props.fetchDeployment(node.deploymentId);
        this.props.fetchNetworkPolicies([...node.policyIds]);
        this.props.setWizardStage(wizardStages.details);
        this.props.openWizard();
    };

    renderGraph = nodes => {
        // If we have more than 1100 nodes, display a message instead of the graph.
        if (nodes.length > 1100) {
            // hopefully a temporal solution
            return (
                <NoResultsMessage message="There are too many deployments to render on the graph. Please refine your search to a set of namespaces or deployments to display." />
            );
        }
        const { networkFlowGraphUpdateKey, networkFlowMapping } = this.props;
        const { closeWizard, filterState } = this.props;
        return (
            <NetworkGraph
                updateKey={networkFlowGraphUpdateKey}
                nodes={nodes}
                networkFlowMapping={networkFlowMapping}
                onNodeClick={this.onNodeClick}
                onNamespaceClick={this.onNamespaceClick}
                onClickOutside={closeWizard}
                filterState={filterState}
            />
        );
    };

    render() {
        const { wizardOpen, wizardStage, filterState } = this.props;
        // Simulator styling.
        const simulatorOn =
            wizardOpen &&
            (wizardStage === wizardStages.simulator || wizardStage === wizardStages.creator);
        const simulatorMode = simulatorOn ? 'simulator-mode' : '';

        // Graph nodes and styling.
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
        return (
            <div className={`${simulatorMode} ${networkGraphStateClass} w-full h-full theme-light`}>
                {this.renderGraph(nodes)}
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
    networkFlowGraphState: selectors.getNetworkFlowGraphState,

    isLoading: selectors.getNetworkGraphLoading
});

const mapDispatchToProps = {
    setSelectedNode: graphActions.setSelectedNode,
    setSelectedNamespace: graphActions.setSelectedNamespace,
    fetchDeployment: deploymentActions.fetchDeployment.request,
    fetchNetworkPolicies: backendActions.fetchNetworkPolicies.request,

    openWizard: pageActions.openNetworkWizard,
    setWizardStage: wizardActions.setNetworkWizardStage,
    setNetworkGraphRef: graphActions.setNetworkGraphRef,
    setNetworkGraphLoading: graphActions.setNetworkGraphLoading,
    closeWizard: pageActions.closeNetworkWizard
};

export default connect(
    mapStateToProps,
    mapDispatchToProps
)(Graph);
