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

class Table extends Component {
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
        selectedIntegrationId: PropTypes.string
    };

    static defaultProps = {
        clusters: [],
        selectedIntegrationId: null
    };

    getPanelButtons = () => (
        <React.Fragment>
            <PanelButton
                icon={<Icon.Trash2 className="h-4 w-4" />}
                text="Delete"
                className="btn btn-danger"
                onClick={this.props.onDelete}
                disabled={this.props.integrations.length === 0 || !this.props.buttonsEnabled}
            />
            <PanelButton
                icon={<Icon.Plus className="h-4 w-4" />}
                text="Add"
                className="btn btn-success"
                onClick={this.props.onAdd}
                disabled={!this.props.buttonsEnabled}
            />
        </React.Fragment>
    );

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
                selectedRowId={this.props.selectedIntegrationId}
                noDataText={`No ${this.props.type} integrations`}
                minRows={20}
            />
        );
    };

    render() {
        return (
            <div className="flex flex-1">
                <Panel header={`${this.props.type} Integration`} buttons={this.getPanelButtons()}>
                    {this.renderTableContent()}
                </Panel>
            </div>
        );
    }
}

const mapStateToProps = createStructuredSelector({
    clusters: selectors.getClusters
});

export default connect(mapStateToProps)(Table);
