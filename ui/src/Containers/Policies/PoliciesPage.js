import React, { Component } from 'react';
import PropTypes from 'prop-types';
import ReactRouterPropTypes from 'react-router-prop-types';
import { connect } from 'react-redux';
import { selectors } from 'reducers';
import { actions as policyActions, types } from 'reducers/policies';
import { actions as notificationActions } from 'reducers/notifications';
import { createSelector, createStructuredSelector } from 'reselect';

import { formValueSelector } from 'redux-form';
import * as Icon from 'react-feather';
import Dialog from 'Components/Dialog';
import Loader from 'Components/Loader';
import CheckboxTable from 'Components/CheckboxTable';
import { toggleRow, toggleSelectAll } from 'utils/checkboxUtils';
import {
    defaultColumnClassName,
    defaultHeaderClassName,
    wrapClassName,
    pageSize
} from 'Components/Table';
import Panel from 'Components/Panel';
import PanelButton from 'Components/PanelButton';
import { formatPolicyFields, getPolicyFormDataKeys } from 'Containers/Policies/policyFormUtils';
import PolicyDetails from 'Containers/Policies/PolicyDetails';
import PageHeader from 'Components/PageHeader';
import SearchInput from 'Components/SearchInput';
import TablePagination from 'Components/TablePagination';

import { severityLabels } from 'messages/common';
import { sortSeverity } from 'sorters/sorters';
import PolicyCreationWizard from 'Containers/Policies/PolicyCreationWizard';
import NoResultsMessage from 'Components/NoResultsMessage';

const getSeverityClassName = severity => {
    switch (severity) {
        case 'Low':
            return 'text-low-500';
        case 'Medium':
            return 'text-medium-500';
        case 'High':
            return 'text-high-severity';
        case 'Critical':
            return 'text-critical-severity';
        default:
            return '';
    }
};

class PoliciesPage extends Component {
    static propTypes = {
        policies: PropTypes.arrayOf(PropTypes.object).isRequired,
        selectedPolicy: PropTypes.shape({
            id: PropTypes.string.isRequired
        }),
        reassessPolicies: PropTypes.func.isRequired,
        deletePolicies: PropTypes.func.isRequired,
        updatePolicyDisabledState: PropTypes.func.isRequired,
        formData: PropTypes.shape({
            name: PropTypes.string
        }),
        wizardState: PropTypes.shape({
            current: PropTypes.string,
            policy: PropTypes.shape({}),
            isNew: PropTypes.bool,
            disabled: PropTypes.bool
        }).isRequired,
        setWizardState: PropTypes.func.isRequired,
        history: ReactRouterPropTypes.history.isRequired,
        match: ReactRouterPropTypes.match.isRequired,
        searchOptions: PropTypes.arrayOf(PropTypes.object).isRequired,
        searchModifiers: PropTypes.arrayOf(PropTypes.object).isRequired,
        searchSuggestions: PropTypes.arrayOf(PropTypes.object).isRequired,
        setSearchOptions: PropTypes.func.isRequired,
        setSearchModifiers: PropTypes.func.isRequired,
        setSearchSuggestions: PropTypes.func.isRequired,
        isViewFiltered: PropTypes.bool.isRequired,
        isFetchingPolicy: PropTypes.bool,
        addToast: PropTypes.func.isRequired,
        removeToast: PropTypes.func.isRequired
    };

    static defaultProps = {
        selectedPolicy: null,
        formData: {
            name: ''
        },
        isFetchingPolicy: false
    };

    constructor(props) {
        super(props);

        this.state = {
            page: 0,
            showConfirmationDialog: false,
            selection: []
        };
    }

    componentWillUnmount() {
        this.props.setWizardState({ current: '' });
    }

    onSearch = searchOptions => {
        if (searchOptions.length && !searchOptions[searchOptions.length - 1].type) {
            this.clearSelection();
            this.props.history.push('/main/policies');
        }
    };

