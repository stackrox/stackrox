import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { selectors } from 'reducers';
import { createStructuredSelector } from 'reselect';

import CheckboxTable from 'Components/CheckboxTable';
import Panel from 'Components/Panel';
import PanelButton from 'Components/PanelButton';
import * as Icon from 'react-feather';

import tableColumnDescriptor from 'Containers/Integrations/tableColumnDescriptor';
import NoResultsMessage from 'Components/NoResultsMessage';

class IntegrationTable extends Component {
    static propTypes = {
        integrations: PropTypes.arrayOf(PropTypes.object).isRequired,

        source: PropTypes.oneOf([
            'dnrIntegrations',
            'imageIntegrations',
            'notifiers',
            'authProviders'
        ]).isRequired,

        type: PropTypes.string.isRequired,
        clusters: PropTypes.arrayOf(
            PropTypes.shape({
                name: PropTypes.string.isRequired,
                id: PropTypes.string.isRequired
            })
        ),

        buttonsEnabled: PropTypes.bool.isRequired,

        onRowClick: PropTypes.func.isRequired,
        onActivate: PropTypes.func.isRequired,
        onAdd: PropTypes.func.isRequired,
        onDelete: PropTypes.func.isRequired,

        setTable: PropTypes.func.isRequired,
        selectedIntegrationId: PropTypes.string,
        toggleRow: PropTypes.func.isRequired,
        toggleSelectAll: PropTypes.func.isRequired,
        selection: PropTypes.arrayOf(PropTypes.string).isRequired
    };

    static defaultProps = {
        clusters: [],
        selectedIntegrationId: null
    };

    getPanelButtons = () => {
        const selectionCount = this.props.selection.length;
        const integrationsCount = this.props.integrations.length;
        return (
            <React.Fragment>
                {selectionCount !== 0 && (
                    <PanelButton
                        icon={<Icon.Trash2 className="h-4 w-4 ml-1" />}
                        text={`Delete (${selectionCount})`}
                        className="btn btn-danger"
                        onClick={this.props.onDelete}
                        disabled={integrationsCount === 0 || !this.props.buttonsEnabled}
                    />
                )}
                {selectionCount === 0 && (
                    <PanelButton
                        icon={<Icon.Plus className="h-4 w-4 ml-1" />}
                        text="New Integration"
                        className="btn btn-base"
                        onClick={this.props.onAdd}
                        disabled={!this.props.buttonsEnabled}
                    />
                )}
            </React.Fragment>
        );
    };

    getColumns = () => {
        const columns = [...tableColumnDescriptor[this.props.source][this.props.type]];
        if (this.props.source === 'authProviders') {
            columns.push({
                Header: 'Actions',
                accessor: '',
                Cell: ({ original }) => (
                    <button
                        className="flex rounded-sm uppercase text-center text-sm items-center self-center"
                        onClick={this.props.onActivate(original)}
                    >
                        {original.validated && <Icon.Power className="h-5 w-4 text-success-500" />}
                        {!original.validated && <Icon.Power className="h-5 w-4 text-base-600" />}
                    </button>
                ),
                width: 75
            });
        }
        return columns;
    };

    getRows = () => {
        if (this.props.source === 'dnrIntegrations') {
            return this.props.integrations.map(dnrIntegration => {
                const clusterNames = [];
                dnrIntegration.clusterIds.forEach(clusterId => {
                    const cluster = this.props.clusters.find(({ id }) => id === clusterId);
                    if (cluster && cluster.name) clusterNames.push(cluster.name);
                });
                return Object.assign({}, dnrIntegration, {
                    clusterNames: clusterNames.join(', '),
                    name: `D&R Integration`
                });
            });
        }
        return this.props.integrations;
    };

    renderTableContent = () => {
        const rows = this.getRows();

        if (!rows.length)
            return <NoResultsMessage message={`No ${this.props.type} integrations`} />;
        return (
            <CheckboxTable
                ref={this.props.setTable}
                rows={rows}
                columns={this.getColumns()}
                onRowClick={this.props.onRowClick}
                toggleRow={this.props.toggleRow}
                toggleSelectAll={this.props.toggleSelectAll}
                selection={this.props.selection}
                selectedRowId={this.props.selectedIntegrationId}
                noDataText={`No ${this.props.type} integrations`}
                minRows={20}
            />
        );
    };

    render() {
        const { type, selection, integrations } = this.props;
        const selectionCount = selection.length;
        const integrationsCount = integrations.length;
        const headerText =
            selectionCount !== 0
                ? `${selectionCount} ${type} Integration${selectionCount === 1 ? '' : 's'} selected`
                : `${integrationsCount} ${type} Integration${integrationsCount === 1 ? '' : 's'}`;
        return (
            <div className="flex flex-1">
                <Panel header={headerText} buttons={this.getPanelButtons()}>
                    {this.renderTableContent()}
                </Panel>
            </div>
        );
    }
}

const mapStateToProps = createStructuredSelector({
    clusters: selectors.getClusters
});

export default connect(mapStateToProps)(IntegrationTable);
