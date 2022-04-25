import React, { Component } from 'react';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import PropTypes from 'prop-types';
import isEmpty from 'lodash/isEmpty';
import cloneDeep from 'lodash/cloneDeep';

import { selectors } from 'reducers';
import { actions as graphActions } from 'reducers/network/graph';
import { actions as pageActions } from 'reducers/network/page';
import { actions as sidepanelActions } from 'reducers/network/sidepanel';
import NetworkGraph from 'Components/NetworkGraph';
import NoResultsMessage from 'Components/NoResultsMessage';
import { filterModes } from 'constants/networkFilterModes';
import { nodeTypes } from 'constants/networkGraph';
import entityTypes from 'constants/entityTypes';
import sidepanelStages from '../SidePanel/sidepanelStages';
import Filters from './Overlays/Filters';
import Legend from './Overlays/Legend';

class Graph extends Component {
    static propTypes = {
        sidePanelOpen: PropTypes.bool.isRequired,
        openSidePanel: PropTypes.func.isRequired,
        closeSidePanel: PropTypes.func.isRequired,
        setSidePanelStage: PropTypes.func.isRequired,

        networkNodeMap: PropTypes.shape({}).isRequired,
        networkEdgeMap: PropTypes.shape({}),

        networkFlowGraphUpdateKey: PropTypes.number.isRequired,

        setSelectedNode: PropTypes.func.isRequired,
        setSelectedNamespace: PropTypes.func.isRequired,
        selectedNamespace: PropTypes.shape({
            id: PropTypes.string,
            deployments: PropTypes.arrayOf(PropTypes.shape({})),
        }),
        clusters: PropTypes.arrayOf(PropTypes.object).isRequired,
        selectedClusterId: PropTypes.string,
        showNamespaceFlows: PropTypes.string.isRequired,
        setShowNamespaceFlows: PropTypes.func.isRequired,

        filterState: PropTypes.number.isRequired,
        isLoading: PropTypes.bool.isRequired,
        featureFlags: PropTypes.arrayOf(PropTypes.shape),
        setNetworkGraphRef: PropTypes.func.isRequired,
        setSelectedNodeInGraph: PropTypes.func,
        lastUpdatedTimestamp: PropTypes.instanceOf(Date),

        isSimulationOn: PropTypes.bool,
        // @TODO: merge this with networkNodeMap and networkEdgeMap somehow
        simulatedBaselines: PropTypes.arrayOf(PropTypes.shape),
    };

    static defaultProps = {
        networkEdgeMap: null,
        selectedClusterId: '',
        featureFlags: [],
        setSelectedNodeInGraph: null,
        lastUpdatedTimestamp: null,
        selectedNamespace: null,
        isSimulationOn: false,
        simulatedBaselines: [],
    };

    shouldComponentUpdate(nextProps) {
        const {
            networkFlowGraphUpdateKey,
            filterState,
            isLoading,
            sidePanelOpen,
            networkEdgeMap,
            networkNodeMap,
            isSimulationOn,
            showNamespaceFlows,
            simulatedBaselines,
        } = this.props;
        return (
            !networkEdgeMap ||
            isEmpty(networkNodeMap) ||
            nextProps.networkFlowGraphUpdateKey !== networkFlowGraphUpdateKey ||
            nextProps.filterState !== filterState ||
            nextProps.isLoading !== isLoading ||
            nextProps.sidePanelOpen !== sidePanelOpen ||
            nextProps.isSimulationOn !== isSimulationOn ||
            nextProps.showNamespaceFlows !== showNamespaceFlows ||
            nextProps.simulatedBaselines !== simulatedBaselines
        );
    }

    onNamespaceClick = (namespace) => {
        this.props.setSelectedNamespace(namespace);
        this.props.setSidePanelStage(sidepanelStages.namespaceDetails);
        this.props.openSidePanel();
    };

    onExternalEntitiesClick = () => {
        this.props.setSidePanelStage(sidepanelStages.externalDetails);
        this.props.openSidePanel();
    };

    onNodeClick = (node) => {
        if (node?.type === entityTypes.CLUSTER) {
            return;
        }
        this.props.setSelectedNode(node);
        this.props.setSidePanelStage(sidepanelStages.details);
        this.props.openSidePanel();
    };

