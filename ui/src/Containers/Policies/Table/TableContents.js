import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { withRouter } from 'react-router';
import ReactRouterPropTypes from 'react-router-prop-types';
import { connect } from 'react-redux';
import { selectors } from 'reducers';
import { actions as backendActions } from 'reducers/policies/backend';
import { actions as pageActions } from 'reducers/policies/page';
import { actions as tableActions } from 'reducers/policies/table';
import { actions as wizardActions } from 'reducers/policies/wizard';
import Tooltip from 'rc-tooltip';
import { createStructuredSelector } from 'reselect';
import wizardStages from 'Containers/Policies/Wizard/wizardStages';

import * as Icon from 'react-feather';
import CheckboxTable from 'Components/CheckboxTable';
import { toggleRow, toggleSelectAll } from 'utils/checkboxUtils';
import {
    defaultColumnClassName,
    defaultHeaderClassName,
    wrapClassName,
    rtTrActionsClassName
} from 'Components/Table';
import { lifecycleStageLabels, severityLabels } from 'messages/common';
import { sortSeverity, sortLifecycle } from 'sorters/sorters';

const getSeverityClassName = severity => {
    switch (severity) {
        case 'Low':
            return 'px-2 rounded-full bg-base-200 border-2 border-base-300 text-base-600';
        case 'Medium':
            return 'px-2 rounded-full bg-warning-200 border-2 border-warning-300 text-warning-800';
        case 'High':
            return 'px-2 rounded-full bg-caution-200 border-2 border-caution-300 text-caution-800';
        case 'Critical':
            return 'px-2 rounded-full bg-alert-200 border-2 border-alert-300 text-alert-800';
        default:
            return '';
    }
};

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

        updatePolicyDisabledState: PropTypes.func.isRequired,
        selectPolicyId: PropTypes.func.isRequired,
        selectPolicyIds: PropTypes.func.isRequired,
        openWizard: PropTypes.func.isRequired,
        setWizardStage: PropTypes.func.isRequired,
        setWizardPolicyDisabled: PropTypes.func.isRequired,
        deletePolicies: PropTypes.func.isRequired,

        history: ReactRouterPropTypes.history.isRequired
    };

    static defaultProps = {
        wizardPolicy: null
    };

    setSelectedPolicy = policy => {
        // Add policy to history.
        const urlSuffix = `/${policy.id}`;
        this.props.history.push({
            pathname: `/main/policies${urlSuffix}`
        });

        // Select the policy so that it is highlighted in the table.
        this.props.selectPolicyId(policy.id);

        // Bring up the wizard with that policy.
        this.props.setWizardStage(wizardStages.details);
        this.props.openWizard();
    };

    toggleEnabledDisabledPolicy = ({ id, disabled }) => e => {
        e.stopPropagation();
        this.props.updatePolicyDisabledState({ policyId: id, disabled: !disabled });
        this.props.setWizardPolicyDisabled(!disabled);
    };

    toggleRow = id => {
        const selection = toggleRow(id, this.props.selectedPolicyIds);
        this.props.selectPolicyIds(selection);
    };

    toggleSelectAll = () => {
        const rowsLength = this.props.policies.length;
        const tableRef = this.checkboxTable.reactTable;
        const selection = toggleSelectAll(rowsLength, this.props.selectedPolicyIds, tableRef);
        this.props.selectPolicyIds(selection);
    };

    onDeletePolicy = ({ id }) => e => {
        e.stopPropagation();
        this.props.deletePolicies([id]);
    };

    renderRowActionButtons = policy => {
        const enableTooltip = `${policy.disabled ? 'Enable' : 'Disable'} policy`;
        const enableIconColor = policy.disabled ? 'text-primary-600' : 'text-success-600';
        const enableIconHoverColor = policy.disabled ? 'text-primary-700' : 'text-success-700';
        return (
            <div className="border-2 border-r-2 border-base-400 bg-base-100 flex">
                <Tooltip placement="top" overlay={<div>{enableTooltip}</div>} mouseLeaveDelay={0}>
                    <button
                        type="button"
                        className={`p-1 px-4 hover:bg-primary-200 ${enableIconColor} hover:${enableIconHoverColor}`}
                        onClick={this.toggleEnabledDisabledPolicy(policy)}
                    >
                        <Icon.Power className="mt-1 h-4 w-4" />
                    </button>
                </Tooltip>
                <Tooltip placement="top" overlay={<div>Delete policy</div>} mouseLeaveDelay={0}>
                    <button
                        type="button"
                        className="p-1 px-4 border-l-2 border-base-400 hover:bg-primary-200 text-primary-600 hover:text-primary-700"
                        onClick={this.onDeletePolicy(policy)}
                    >
                        <Icon.Trash2 className="mt-1 h-4 w-4" />
                    </button>
                </Tooltip>
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
                            data-test-id="enable-disable-icon"
                        />
                        <div className="pl-4" data-test-id="policy-name">
                            {original.name}
                        </div>
                    </div>
                ),
                className: `w-1/5 sticky-column left-checkbox-offset ${wrapClassName} ${defaultColumnClassName}`,
                headerClassName: `w-1/5 sticky-column left-checkbox-offset ${defaultHeaderClassName}`
            },
            {
                Header: 'Description',
                accessor: 'description',
                className: `w-1/3 ${wrapClassName} ${defaultColumnClassName}`,
                headerClassName: `w-1/3 ${defaultHeaderClassName}`
            },
            {
                Header: 'Lifecycle',
                accessor: 'lifecycleStages',
                className: `${wrapClassName} ${defaultColumnClassName}`,
                headerClassName: `${defaultHeaderClassName}`,
                Cell: ({ original }) => {
                    const { lifecycleStages } = original;
                    return lifecycleStages.map(stage => lifecycleStageLabels[stage]).join(', ');
                },
                sortMethod: sortLifecycle
            },
            {
                Header: 'Severity',
                accessor: 'severity',
                Cell: ci => {
                    const severity = severityLabels[ci.value];
                    return <span className={getSeverityClassName(severity)}>{severity}</span>;
                },
                width: 100,
                sortMethod: sortSeverity
            },
            {
                Header: '',
                accessor: '',
                headerClassName: 'hidden',
                className: rtTrActionsClassName,
                Cell: ({ original }) => this.renderRowActionButtons(original)
            }
        ];

        const id = this.props.selectedPolicyId;
        return (
            <div
                data-test-id="policies-table-container"
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
    wizardPolicy: selectors.getWizardPolicy
});

const mapDispatchToProps = {
    selectPolicyId: tableActions.selectPolicyId,
    selectPolicyIds: tableActions.selectPolicyIds,
    updatePolicyDisabledState: tableActions.updatePolicyDisabledState,
    deletePolicies: backendActions.deletePolicies,
    openWizard: pageActions.openWizard,
    setWizardStage: wizardActions.setWizardStage,
    setWizardPolicyDisabled: wizardActions.setWizardPolicyDisabled
};

export default withRouter(
    connect(
        mapStateToProps,
        mapDispatchToProps
    )(TableContents)
);
