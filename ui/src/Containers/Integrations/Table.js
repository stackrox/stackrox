import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { selectors } from 'reducers';
import { createStructuredSelector } from 'reselect';

import CheckboxTable from 'Components/CheckboxTable';
import { rtTrActionsClassName } from 'Components/Table';
import Panel from 'Components/Panel';
import PanelButton from 'Components/PanelButton';
import * as Icon from 'react-feather';
import Tooltip from 'rc-tooltip';

import tableColumnDescriptor from 'Containers/Integrations/tableColumnDescriptor';
import NoResultsMessage from 'Components/NoResultsMessage';

class IntegrationTable extends Component {
    static propTypes = {
        integrations: PropTypes.arrayOf(PropTypes.object).isRequired,

        source: PropTypes.oneOf(['imageIntegrations', 'notifiers', 'authProviders']).isRequired,

        type: PropTypes.string.isRequired,
        visibleType: PropTypes.string.isRequired,

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
        selectedIntegrationId: null
    };

    onDeleteHandler = integration => e => {
        e.stopPropagation();
        this.props.onDelete(integration);
    };

    onActivateHandler = integration => e => {
        e.stopPropagation();
        this.props.onActivate(integration);
    };

    getPanelButtons = () => {
        const { selection, onDelete, integrations, buttonsEnabled, onAdd } = this.props;
        const selectionCount = selection.length;
        const integrationsCount = integrations.length;
        return (
            <React.Fragment>
                {selectionCount !== 0 && (
                    <PanelButton
                        icon={<Icon.Trash2 className="h-4 w-4 ml-1" />}
                        text={`Delete (${selectionCount})`}
                        className="btn btn-alert"
                        onClick={onDelete}
                        disabled={integrationsCount === 0 || !buttonsEnabled}
                    />
                )}
                {selectionCount === 0 && (
                    <PanelButton
                        icon={<Icon.Plus className="h-4 w-4 ml-1" />}
                        text="New Integration"
                        className="btn btn-base"
                        onClick={onAdd}
                        disabled={!buttonsEnabled}
                    />
                )}
            </React.Fragment>
        );
    };

    getColumns = () => {
        const { source, type } = this.props;
        const columns = [...tableColumnDescriptor[source][type]];
        columns.push({
            Header: '',
            accessor: '',
            headerClassName: 'hidden',
            className: rtTrActionsClassName,
            Cell: ({ original }) => this.renderRowActionButtons(original)
        });
        return columns;
    };

    renderRowActionButtons = integration => {
        const { source } = this.props;
        let activateBtn = null;
        if (source === 'authProviders') {
            const enableTooltip = `${!integration.validated ? 'Enable' : 'Disable'} auth provider`;
            const enableIconColor = integration.disabled ? 'text-primary-600' : 'text-success-600';
            const enableIconHoverColor = integration.disabled
                ? 'text-primary-700'
                : 'text-success-700';
            activateBtn = (
                <Tooltip placement="top" overlay={<div>{enableTooltip}</div>} mouseLeaveDelay={0}>
                    <button
                        type="button"
                        className={`p-1 px-4 hover:bg-primary-200 ${enableIconColor} hover:${enableIconHoverColor}`}
                        onClick={this.onActivateHandler(integration)}
                    >
                        <Icon.Power className="mt-1 h-4 w-4" />
                    </button>
                </Tooltip>
            );
        }
        return (
            <div className="border-2 border-r-2 border-base-400 bg-base-100 flex">
                {activateBtn}
                <Tooltip
                    placement="top"
                    overlay={<div>Delete integration</div>}
                    mouseLeaveDelay={0}
                >
                    <button
                        type="button"
                        className={`p-1 px-4 ${
                            source === 'authProviders' ? 'border-l-2 border-base-400' : ''
                        } hover:bg-primary-200 text-primary-600 hover:text-primary-700`}
                        onClick={this.onDeleteHandler(integration)}
                    >
                        <Icon.Trash2 className="mt-1 h-4 w-4" />
                    </button>
                </Tooltip>
            </div>
        );
    };

    renderTableContent = () => {
        const rows = this.props.integrations;

        let { visibleType } = this.props;
        if (visibleType === undefined) {
            visibleType = this.props.type;
        }

        if (!rows.length) return <NoResultsMessage message={`No ${visibleType} integrations`} />;
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
                noDataText={`No ${visibleType} integrations`}
                minRows={20}
            />
        );
    };

    render() {
        const { type, selection, integrations } = this.props;
        let { visibleType } = this.props;
        if (visibleType === undefined) {
            visibleType = type;
        }
        const selectionCount = selection.length;
        const integrationsCount = integrations.length;
        const headerText =
            selectionCount !== 0
                ? `${selectionCount} ${visibleType} Integration${
                      selectionCount === 1 ? '' : 's'
                  } selected`
                : `${integrationsCount} ${visibleType} Integration${
                      integrationsCount === 1 ? '' : 's'
                  }`;
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