    renderGraph = () => {
        const {
            networkNodeMap,
            networkFlowGraphUpdateKey,
            networkEdgeMap,
            closeSidePanel,
            filterState,
            clusters,
            selectedClusterId,
            showNamespaceFlows,
            featureFlags,
            setNetworkGraphRef,
            setSelectedNamespace,
            setSelectedNodeInGraph,
            lastUpdatedTimestamp,
            selectedNamespace,
            isSimulationOn,
            simulatedBaselines,
        } = this.props;

        // If we have more than 2000 nodes, display a message instead of the graph.
        const nodeLimit = 2000;
        if (Object.keys(networkNodeMap).length > nodeLimit) {
            // hopefully a temporal solution
            return (
                <NoResultsMessage message="There are too many deployments to render on the graph. Please refine your search to a set of namespaces or deployments to display." />
            );
        }

        const selectedClusterName =
            clusters.find((cluster) => cluster.id === selectedClusterId)?.name || 'Unknown cluster';

        const augmentedNetworkNodeMap =
            filterState === filterModes.all ? augmentCidrs(networkNodeMap) : networkNodeMap;

        return (
            <NetworkGraph
                updateKey={networkFlowGraphUpdateKey}
                networkEdgeMap={networkEdgeMap}
                networkNodeMap={augmentedNetworkNodeMap}
                onNodeClick={this.onNodeClick}
                onNamespaceClick={this.onNamespaceClick}
                onExternalEntitiesClick={this.onExternalEntitiesClick}
                onClickOutside={closeSidePanel}
                filterState={filterState}
                selectedClusterName={selectedClusterName}
                showNamespaceFlows={showNamespaceFlows}
                featureFlags={featureFlags}
                setNetworkGraphRef={setNetworkGraphRef}
                setSelectedNamespace={setSelectedNamespace}
                setSelectedNodeInGraph={setSelectedNodeInGraph}
                lastUpdatedTimestamp={lastUpdatedTimestamp}
                selectedNamespace={selectedNamespace}
                selectedClusterId={selectedClusterId}
                isReadOnly={isSimulationOn}
                simulatedBaselines={simulatedBaselines}
            />
        );
    };

    render() {
        const { isSimulationOn, showNamespaceFlows, setShowNamespaceFlows } = this.props;

        return (
            <div className="w-full h-full">
                {this.renderGraph()}
                {!isSimulationOn && (
                    <Filters
                        showNamespaceFlows={showNamespaceFlows}
                        setShowNamespaceFlows={setShowNamespaceFlows}
                    />
                )}
                <Legend />
            </div>
        );
    }
}

const mapStateToProps = createStructuredSelector({
    sidePanelOpen: selectors.getSidePanelOpen,
    filterState: selectors.getNetworkGraphFilterMode,
    networkNodeMap: selectors.getNetworkNodeMap,
    networkEdgeMap: selectors.getNetworkEdgeMap,
    networkFlowGraphUpdateKey: selectors.getNetworkFlowGraphUpdateKey,
    clusters: selectors.getClusters,
    selectedClusterId: selectors.getSelectedNetworkClusterId,
    isLoading: selectors.getNetworkGraphLoading,
    featureFlags: selectors.getFeatureFlags,
    sidePanelStage: selectors.getSidePanelStage,
    networkPolicyModification: selectors.getNetworkPolicyModification,
    lastUpdatedTimestamp: selectors.getLastUpdatedTimestamp,
    selectedNamespace: selectors.getSelectedNamespace,
});

const mapDispatchToProps = {
    setSelectedNode: graphActions.setSelectedNode,
    setSelectedNamespace: graphActions.setSelectedNamespace,
    openSidePanel: pageActions.openSidePanel,
    setSidePanelStage: sidepanelActions.setSidePanelStage,
    setNetworkGraphRef: graphActions.setNetworkGraphRef,
    setNetworkGraphLoading: graphActions.setNetworkGraphLoading,
    closeSidePanel: pageActions.closeSidePanel,
    setSelectedNodeInGraph: graphActions.setSelectedNode,
};

export default connect(mapStateToProps, mapDispatchToProps)(Graph);

function augmentCidrs(originalNodeMap) {
    const clonedMap = cloneDeep(originalNodeMap);

    Object.keys(clonedMap).forEach((id) => {
        if (
            clonedMap[id]?.active?.entity?.type === nodeTypes.CIDR_BLOCK &&
            !clonedMap[id]?.allowed
        ) {
            clonedMap[id].allowed = cloneDeep(originalNodeMap[id].active);
        }
    });
    return clonedMap;
}
