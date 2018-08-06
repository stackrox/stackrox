import React, { Component } from 'react';
import PropTypes from 'prop-types';
import * as Icon from 'react-feather';
import { connect } from 'react-redux';
import { createSelector, createStructuredSelector } from 'reselect';
import dateFns from 'date-fns';

import dateTimeFormat from 'constants/dateTimeFormat';
import { actions, clusterTypes } from 'reducers/clusters';
import { selectors } from 'reducers';

import Dialog from 'Components/Dialog';
import Modal from 'Components/Modal';
import Table from 'Components/Table';
import Panel from 'Components/Panel';
import PanelButton from 'Components/PanelButton';
import { clusterTypeLabels } from 'messages/common';
import { deleteCluster } from 'services/ClustersService';
import ClusterWizardPanel from './ClusterWizardPanel';
import ClusterDetails from './ClusterDetails';

class ClustersModal extends Component {
    static propTypes = {
        clusterType: PropTypes.oneOf(clusterTypes).isRequired,
        clusters: PropTypes.arrayOf(PropTypes.object).isRequired,
        selectedCluster: PropTypes.shape({
            id: PropTypes.string.isRequired,
            name: PropTypes.string.isRequired
        }),
        onRequestClose: PropTypes.func.isRequired,
        isWizardActive: PropTypes.bool.isRequired,
        startWizard: PropTypes.func.isRequired,
        selectCluster: PropTypes.func.isRequired,
        fetchClusters: PropTypes.func.isRequired
    };

    static defaultProps = {
        selectedCluster: null
    };

    state = {
        checkedClusterIds: [],
        showConfirmationDialog: false
    };

    componentWillUnmount() {
        this.props.selectCluster(null);
    }

    onRowChecked = selectedRows => this.setState({ checkedClusterIds: selectedRows });

    onClusterRowClick = cluster => this.props.selectCluster(cluster.id);

    onAddCluster = () => this.props.startWizard();

    onClusterDetailsClose = () => this.props.selectCluster(null);

    deleteClusters = () => {
        const promises = this.state.checkedClusterIds.map(deleteCluster);
        Promise.all(promises).then(() => {
            this.clusterTableRef.clearSelectedRows();
            this.hideConfirmationDialog();
            this.props.fetchClusters();
        });
    };

    showConfirmationDialog = () => {
        this.setState({ showConfirmationDialog: true });
    };

    hideConfirmationDialog = () => {
        this.setState({ showConfirmationDialog: false });
    };

    renderTable = () => {
        const header = `${clusterTypeLabels[this.props.clusterType]} Integrations`;
        const buttons = (
            <React.Fragment>
                <PanelButton
                    icon={<Icon.Trash2 className="h-4 w-4" />}
                    text="Delete"
                    className="btn btn-danger"
                    onClick={this.showConfirmationDialog}
                    disabled={this.state.checkedClusterIds.length === 0}
                />
                <PanelButton
                    icon={<Icon.Plus className="h-4 w-4" />}
                    text="Add"
                    className="btn btn-success"
                    onClick={this.onAddCluster}
                    disabled={
                        this.state.checkedClusterIds.length !== 0 || this.props.isWizardActive
                    }
                />
            </React.Fragment>
        );

        const columns = [
            { key: 'name', label: 'Name', className: 'word-break' },
            { key: 'preventImage', label: 'StackRox Image', className: 'word-break' },
            {
                key: 'lastContact',
                label: 'Last Check-In',
                keyValueFunc: date => {
                    if (date) return dateFns.format(date, dateTimeFormat);
                    return 'N/A';
                }
            }
        ];

        return (
            <div className="flex flex-1">
                <Panel header={header} buttons={buttons}>
                    <Table
                        columns={columns}
                        rows={this.props.clusters}
                        checkboxes
                        onRowClick={this.onClusterRowClick}
                        onRowChecked={this.onRowChecked}
                        ref={table => {
                            this.clusterTableRef = table;
                        }}
                    />
                </Panel>
            </div>
        );
    };

    renderSidePanel() {
        if (this.props.isWizardActive) {
            return (
                <div className="flex w-1/2">
                    <ClusterWizardPanel clusterType={this.props.clusterType} />
                </div>
            );
        }
        if (!this.props.selectedCluster) return null;
        return (
            <Panel
                className="w-1/2"
                header={this.props.selectedCluster.name}
                onClose={this.onClusterDetailsClose}
            >
                <ClusterDetails cluster={this.props.selectedCluster} />
            </Panel>
        );
    }

    render() {
        const { clusterType, onRequestClose } = this.props;
        const numCheckedClusters = this.state.checkedClusterIds.length;
        return (
            <Modal isOpen onRequestClose={onRequestClose} className="w-full lg:w-5/6 h-full">
                <header className="flex items-center w-full p-4 bg-primary-500 text-white uppercase">
                    <span className="flex flex-1">{clusterTypeLabels[clusterType]}</span>
                    <Icon.X className="h-4 w-4 cursor-pointer" onClick={onRequestClose} />
                </header>
                <div className="flex flex-1 w-full bg-white">
                    {this.renderTable()}
                    {this.renderSidePanel()}
                </div>
                <Dialog
                    isOpen={this.state.showConfirmationDialog}
                    text={`Are you sure you want to delete ${numCheckedClusters} cluster(s)?`}
                    onConfirm={this.deleteClusters}
                    onCancel={this.hideConfirmationDialog}
                />
            </Modal>
        );
    }
}

const getSelectedCluster = createSelector(
    [selectors.getClusters, selectors.getSelectedClusterId],
    (clusters, id) => clusters.find(cluster => cluster.id === id)
);

const getClusters = createSelector(
    [selectors.getClusters, (state, { clusterType }) => clusterType],
    (clusters, type) => clusters.filter(cluster => cluster.type === type)
);

const mapStateToProps = createStructuredSelector({
    clusters: getClusters,
    selectedCluster: getSelectedCluster,
    isWizardActive: state => !!selectors.getWizardCurrentPage(state)
});

const mapDispatchToProps = {
    fetchClusters: actions.fetchClusters.request,
    selectCluster: actions.selectCluster,
    startWizard: actions.startWizard
};

export default connect(mapStateToProps, mapDispatchToProps)(ClustersModal);
