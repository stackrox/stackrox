import React, { useState } from 'react';
import PropTypes from 'prop-types';
import { createStructuredSelector } from 'reselect';
import { connect } from 'react-redux';
import { selectors } from 'reducers';
import { actions as graphActions } from 'reducers/network/graph';

import Panel from 'Components/Panel';
import TablePagination from 'Components/TablePagination';
import NoResultsMessage from 'Components/NoResultsMessage';
import Table, { rtTrActionsClassName } from 'Components/Table';
import RowActionButton from 'Components/RowActionButton';
import { filterModes, filterLabels } from 'Containers/Network/Graph/filterModes';
import * as Icon from 'react-feather';

const DeploymentNetworkFlows = ({
    deploymentEdges,
    networkGraphRef,
    filterState,
    onDeploymentClick,
}) => {
    const [selectedNode, setSelectedNode] = useState(null);
    const [page, setPage] = useState(0);

    function getNodeData(data) {
        const { getNodeData: getNodeDataFromRef } = networkGraphRef;
        const node = getNodeDataFromRef(data.destNodeId);
        return node && node[0] && node[0].data;
    }

    function highlightNode({ data }) {
        const node = getNodeData(data);
        if (node) {
            if (onDeploymentClick) onDeploymentClick(node.deploymentId);
            networkGraphRef.setSelectedNode(node);
            setSelectedNode(node);
        }
    }

    const navigate = ({ data }) => () => {
        const { onNodeClick } = networkGraphRef;
        const node = getNodeData(data);
        if (node) {
            onNodeClick(node);
        }
    };

    function renderRowActionButtons(node) {
        return (
            <div className="border-2 border-r-2 border-base-400 bg-base-100 flex">
                <RowActionButton
                    text="Navigate to Deployment"
                    onClick={navigate(node)}
                    icon={<Icon.ArrowUpRight className="my-1 h-4 w-4" />}
                />
            </div>
        );
    }

    function renderTable() {
        const filterStateString = filterState !== filterModes.all ? filterLabels[filterState] : '';
        const columns = [
            {
                Header: 'Deployment',
                accessor: 'data.destNodeName',
                // eslint-disable-next-line react/prop-types
                Cell: ({ value }) => <span>{value}</span>,
            },
            {
                Header: 'Namespace',
                accessor: 'data.destNodeNS',
                // eslint-disable-next-line react/prop-types
                Cell: ({ value }) => <span>{value}</span>,
            },
            {
                Header: 'Flow',
                accessor: 'data.isActive',
                show: filterState === filterModes.all,
                // eslint-disable-next-line react/prop-types
                Cell: ({ value }) => <span>{value ? 'active' : 'allowed'}</span>,
            },
            {
                accessor: '',
                headerClassName: 'hidden',
                className: rtTrActionsClassName,
                Cell: ({ original }) => renderRowActionButtons(original),
            },
        ];
        if (!deploymentEdges.length)
            return <NoResultsMessage message={`No ${filterStateString} deployment flows`} />;
        return (
            <Table
                rows={deploymentEdges}
                columns={columns}
                onRowClick={highlightNode}
                noDataText={`No ${filterStateString} deployment flows`}
                page={page}
                idAttribute="data.destNodeId"
                selectedRowId={selectedNode && selectedNode.id}
            />
        );
    }

    function renderOverview() {
        const filterStateString = filterState !== filterModes.all ? filterLabels[filterState] : '';
        if (!deploymentEdges.length)
            return <NoResultsMessage message={`No ${filterStateString} deployment flows`} />;
        const paginationComponent = (
            <TablePagination page={page} dataLength={deploymentEdges.length} setPage={setPage} />
        );
        const subHeaderText = `${deploymentEdges.length} ${filterStateString} Flow${
            deploymentEdges.length === 1 ? '' : 's'
        }`;
        const content = <div>{renderTable()}</div>;

        return (
            <Panel
                header={subHeaderText}
                headerComponents={paginationComponent}
                isUpperCase={false}
            >
                <div className="w-full h-full bg-base-100">{content}</div>
            </Panel>
        );
    }

    return <div className="w-full h-full">{renderOverview()}</div>;
};

DeploymentNetworkFlows.propTypes = {
    deploymentEdges: PropTypes.arrayOf(PropTypes.shape({})),
    networkGraphRef: PropTypes.shape({
        setSelectedNode: PropTypes.func,
        getNodeData: PropTypes.func,
        onNodeClick: PropTypes.func,
    }),
    filterState: PropTypes.number.isRequired,
    onDeploymentClick: PropTypes.func,
};

DeploymentNetworkFlows.defaultProps = {
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

export default connect(mapStateToProps, mapDispatchToProps)(DeploymentNetworkFlows);
