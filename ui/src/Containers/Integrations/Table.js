import React, { Component } from 'react';
import PropTypes from 'prop-types';

import ComponentTable from 'Components/Table';
import Panel from 'Components/Panel';
import PanelButton from 'Components/PanelButton';
import * as Icon from 'react-feather';

import tableColumnDescriptor from 'Containers/Integrations/tableColumnDescriptor';

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

        setTable: PropTypes.func.isRequired
    };

    static defaultProps = {
        clusters: []
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

    getActions = () => {
        const actions = [];
        switch (this.props.source) {
            case 'authProviders':
                actions.push({
                    renderIcon: row =>
                        row.validated ? (
                            <Icon.Power className="h-5 w-4 text-success-500" />
                        ) : (
                            <Icon.Power className="h-5 w-4 text-base-600" />
                        ),
                    className: 'flex rounded-sm uppercase text-center text-sm items-center',
                    onClick: this.props.onActivate
                });
                break;
            default:
        }
        return actions;
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

    renderTableContent = () => (
        <ComponentTable
            columns={tableColumnDescriptor[this.props.source][this.props.type]}
            rows={this.getRows()}
            actions={this.getActions()}
            checkboxes
            onRowClick={this.props.onRowClick}
            ref={this.props.setTable}
            messageIfEmpty={`No ${this.props.type} integrations`}
        />
    );

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

export default Table;
