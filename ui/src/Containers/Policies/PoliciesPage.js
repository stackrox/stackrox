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
import Table from 'Components/Table';
import Panel from 'Components/Panel';
import PanelButton from 'Components/PanelButton';
import { formatPolicyFields, getPolicyFormDataKeys } from 'Containers/Policies/policyFormUtils';
import PolicyDetails from 'Containers/Policies/PolicyDetails';
import PageHeader from 'Components/PageHeader';
import SearchInput from 'Components/SearchInput';

import { severityLabels } from 'messages/common';
import { sortSeverity } from 'sorters/sorters';
import PolicyCreationWizard from 'Containers/Policies/PolicyCreationWizard';

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
            showConfirmationDialog: false
        };
    }

    componentWillUnmount() {
        this.props.setWizardState({ current: '' });
    }

    onSearch = searchOptions => {
        if (searchOptions.length && !searchOptions[searchOptions.length - 1].type) {
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

    deletePolicies = () => {
        const policyIds = [];
        this.policyTable.getSelectedRows().forEach(rowId => {
            // close the view panel if that policy is being deleted
            if (rowId === this.props.match.params.policyId) {
                this.setSelectedPolicy();
            }
            policyIds.push(rowId);
        });
        this.policyTable.clearSelectedRows();
        this.hideConfirmationDialog();
        this.props.deletePolicies(policyIds);
    };

    addPolicy = () => {
        this.setSelectedPolicy();
        this.props.setWizardState({ current: 'EDIT', policy: null, isNew: true });
    };

    toggleEnabledDisabledPolicy = policy => {
        this.props.updatePolicyDisabledState({ policyId: policy.id, disabled: !policy.disabled });
    };

    showConfirmationDialog = () => {
        this.setState({ showConfirmationDialog: true });
    };

    hideConfirmationDialog = () => {
        this.setState({ showConfirmationDialog: false });
    };

    renderTablePanel = () => {
        const columns = [
            {
                key: 'name',
                keys: ['name', 'disabled'],
                keyValueFunc: (name, disabled) => (
                    <div className="flex items-center relative">
                        <div
                            className={`h-2 w-2 rounded-lg absolute -ml-4 ${
                                !disabled ? 'bg-success-500' : 'bg-base-300'
                            }`}
                        />
                        <div>{name}</div>
                    </div>
                ),
                label: 'Name'
            },
            { key: 'description', label: 'Description' },
            {
                key: 'severity',
                label: 'Severity',
                keyValueFunc: severity => severityLabels[severity],
                classFunc: severity => {
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
                },
                sortMethod: sortSeverity('severity')
            }
        ];
        const actions = [
            {
                renderIcon: row =>
                    row.disabled ? (
                        <Icon.Power className="h-5 w-4 text-base-600" />
                    ) : (
                        <Icon.Power className="h-5 w-4 text-success-500" />
                    ),
                className: 'flex rounded-sm uppercase text-center text-sm items-center',
                onClick: this.toggleEnabledDisabledPolicy
            }
        ];

        const buttonsDisabled = this.props.wizardState.current !== '';
        const panelButtons = (
            <React.Fragment>
                <PanelButton
                    icon={<Icon.Trash2 className="h-4 w-4" />}
                    text="Delete"
                    className="btn btn-danger"
                    onClick={this.showConfirmationDialog}
                    disabled={buttonsDisabled}
                />
                <PanelButton
                    icon={<Icon.FileText className="h-4 w-4" />}
                    text="Reassess Policies"
                    className="btn btn-success"
                    onClick={this.props.reassessPolicies}
                    tooltip="Manually enrich external data"
                    disabled={buttonsDisabled}
                />
                <PanelButton
                    icon={<Icon.Plus className="h-4 w-4" />}
                    text="Add"
                    className="btn btn-success"
                    onClick={this.addPolicy}
                    disabled={buttonsDisabled}
                />
            </React.Fragment>
        );
        return (
            <Panel header={`${this.props.policies.length} Policies`} buttons={panelButtons}>
                <div
                    data-test-id="policies-table-container"
                    className={`w-full
                        ${
                            this.props.wizardState.current !== ''
                                ? 'pointer-events-none opacity-25'
                                : ''
                        }`}
                >
                    <Table
                        columns={columns}
                        rows={this.props.policies}
                        onRowClick={this.setSelectedPolicy}
                        actions={actions}
                        checkboxes
                        ref={table => {
                            this.policyTable = table;
                        }}
                    />
                </div>
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
            <Panel header={header} buttons={buttons} onClose={this.setSelectedPolicy} width="w-2/3">
                {this.renderSidePanelView(selectedPolicy)}
            </Panel>
        );
    };

    renderConfirmationDialog = () => {
        const numSelectedRows = this.policyTable ? this.policyTable.getSelectedRows().length : 0;
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
                <div className="flex flex-1 bg-base-100">
                    <div className="flex flex-row w-full h-full bg-white rounded-sm shadow">
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
