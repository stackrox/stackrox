import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { withRouter } from 'react-router-dom';
import * as Icon from 'react-feather';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';

import { selectors } from 'reducers';
import { actions as backendActions } from 'reducers/policies/backend';
import { actions as tableActions } from 'reducers/policies/table';
import { actions as wizardActions } from 'reducers/policies/wizard';

import wizardStages from 'Containers/Policies/Wizard/wizardStages';
import CheckboxTable from 'Components/CheckboxTable';
import SeverityLabel from 'Components/SeverityLabel';
import RowActionButton from 'Components/RowActionButton';
import {
    defaultColumnClassName,
    defaultHeaderClassName,
    wrapClassName,
    rtTrActionsClassName,
} from 'Components/Table';
import { lifecycleStageLabels } from 'messages/common';
import { sortAscii, sortSeverity, sortLifecycle } from 'sorters/sorters';
import { toggleRow, toggleSelectAll } from 'utils/checkboxUtils';

// TableContents are the policy rows.
class TableContents extends Component {
    static propTypes = {
        page: PropTypes.number.isRequired,
        policies: PropTypes.arrayOf(PropTypes.object).isRequired,
        selectedPolicyId: PropTypes.string.isRequired,
        selectedPolicyIds: PropTypes.arrayOf(PropTypes.string).isRequired,
        wizardOpen: PropTypes.bool.isRequired,
        wizardStage: PropTypes.string.isRequired,
        wizardPolicy: PropTypes.shape({}),

        setSelectedPolicy: PropTypes.func.isRequired,
        updatePolicyDisabledState: PropTypes.func.isRequired,
        selectPolicyIds: PropTypes.func.isRequired,
        setWizardPolicyDisabled: PropTypes.func.isRequired,
        deletePolicies: PropTypes.func.isRequired,
    };

    static defaultProps = {
        wizardPolicy: null,
    };

    setSelectedPolicy = (policy) => {
        this.props.setSelectedPolicy(policy.id);
    };

    toggleEnabledDisabledPolicy = ({ id, disabled }) => (e) => {
        e.stopPropagation();
        this.props.updatePolicyDisabledState({
            policyId: id,
            disabled: !disabled,
        });
        this.props.setWizardPolicyDisabled(!disabled);
    };

    toggleRow = (id) => {
        const selection = toggleRow(id, this.props.selectedPolicyIds);
        this.props.selectPolicyIds(selection);
    };

    toggleSelectAll = () => {
        const rowsLength = this.props.policies.length;
        const tableRef = this.checkboxTable.reactTable;
        const selection = toggleSelectAll(rowsLength, this.props.selectedPolicyIds, tableRef);
        this.props.selectPolicyIds(selection);
    };

    onDeletePolicy = ({ id }) => (e) => {
        e.stopPropagation();
        this.props.deletePolicies([id]);
    };

    renderRowActionButtons = (policy) => {
        const enableTooltip = `${policy.disabled ? 'Enable' : 'Disable'} policy`;
        const enableIconColor = policy.disabled ? 'text-primary-600' : 'text-success-600';
        const enableIconHoverColor = policy.disabled ? 'text-primary-700' : 'text-success-700';
        return (
            <div className="border-2 border-r-2 border-base-400 bg-base-100 flex">
                <RowActionButton
                    text={enableTooltip}
                    onClick={this.toggleEnabledDisabledPolicy(policy)}
                    className={`hover:bg-primary-200 ${enableIconColor} hover:${enableIconHoverColor}`}
                    icon={<Icon.Power className="h-4 w-4" />}
                />
                <RowActionButton
                    text="Delete policy"
                    onClick={this.onDeletePolicy(policy)}
                    border="border-l-2 border-base-400"
                    icon={<Icon.Trash2 className="h-4 w-4" />}
                />
            </div>
        );
    };

