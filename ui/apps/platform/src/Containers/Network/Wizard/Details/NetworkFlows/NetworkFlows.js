import React, { useState } from 'react';
import PropTypes from 'prop-types';
import pluralize from 'pluralize';

import { createStructuredSelector } from 'reselect';
import { connect } from 'react-redux';
import { selectors } from 'reducers';
import { actions as graphActions } from 'reducers/network/graph';

import Panel from 'Components/Panel';
import TablePagination from 'Components/TablePagination';
import NoResultsMessage from 'Components/NoResultsMessage';
import { filterModes, filterLabels } from 'constants/networkFilterModes';
import NetworkFlowsTable from './NetworkFlowsTable';

/**
 * Grabs the deployment-to-deployment edges and filters based on the filter state
 *
 * @param {!Object[]} edges
 * @param {!Number} filterState
 * @returns {!Object[]}
 */
export function getNetworkFlows(edges, filterState) {
    let results;
    const nodeMapping = edges.reduce(
        (
            acc,
            {
                data: {
                    destNodeId,
                    traffic,
                    destNodeName,
                    destNodeNamespace,
                    isActive,
                    portsAndProtocols,
                },
            }
        ) => {
            if (acc[destNodeId]) {
                return acc;
            }
            return {
                ...acc,
                [destNodeId]: {
                    traffic,
                    deploymentId: destNodeId,
                    deploymentName: destNodeName,
                    namespace: destNodeNamespace,
                    connection: isActive ? 'active' : 'allowed',
                    portsAndProtocols,
                },
            };
        },
        {}
    );
    switch (filterState) {
        case filterModes.active:
            results = Object.values(nodeMapping).filter((edge) => edge.connection === 'active');
            break;
        case filterModes.allowed:
            results = Object.values(nodeMapping).filter((edge) => edge.connection === 'allowed');
            break;
        default:
            results = Object.values(nodeMapping);
    }
    return results;
}

const NetworkFlows = ({ deploymentEdges, networkGraphRef, filterState, onDeploymentClick }) => {
    const [selectedNode, setSelectedNode] = useState(null);
    const [page, setPage] = useState(0);

    const filterStateString = filterState !== filterModes.all ? filterLabels[filterState] : '';

    if (!deploymentEdges.length) {
        return <NoResultsMessage message={`No ${filterStateString} deployment flows`} />;
    }

    const networkFlows = getNetworkFlows(deploymentEdges, filterState);

    const paginationComponent = (
        <TablePagination page={page} dataLength={networkFlows.length} setPage={setPage} />
    );
    const subHeaderText = `${networkFlows.length} ${filterStateString} ${pluralize(
        'Flow',
        networkFlows.length
    )}`;

    function getNodeDataById(nodeId) {
        const { getNodeData: getNodeDataFromRef } = networkGraphRef;
        const node = getNodeDataFromRef(nodeId);
        return node?.[0]?.data;
    }

    function onHighlightNode({ deploymentId }) {
        const node = getNodeDataById(deploymentId);
        if (node) {
            if (onDeploymentClick) {
                onDeploymentClick(node.deploymentId);
            }
            networkGraphRef.setSelectedNode(node);
            setSelectedNode(node);
        }
    }

    function onNavigateToNodeById(nodeId) {
        return function onNavigate() {
            const { onNodeClick } = networkGraphRef;
            const node = getNodeDataById(nodeId);
            if (node) {
                onNodeClick(node);
            }
        };
    }

    return (
        <div className="w-full h-full">
            <Panel
                header={subHeaderText}
                headerComponents={paginationComponent}
                isUpperCase={false}
            >
                <div className="w-full h-full bg-base-100">
                    <NetworkFlowsTable
                        networkFlows={networkFlows}
                        page={page}
                        selectedNode={selectedNode}
                        filterState={filterState}
                        onHighlightNode={onHighlightNode}
                        onNavigateToNodeById={onNavigateToNodeById}
                    />
                </div>
            </Panel>
        </div>
    );
};

NetworkFlows.propTypes = {
    deploymentEdges: PropTypes.arrayOf(PropTypes.shape({})),
    networkGraphRef: PropTypes.shape({
        setSelectedNode: PropTypes.func,
        getNodeData: PropTypes.func,
        onNodeClick: PropTypes.func,
    }),
    filterState: PropTypes.number.isRequired,
    onDeploymentClick: PropTypes.func,
};

NetworkFlows.defaultProps = {
    deploymentEdges: [],
    networkGraphRef: null,
    onDeploymentClick: null,
};

const mapStateToProps = createStructuredSelector({
    networkGraphRef: selectors.getNetworkGraphRef,
    filterState: selectors.getNetworkGraphFilterMode,
});

const mapDispatchToProps = {
    setSelectedNamespace: graphActions.setSelectedNamespace,
};

export default connect(mapStateToProps, mapDispatchToProps)(NetworkFlows);