    onSubmit = () => {
        const { selectedPolicy } = this.props;
        const { isNew, policy, disabled } = this.props.wizardState;
        const newPolicy = Object.assign({}, selectedPolicy, policy);
        const newState = {};
        newState.current = isNew ? 'CREATE' : 'SAVE';
        newState.policy = newPolicy;
        if (disabled) newState.policy.disabled = disabled;
        this.props.setWizardState(newState);
    };

    onPolicyEdit = () => {
        const wizardState = { current: 'EDIT', policy: null };
        this.props.setWizardState(wizardState);
    };

    onBackToEditFields = () => this.props.setWizardState({ current: 'EDIT' });

    setTablePage = newPage => {
        this.setState({ page: newPage });
    };

    getPolicyDryRun = () => {
        const dryRunOK = this.checkPreDryRun();
        if (dryRunOK) {
            const serverFormattedPolicy = formatPolicyFields(this.props.formData);
            const enabledPolicy = Object.assign({}, serverFormattedPolicy);
            // set disabled to false for dryrun so that we can see what deployments the policy will affect
            enabledPolicy.disabled = false;

            const wizardState = {
                current: 'PRE_PREVIEW',
                policy: enabledPolicy,
                disabled: serverFormattedPolicy.disabled
            };
            this.props.setWizardState(wizardState);
        }
    };

    setSelectedPolicy = policy => {
        const urlSuffix = policy && policy.id ? `/${policy.id}` : '';
        this.props.history.push({
            pathname: `/main/policies${urlSuffix}`
        });
        this.props.setWizardState({ current: '', isNew: false });
    };

    getTableHeaderText = () => {
        const selectionCount = this.state.selection.length;
        const rowCount = this.props.policies.length;
        return selectionCount !== 0
            ? `${selectionCount} ${selectionCount === 1 ? 'Policy' : 'Policies'} Selected`
            : `${rowCount} ${rowCount === 1 ? 'Policy' : 'Policies'} ${
                  this.props.isViewFiltered ? 'Matched' : ''
              }`;
    };

    checkPreDryRun = () => {
        if (!this.props.wizardState.isNew) return true;
        // throw an error if adding new policy that has the same name
        const policyNames = this.props.policies.map(policy => policy.name);
        if (policyNames.find(name => name === this.props.formData.name)) {
            const error = `Could not add policy due to name validation: "${
                this.props.formData.name
            }" already exists`;
            this.showToast(error);
            return false;
        }
        return true;
    };

    showToast = error => {
        this.props.addToast(error);
        setTimeout(this.props.removeToast, 500);
    };

    clearSelection = () => this.setState({ selection: [] });

    deletePolicies = () => {
        const policyIds = [];
        this.state.selection.forEach(rowId => {
            // close the view panel if that policy is being deleted
            if (rowId === this.props.match.params.policyId) {
                this.setSelectedPolicy();
            }
            policyIds.push(rowId);
        });
        this.clearSelection();
        this.hideConfirmationDialog();
        this.props.deletePolicies(policyIds);
    };

    addPolicy = () => {
        this.setSelectedPolicy();
        this.props.setWizardState({ current: 'EDIT', policy: null, isNew: true });
    };

    toggleEnabledDisabledPolicy = ({ id, disabled }) => e => {
        e.stopPropagation();
        this.props.updatePolicyDisabledState({ policyId: id, disabled: !disabled });
    };

    showConfirmationDialog = () => {
        this.setState({ showConfirmationDialog: true });
    };

    hideConfirmationDialog = () => {
        this.setState({ showConfirmationDialog: false });
    };

    updateSelection = selection => this.setState({ selection });

    toggleRow = id => {
        const selection = toggleRow(id, this.state.selection);
        this.updateSelection(selection);
    };

    toggleSelectAll = () => {
        const rowsLength = this.props.policies.length;
        const tableRef = this.checkboxTable.reactTable;
        const selection = toggleSelectAll(rowsLength, this.state.selection, tableRef);
        this.updateSelection(selection);
    };

