import React, { Component } from 'react';
import PropTypes from 'prop-types';
import * as Icon from 'react-feather';
import { connect } from 'react-redux';
import { createSelector, createStructuredSelector } from 'reselect';
import { actions as clusterActions } from 'reducers/clusters';
import { selectors } from 'reducers';

import Modal from 'Components/Modal';
import Table from 'Components/Table';
import Panel from 'Components/Panel';
import ClusterCreationPanel from 'Containers/Integrations/ClusterCreationPanel';

import tableColumnDescriptor from 'Containers/Integrations/tableColumnDescriptor';
import { clusterTypeLabels } from 'messages/common';
import { deleteCluster } from 'services/ClustersService';

class ClustersModal extends Component {
    static propTypes = {
        clusters: PropTypes.arrayOf(PropTypes.object).isRequired,
        onRequestClose: PropTypes.func.isRequired,
        editCluster: PropTypes.func.isRequired,
        editingCluster: PropTypes.shape(),
        selectedClusterType: PropTypes.string.isRequired
    };

    static defaultProps = {
        editingCluster: null
    };

    constructor(props) {
        super(props);

        this.state = {
            disableDeleteButton: true
        };
    }

    onRowChecked = selectedRows => {
        if (selectedRows.length && this.state.disableDeleteButton === true)
            this.setState({ disableDeleteButton: false });
        else if (!selectedRows.length && this.state.disableDeleteButton === false)
            this.setState({ disableDeleteButton: true });
    };

    /* 
     *  @TODO: Need to use a redux table so we don't need to use refs
     */
    deleteCluster = () => {
        const promises = [];
        this.clusterTable.getSelectedRows().forEach(obj => {
            // close the view panel if that policy is being deleted
            if (
                this.props.editingCluster &&
                this.props.editingCluster.id &&
                obj.id === this.props.editingCluster.id
            ) {
                this.props.editCluster(null);
            }
            const promise = deleteCluster(obj.id);
            promises.push(promise);
        });
        Promise.all(promises).then(() => {
            this.clusterTable.clearSelectedRows();
        });
    };

    addCluster = () => {
        this.props.editCluster(undefined);
    };

    renderTable = () => {
        const header = `${clusterTypeLabels[this.props.selectedClusterType]} Integrations`;
        const buttons = [
            {
                renderIcon: () => <Icon.Trash2 className="h-4 w-4" />,
                text: 'Delete',
                className:
                    'flex py-2 px-2 rounded-sm font-400 uppercase text-center text-sm items-center ml-2 w-24 justify-center text-danger-500 hover:text-white bg-white hover:bg-danger-400 border border-danger-400',
                onClick: this.deleteCluster,
                disabled: this.state.disableDeleteButton
            },
            {
                renderIcon: () => <Icon.Plus className="h-4 w-4" />,
                text: 'Add',
                className:
                    'flex py-2 px-2 rounded-sm font-400 uppercase text-center text-sm items-center ml-2 w-24 justify-center text-success-500 hover:text-white bg-white hover:bg-success-400 border border-success-400',
                onClick: this.addCluster,
                disabled: this.props.editingCluster !== null
            }
        ];
        const columns = tableColumnDescriptor.clusters;
        const rows = this.props.clusters;
        const onRowClickHandler = () => cluster => this.props.editCluster(cluster.id);
        return (
            <div className="flex flex-1">
                <Panel header={header} buttons={buttons}>
                    <Table
                        columns={columns}
                        rows={rows}
                        checkboxes
                        onRowClick={onRowClickHandler()}
                        onRowChecked={this.onRowChecked}
                        ref={table => {
                            this.clusterTable = table;
                        }}
                    />
                </Panel>
            </div>
        );
    };

    renderClusterCreationPanel = () => {
        if (!this.props.editingCluster) return null;
        return (
            <div className="flex w-1/2">
                <ClusterCreationPanel />
            </div>
        );
    };

    render() {
        const { selectedClusterType, onRequestClose } = this.props;
        return (
            <Modal isOpen onRequestClose={onRequestClose} className="w-5/6 h-full">
                <header className="flex items-center w-full p-4 bg-primary-500 text-white uppercase">
                    <span className="flex flex-1">{clusterTypeLabels[selectedClusterType]}</span>
                    <Icon.X className="h-4 w-4 cursor-pointer" onClick={onRequestClose} />
                </header>
                <div className="flex flex-1 w-full bg-white">
                    {this.renderTable()}
                    {this.renderClusterCreationPanel()}
                </div>
            </Modal>
        );
    }
}

const getEditingCluster = createSelector(
    [selectors.getClusters, selectors.getEditingCluster],
    (clusters, editingCluster) => {
        if (!editingCluster) {
            return null;
        }
        let result = {};
        if (!editingCluster.id) {
            return result;
        }
        result = clusters.find(obj => obj.id === editingCluster.id);
        return result || null;
    }
);

const mapStateToProps = createStructuredSelector({
    editingCluster: getEditingCluster
});

const mapDispatchToProps = dispatch => ({
    editCluster: clusterId => dispatch(clusterActions.editCluster(clusterId))
});

export default connect(mapStateToProps, mapDispatchToProps)(ClustersModal);
