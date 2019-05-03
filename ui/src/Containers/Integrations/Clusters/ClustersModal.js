import React, { Component } from 'react';
import PropTypes from 'prop-types';
import * as Icon from 'react-feather';
import { connect } from 'react-redux';
import { createSelector, createStructuredSelector } from 'reselect';
import Tooltip from 'rc-tooltip';

import { actions, clusterTypes } from 'reducers/clusters';
import { selectors } from 'reducers';

import Dialog from 'Components/Dialog';
import Modal from 'Components/Modal';
import CheckboxTable from 'Components/CheckboxTable';
import { toggleRow, toggleSelectAll } from 'utils/checkboxUtils';
import { defaultColumnClassName, wrapClassName, rtTrActionsClassName } from 'Components/Table';
import Panel from 'Components/Panel';
import NoResultsMessage from 'Components/NoResultsMessage';
import PanelButton from 'Components/PanelButton';
import { clusterTypeLabels } from 'messages/common';
import ClusterWizardPanel from './ClusterWizardPanel';
import ClusterDetails, {
    checkInLabel,
    formatCollectionMethod,
    formatAdmissionController,
    formatLastCheckIn,
    formatSensorVersion,
    sensorVersionLabel
} from './ClusterDetails';

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

    onDeleteHandler = cluster => e => {
        e.stopPropagation();
        this.setState({ selection: [cluster.id] });
        this.showConfirmationDialog();
    };

    onClusterDetailsClose = () => this.props.selectCluster(null);

    clearSelection = () => this.setState({ selection: [] });

    deleteClusters = ({ id }) => {
        if (!id) {
            this.props.deleteClusters(this.state.selection);
            this.hideConfirmationDialog();
            this.clearSelection();
        } else this.props.deleteClusters([id]);
    };

    showConfirmationDialog = () => {
        this.setState({ showConfirmationDialog: true });
    };

    hideConfirmationDialog = () => {
        this.setState({ showConfirmationDialog: false, selection: [] });
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
                Header: 'Runtime Support',
                Cell: ({ original }) => formatCollectionMethod(original)
            }
        ];

        if (this.props.clusterType === 'KUBERNETES_CLUSTER') {
            columns.push({
                Header: 'Admission Controller',
                Cell: ({ original }) => formatAdmissionController(original)
            });
        }
        columns.push(
            ...[
                {
                    Header: checkInLabel,
                    Cell: ({ original }) => formatLastCheckIn(original)
                },
                {
                    Header: sensorVersionLabel,
                    Cell: ({ original }) => formatSensorVersion(original),
                    className: `${wrapClassName} ${defaultColumnClassName} word-break`
                },
                {
                    Header: '',
                    accessor: '',
                    headerClassName: 'hidden',
                    className: rtTrActionsClassName,
                    Cell: ({ original }) => this.renderRowActionButtons(original)
                }
            ]
        );
        const { selectedCluster, clusters } = this.props;
        const selectedClusterId = selectedCluster && selectedCluster.id;
        if (!clusters || !clusters.length)
            return <NoResultsMessage message="No clusters to show." />;

        return (
            <CheckboxTable
                ref={table => {
                    this.clusterTableRef = table;
                }}
                rows={clusters}
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

    renderRowActionButtons = cluster => (
        <div className="border-2 border-r-2 border-base-400 bg-base-100">
            <Tooltip placement="top" overlay={<div>Delete cluster</div>} mouseLeaveDelay={0}>
                <button
                    type="button"
                    className="p-1 px-4 hover:bg-primary-200 text-primary-600 hover:text-primary-700"
                    onClick={this.onDeleteHandler(cluster)}
                >
                    <Icon.Trash2 className="mt-1 h-4 w-4" />
                </button>
            </Tooltip>
        </div>
    );

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
                        className="btn btn-alert"
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
                <Panel header={headerText} headerComponents={buttons}>
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
        const confirmationText = `Cluster deletion won't tear down StackRox services running on this cluster. You can remove them from the corresponding cluster by running the "delete-sensor.sh" script from the sensor installation bundle. Are you sure you want to delete ${numCheckedClusters} cluster(s)?`;
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
                    className="w-1/3"
                    isOpen={this.state.showConfirmationDialog}
                    text={confirmationText}
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