    renderSelectTable = () => {
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
                        />
                        <div className="pl-4">{original.name}</div>
                    </div>
                ),
                className: `w-1/5 ${wrapClassName} ${defaultColumnClassName}`,
                headerClassName: `w-1/5 ${defaultHeaderClassName}`
            },
            {
                Header: 'Description',
                accessor: 'description',
                className: `w-1/3 ${wrapClassName} ${defaultColumnClassName}`,
                headerClassName: `w-1/3 ${defaultHeaderClassName}`
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
                Header: 'Actions',
                accessor: '',
                Cell: ({ original }) => (
                    <button
                        className="flex rounded-sm uppercase text-center text-sm items-center"
                        onClick={this.toggleEnabledDisabledPolicy(original)}
                    >
                        {original.disabled && <Icon.Power className="h-5 w-4 text-base-600" />}
                        {!original.disabled && <Icon.Power className="h-5 w-4 text-success-500" />}
                    </button>
                ),
                width: 75
            }
        ];
        const id = this.props.selectedPolicy && this.props.selectedPolicy.id;
        return (
            <div
                data-test-id="policies-table-container"
                className={`w-full
                    ${
                        this.props.wizardState.current !== ''
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
                    selection={this.state.selection}
                    selectedRowId={id}
                    noDataText="No results found. Please refine your search."
                    page={this.state.page}
                />
            </div>
        );
    };

    renderTablePanel = () => {
        const { length } = this.props.policies;
        if (!length)
            return <NoResultsMessage message="No results found. Please refine your search." />;
        const buttonsDisabled = this.props.wizardState.current !== '';
        const selectionCount = this.state.selection.length;
        const panelButtons = (
            <React.Fragment>
                {selectionCount !== 0 && (
                    <PanelButton
                        icon={<Icon.Trash2 className="h-4 w- ml-1" />}
                        text={`Delete (${selectionCount})`}
                        className="btn btn-danger"
                        onClick={this.showConfirmationDialog}
                        disabled={buttonsDisabled}
                    />
                )}
                {selectionCount === 0 && (
                    <PanelButton
                        icon={<Icon.RefreshCw className="h-4 w-4 ml-1" />}
                        text="Reassess All"
                        className="btn btn-base"
                        onClick={this.props.reassessPolicies}
                        tooltip="Manually enrich external data"
                        disabled={buttonsDisabled}
                    />
                )}
                {selectionCount === 0 && (
                    <PanelButton
                        icon={<Icon.Plus className="h-4 w-4 ml-1" />}
                        text="New Policy"
                        className="btn btn-base"
                        onClick={this.addPolicy}
                        disabled={buttonsDisabled}
                    />
                )}
            </React.Fragment>
        );
        const totalPages = length === pageSize ? 1 : Math.floor(length / pageSize) + 1;
        const paginationComponent = (
            <TablePagination
                page={this.state.page}
                totalPages={totalPages}
                setPage={this.setTablePage}
            />
        );
        return (
            <Panel
                header={this.getTableHeaderText()}
                buttons={panelButtons}
                headerComponents={paginationComponent}
            >
                {this.renderSelectTable()}
            </Panel>
        );
    };

    renderSidePanelButtons = () => {
        switch (this.props.wizardState.current) {
            case 'EDIT':
            case 'PRE_PREVIEW':
                return (
                    <PanelButton
                        icon={<Icon.ArrowRight className="h-4 w-4" />}
                        text="Next"
                        className="btn btn-primary"
                        onClick={this.getPolicyDryRun}
                    />
                );
            case 'PREVIEW':
                return (
                    <React.Fragment>
                        <PanelButton
                            icon={<Icon.ArrowLeft className="h-4 w-4" />}
                            text="Previous"
                            className="btn btn-primary"
                            onClick={this.onBackToEditFields}
                        />
                        <PanelButton
                            icon={<Icon.Save className="h-4 w-4" />}
                            text="Save"
                            className="btn btn-success"
                            onClick={this.onSubmit}
                        />
                    </React.Fragment>
                );
            default:
                return (
                    <PanelButton
                        icon={<Icon.Edit className="h-4 w-4" />}
                        text="Edit"
                        className="btn btn-success"
                        onClick={this.onPolicyEdit}
                        disabled={this.props.isFetchingPolicy}
                    />
                );
        }
    };

    renderSidePanelView = selectedPolicy => {
        if (this.props.isFetchingPolicy) return <Loader />;

        if (this.props.wizardState.current === '') return <PolicyDetails policy={selectedPolicy} />;
        return <PolicyCreationWizard />;
    };

    renderSidePanel = () => {
        const { selectedPolicy } = this.props;
        if (!this.props.wizardState.current && !selectedPolicy) return null;

        const editingPolicy = Object.assign({}, this.props.wizardState.policy, selectedPolicy);
        const header = editingPolicy ? editingPolicy.name : '';
        const buttons = this.renderSidePanelButtons();
        return (
            <Panel
                header={header}
                buttons={buttons}
                onClose={this.setSelectedPolicy}
                className="w-1/2 bg-primary-200"
            >
                <div className="bg-primary-200 w-full">
                    {this.renderSidePanelView(selectedPolicy)}
                </div>
            </Panel>
        );
    };

    renderConfirmationDialog = () => {
        const numSelectedRows = this.state.selection.length;
        return (
            <Dialog
                isOpen={this.state.showConfirmationDialog}
                text={`Are you sure you want to delete ${numSelectedRows} ${
                    numSelectedRows === 1 ? 'policy' : 'policies'
                }?`}
                onConfirm={this.deletePolicies}
                onCancel={this.hideConfirmationDialog}
            />
        );
    };

    render() {
        const subHeader = this.props.isViewFiltered ? 'Filtered view' : 'Default view';
        return (
            <section className="flex flex-1 flex-col h-full">
                <div>
                    <PageHeader header="Policies" subHeader={subHeader}>
                        <SearchInput
                            className="flex flex-1"
                            id="policies"
                            searchOptions={this.props.searchOptions}
                            searchModifiers={this.props.searchModifiers}
                            searchSuggestions={this.props.searchSuggestions}
                            setSearchOptions={this.props.setSearchOptions}
                            setSearchModifiers={this.props.setSearchModifiers}
                            setSearchSuggestions={this.props.setSearchSuggestions}
                            onSearch={this.onSearch}
                        />
                    </PageHeader>
                </div>
                <div className="flex flex-1 bg-base-200">
                    <div className="flex w-full h-full bg-base-100 rounded-sm shadow">
                        {this.renderTablePanel()}
                        {this.renderSidePanel()}
                    </div>
                </div>
                {this.renderConfirmationDialog()}
            </section>
        );
    }
}

const isViewFiltered = createSelector(
    [selectors.getPoliciesSearchOptions],
    searchOptions => searchOptions.length !== 0
);

const getFormData = state =>
    formValueSelector('policyCreationForm')(state, ...getPolicyFormDataKeys());

const getSelectedPolicy = (state, props) => {
    const { policyId } = props.match.params;
    return policyId ? selectors.getPolicy(state, policyId) : null;
};

const mapStateToProps = createStructuredSelector({
    policies: selectors.getFilteredPolicies,
    selectedPolicy: getSelectedPolicy,
    formData: getFormData,
    wizardState: selectors.getPolicyWizardState,
    searchOptions: selectors.getPoliciesSearchOptions,
    searchModifiers: selectors.getPoliciesSearchModifiers,
    searchSuggestions: selectors.getPoliciesSearchSuggestions,
    isViewFiltered,
    isFetchingPolicy: state => selectors.getLoadingStatus(state, types.FETCH_POLICY)
});

const mapDispatchToProps = {
    setSearchOptions: policyActions.setPoliciesSearchOptions,
    setSearchModifiers: policyActions.setPoliciesSearchModifiers,
    setSearchSuggestions: policyActions.setPoliciesSearchSuggestions,
    reassessPolicies: policyActions.reassessPolicies,
    deletePolicies: policyActions.deletePolicies,
    updatePolicyDisabledState: policyActions.updatePolicyDisabledState,
    setWizardState: policyActions.setPolicyWizardState,
    addToast: notificationActions.addNotification,
    removeToast: notificationActions.removeOldestNotification
};

export default connect(mapStateToProps, mapDispatchToProps)(PoliciesPage);