    render() {
        const columns = [
            {
                Header: 'Name',
                accessor: 'name',
                Cell: ({ original }) => (
                    <div className="flex items-center relative">
                        <div
                            className={`h-2 w-2 rounded-lg absolute ${
                                !original.disabled ? 'bg-success-500' : 'bg-base-300'
                            }`}
                            data-testid="enable-disable-icon"
                        />
                        <div
                            className={`pl-4 ${
                                original.notifiers.length === 0
                                    ? 'text-base-400'
                                    : 'text-success-600'
                            }`}
                            title={`Notification ${original.notifiers.length === 0 ? 'Off' : 'On'}`}
                        >
                            <Icon.Bell
                                className="h-2 w-2"
                                hidden={original.notifiers.length === 0}
                            />
                            <Icon.BellOff
                                className="h-2 w-2"
                                hidden={original.notifiers.length !== 0}
                            />
                        </div>
                        <div className="pl-4" data-testid="policy-name">
                            {original.name}
                        </div>
                    </div>
                ),
                sortMethod: sortAscii,
                className: `w-1/5 left-checkbox-offset ${wrapClassName} ${defaultColumnClassName}`,
                headerClassName: `w-1/5 left-checkbox-offset ${defaultHeaderClassName}`,
            },
            {
                Header: 'Description',
                accessor: 'description',
                className: `w-1/3 ${wrapClassName} ${defaultColumnClassName}`,
                headerClassName: `w-1/3 ${defaultHeaderClassName}`,
            },
            {
                Header: 'Lifecycle',
                accessor: 'lifecycleStages',
                className: `${wrapClassName} ${defaultColumnClassName}`,
                headerClassName: `${defaultHeaderClassName}`,
                Cell: ({ original }) => {
                    const { lifecycleStages } = original;
                    return lifecycleStages.map((stage) => lifecycleStageLabels[stage]).join(', ');
                },
                sortMethod: sortLifecycle,
            },
            {
                Header: 'Severity',
                accessor: 'severity',
                Cell: (ci) => {
                    const severity = ci.value;
                    return <SeverityLabel severity={severity} />;
                },
                width: 100,
                sortMethod: sortSeverity,
            },
            {
                Header: '',
                accessor: '',
                headerClassName: 'hidden',
                className: rtTrActionsClassName,
                Cell: ({ original }) => this.renderRowActionButtons(original),
            },
        ];

        const id = this.props.selectedPolicyId;
        return (
            <div
                data-testid="policies-table-container"
                className={`w-full
                    ${
                        this.props.wizardOpen && this.props.wizardStage !== wizardStages.details
                            ? 'pointer-events-none opacity-25'
                            : ''
                    }`}
            >
                <CheckboxTable
                    ref={r => (this.checkboxTable = r)} // eslint-disable-line
                    rows={this.props.policies}
                    columns={columns}
                    onRowClick={this.setSelectedPolicy}
                    toggleRow={this.toggleRow}
                    toggleSelectAll={this.toggleSelectAll}
                    selection={this.props.selectedPolicyIds}
                    selectedRowId={id}
                    noDataText="No results found. Please refine your search."
                    page={this.props.page}
                    defaultSorted={[
                        {
                            id: 'name',
                            desc: false,
                        },
                    ]}
                />
            </div>
        );
    }
}

const mapStateToProps = createStructuredSelector({
    page: selectors.getTablePage,
    policies: selectors.getFilteredPolicies,
    selectedPolicyId: selectors.getSelectedPolicyId,
    selectedPolicyIds: selectors.getSelectedPolicyIds,
    wizardOpen: selectors.getWizardOpen,
    wizardStage: selectors.getWizardStage,
    wizardPolicy: selectors.getWizardPolicy,
});

const mapDispatchToProps = {
    selectPolicyIds: tableActions.selectPolicyIds,
    updatePolicyDisabledState: tableActions.updatePolicyDisabledState,
    deletePolicies: backendActions.deletePolicies,
    setWizardPolicyDisabled: wizardActions.setWizardPolicyDisabled,
};

export default withRouter(connect(mapStateToProps, mapDispatchToProps)(TableContents));
