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
import CheckboxTable from 'Components/CheckboxTable';
import { toggleRow, toggleSelectAll } from 'utils/checkboxUtils';
import { defaultColumnClassName, wrapClassName } from 'Components/Table';
import Panel from 'Components/Panel';
import NoResultsMessage from 'Components/NoResultsMessage';
import PanelButton from 'Components/PanelButton';
import { clusterTypeLabels } from 'messages/common';
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
        deleteClusters: PropTypes.func.isRequired
    };

    static defaultProps = {
        selectedCluster: null
    };

    state = {
        showConfirmationDialog: false,
        selection: []
    };

    componentWillUnmount() {
        this.props.selectCluster(null);
    }

    onClusterRowClick = cluster => this.props.selectCluster(cluster.id);

    onAddCluster = () => this.props.startWizard();

    onClusterDetailsClose = () => this.props.selectCluster(null);

    clearSelection = () => this.setState({ selection: [] });

    deleteClusters = () => {
        if (this.state.selection.length === 0) return;
        this.props.deleteClusters(this.state.selection);
        this.hideConfirmationDialog();
        this.clearSelection();
    };

    showConfirmationDialog = () => {
        this.setState({ showConfirmationDialog: true });
    };

    hideConfirmationDialog = () => {
        this.setState({ showConfirmationDialog: false });
    };

    toggleRow = id => {
        const selection = toggleRow(id, this.state.selection);
        this.updateSelection(selection);
    };

    toggleSelectAll = () => {
        const rowsLength = this.props.clusters.length;
        const tableRef = this.clusterTableRef.reactTable;
        const selection = toggleSelectAll(rowsLength, this.state.selection, tableRef);
        this.updateSelection(selection);
    };

    updateSelection = selection => this.setState({ selection });

    showModalView = () => {
        const columns = [
            {
                accessor: 'name',
                Header: 'Name',
                className: `${wrapClassName} ${defaultColumnClassName}`
            },
            {
                accessor: 'preventImage',
                Header: 'StackRox Image',
                className: `${wrapClassName} ${defaultColumnClassName}`
            },
            {
                accessor: 'lastContact',
                Header: 'Last Check-In',
                Cell: ({ original }) => {
                    if (original.lastContact)
                        return dateFns.format(original.lastContact, dateTimeFormat);
                    return 'N/A';
                }
            }
        ];
        const { selectedCluster } = this.props;
        const selectedClusterId = selectedCluster && selectedCluster.id;
        if (!this.props.clusters || !this.props.clusters.length)
            return <NoResultsMessage message="No clusters to show." />;

        return (
            <CheckboxTable
                ref={table => {
                    this.clusterTableRef = table;
                }}
                rows={this.props.clusters}
                columns={columns}
                onRowClick={this.onClusterRowClick}
                toggleRow={this.toggleRow}
                toggleSelectAll={this.toggleSelectAll}
                selection={this.state.selection}
                selectedRowId={selectedClusterId}
                noDataText="No clusters to show."
                minRows={20}
            />
        );
    };

    renderTable = () => {
        const { clusterType, selectedCluster, clusters, isWizardActive } = this.props;
        const cluster = clusterTypeLabels[clusterType];
        const selectionCount = this.state.selection.length;
        const clusterCount = clusters.length;
        const headerText =
            selectionCount !== 0
                ? `${selectionCount} ${cluster} Integration${
                      selectionCount === 1 ? `` : `s`
                  } Selected`
                : `${clusterCount} ${cluster} Integration${clusterCount === 1 ? `` : `s`}`;
        const buttons = (
            <React.Fragment>
                {selectionCount !== 0 && (
                    <PanelButton
                        icon={<Icon.Trash2 className="h-4 w-4 ml-1" />}
                        text={`Delete (${selectionCount})`}
                        className="btn btn-danger"
                        onClick={this.showConfirmationDialog}
                        disabled={clusterCount === 0 || selectedCluster !== null}
                    />
                )}
                {selectionCount === 0 && (
                    <PanelButton
                        icon={<Icon.Plus className="h-4 w-4 ml-1" />}
                        text="New Cluster"
                        className="btn btn-base"
                        onClick={this.onAddCluster}
                        disabled={selectedCluster !== null || isWizardActive}
                    />
                )}
            </React.Fragment>
        );
        return (
            <div className="flex flex-1">
                <Panel header={headerText} buttons={buttons}>
                    {this.showModalView()}
                </Panel>
            </div>
        );
    };

    renderSidePanel() {
        const { isWizardActive, clusterType, selectedCluster } = this.props;
        if (isWizardActive) {
            return (
                <div className="flex w-1/2">
                    <ClusterWizardPanel clusterType={clusterType} />
                </div>
            );
        }
        if (!selectedCluster) return null;
        return (
            <Panel
                className="w-1/2"
                header={selectedCluster.name}
                onClose={this.onClusterDetailsClose}
            >
                <ClusterDetails cluster={selectedCluster} />
            </Panel>
        );
    }

    render() {
        const { clusterType, onRequestClose } = this.props;
        const numCheckedClusters = this.state.selection.length;
        return (
            <Modal isOpen onRequestClose={onRequestClose} className="w-full lg:w-5/6 h-full">
                <header className="flex items-center w-full p-4 bg-primary-500 text-base-100 uppercase">
                    <span className="flex flex-1">{clusterTypeLabels[clusterType]}</span>
                    <Icon.X className="h-4 w-4 cursor-pointer" onClick={onRequestClose} />
                </header>
                <div className="flex flex-1 w-full bg-base-100">
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
    selectCluster: actions.selectCluster,
    deleteClusters: actions.deleteClusters,
    startWizard: actions.startWizard
};

export default connect(
    mapStateToProps,
    mapDispatchToProps
)(ClustersModal);
