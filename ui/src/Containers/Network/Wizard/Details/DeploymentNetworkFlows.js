import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { createStructuredSelector } from 'reselect';
import { connect } from 'react-redux';
import { selectors } from 'reducers';
import Panel from 'Components/Panel';
import TablePagination from 'Components/TablePagination';
import NoResultsMessage from 'Components/NoResultsMessage';
import Table, { rtTrActionsClassName } from 'Components/Table';
import { actions as graphActions } from 'reducers/network/graph';
import Tooltip from 'rc-tooltip';
import * as Icon from 'react-feather';

class DeploymentNetworkFlows extends Component {
    static propTypes = {
        deploymentEdges: PropTypes.arrayOf(PropTypes.shape({})),
        networkGraphRef: PropTypes.shape({
            setSelectedNode: PropTypes.func,
            getNodeData: PropTypes.func,
            onNodeClick: PropTypes.func
        })
    };

    static defaultProps = {
        deploymentEdges: [],
        networkGraphRef: null
    };

    constructor(props) {
        super(props);
        this.state = {
            page: 0,
            selectedNode: null
        };
    }

    getNodeData = data => {
        const { getNodeData } = this.props.networkGraphRef;
        const node = getNodeData(data.targetId || data.target);
        return node && node[0] && node[0].data;
    };

    highlightNode = ({ data }) => {
        const { networkGraphRef } = this.props;
        const node = this.getNodeData(data);
        if (node) {
            networkGraphRef.setSelectedNode(node);
            this.setState({ selectedNode: node });
        }
    };

    navigate = ({ data }) => () => {
        const { onNodeClick } = this.props.networkGraphRef;
        const node = this.getNodeData(data);
        if (node) {
            onNodeClick(node);
        }
    };

    setTablePage = newPage => {
        this.setState({ page: newPage });
    };

    renderRowActionButtons = node => {
        const enableIconColor = 'text-primary-600';
        const enableIconHoverColor = 'text-primary-700';
        return (
            <div className="border-2 border-r-2 border-base-400 bg-base-100 flex">
                <Tooltip
                    placement="left"
                    mouseLeaveDelay={0}
                    overlay={<div>Navigate to Deployment</div>}
                    overlayClassName="pointer-events-none"
                >
                    <button
                        type="button"
                        className={`p-1 px-4 hover:bg-primary-200 ${enableIconColor} hover:${enableIconHoverColor}`}
                        onClick={this.navigate(node)}
                    >
                        <Icon.ArrowUpRight className="mt-1 h-4 w-4" />
                    </button>
                </Tooltip>
            </div>
        );
    };

    renderTable() {
        const columns = [
            {
                Header: 'Deployment',
                accessor: 'data.targetName',
                Cell: ({ value }) => <span>{value}</span>
            },
            {
                Header: 'Namespace',
                accessor: 'data.targetNS',
                Cell: ({ value }) => <span>{value}</span>
            },
            {
                accessor: '',
                headerClassName: 'hidden',
                className: rtTrActionsClassName,
                Cell: ({ original }) => this.renderRowActionButtons(original)
            }
        ];

        const { deploymentEdges } = this.props;
        if (!deploymentEdges.length)
            return <NoResultsMessage message="No results found. Please refine your search." />;
        return (
            <Table
                rows={deploymentEdges}
                columns={columns}
                onRowClick={this.highlightNode}
                noDataText="No results found. Please refine your search."
                page={this.state.page}
                idAttribute="data.target"
                selectedRowId={this.state.selectedNode && this.state.selectedNode.id}
            />
        );
    }

    renderOverview() {
        const { deploymentEdges } = this.props;
        if (!deploymentEdges.length)
            return <NoResultsMessage message="No deployment network flows" />;
        const paginationComponent = (
            <TablePagination
                page={this.state.page}
                dataLength={deploymentEdges.length}
                setPage={this.setTablePage}
            />
        );
        const subHeaderText = `${deploymentEdges.length} Network Flow${
            deploymentEdges.length === 1 ? '' : 's'
        }`;
        const content = <div>{this.renderTable()}</div>;

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

    render() {
        return <div className="w-full h-full">{this.renderOverview()}</div>;
    }
}

const mapStateToProps = createStructuredSelector({
    networkGraphRef: selectors.getNetworkGraphRef
});

const mapDispatchToProps = {
    setSelectedNamespace: graphActions.setSelectedNamespace
};

export default connect(
    mapStateToProps,
    mapDispatchToProps
)(DeploymentNetworkFlows);
