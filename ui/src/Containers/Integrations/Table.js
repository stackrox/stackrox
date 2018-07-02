import React, { Component } from 'react';
import PropTypes from 'prop-types';

import ComponentTable from 'Components/Table';
import Panel from 'Components/Panel';
import PanelButton from 'Components/PanelButton';
import * as Icon from 'react-feather';

import tableColumnDescriptor from 'Containers/Integrations/tableColumnDescriptor';

class Table extends Component {
    static propTypes = {
        integrations: PropTypes.arrayOf(
            PropTypes.shape({
                type: PropTypes.string.isRequired
            })
        ).isRequired,

        source: PropTypes.oneOf(['imageIntegrations', 'notifiers', 'authProviders']).isRequired,
        type: PropTypes.string.isRequired,

        buttonsEnabled: PropTypes.bool.isRequired,

        onRowClick: PropTypes.func.isRequired,
        onActivate: PropTypes.func.isRequired,
        onAdd: PropTypes.func.isRequired,
        onDelete: PropTypes.func.isRequired,

        setTable: PropTypes.func.isRequired
    };

    getPanelButtons = () => (
        <React.Fragment>
            <PanelButton
                icon={<Icon.Trash2 className="h-4 w-4" />}
                text="Delete"
                className="btn-danger"
                onClick={this.props.onDelete}
                disabled={!this.props.buttonsEnabled}
            />
            <PanelButton
                icon={<Icon.Plus className="h-4 w-4" />}
                text="Add"
                className="btn-success"
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

    renderEmpty = () => (
        <div className="p3 w-full my-auto text-center capitalize">
            {`No ${this.props.type} integrations`}
        </div>
    );

    renderTableContent = () => (
        <ComponentTable
            columns={tableColumnDescriptor[this.props.source][this.props.type]}
            rows={this.props.integrations}
            actions={this.getActions()}
            checkboxes
            onRowClick={this.props.onRowClick}
            ref={this.props.setTable}
        />
    );

    render() {
        return (
            <div className="flex flex-1">
                <Panel header={`${this.props.type} Integration`} buttons={this.getPanelButtons()}>
                    {this.props.integrations.length !== 0
                        ? this.renderTableContent()
                        : this.renderEmpty()}
                </Panel>
            </div>
        );
    }
}

export default Table;
