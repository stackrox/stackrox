import React, { Component } from 'react';
import PropTypes from 'prop-types';
import ReactRouterPropTypes from 'react-router-prop-types';
import { connect } from 'react-redux';
import { selectors } from 'reducers';
import { actions as policiesActions } from 'reducers/policies';
import { createSelector, createStructuredSelector } from 'reselect';

import { ToastContainer, toast } from 'react-toastify';
import { Form } from 'react-form';
import * as Icon from 'react-feather';
import Dialog from 'Components/Dialog';
import Table from 'Components/Table';
import Panel from 'Components/Panel';
import {
    postFormatScopeField,
    postFormatWhitelistField,
    removeEmptyFields
} from 'Containers/Policies/policyFormUtils';
import {
    reassessPolicies,
    deletePolicy,
    savePolicy,
    createPolicy,
    getDryRun
} from 'services/PoliciesService';
import PolicyCreationForm from 'Containers/Policies/PolicyCreationForm';
import PolicyView from 'Containers/Policies/PoliciesView';
import PoliciesPreview from 'Containers/Policies/PoliciesPreview';
import PageHeader from 'Components/PageHeader';
import SearchInput from 'Components/SearchInput';

import { severityLabels } from 'messages/common';
import { sortSeverity } from 'sorters/sorters';

const reducer = (action, prevState, nextState) => {
    switch (action) {
        case 'UPDATE_POLICIES':
            return { policies: nextState.policies };
        case 'SELECT_POLICY':
            return {
                editingPolicy: null,
                addingPolicy: false,
                showPreviewPolicy: false
            };
        case 'UNSELECT_POLICY':
            return { editingPolicy: null, addingPolicy: false };
        case 'EDIT_POLICY':
            return { editingPolicy: nextState.policy };
        case 'ADD_POLICY':
            return { editingPolicy: nextState.policy, addingPolicy: true };
        default:
            return prevState;
    }
};

class PoliciesPage extends Component {
    static propTypes = {
        policies: PropTypes.arrayOf(PropTypes.object).isRequired,
        fetchPolicies: PropTypes.func.isRequired,
        history: ReactRouterPropTypes.history.isRequired,
        match: ReactRouterPropTypes.match.isRequired,
        searchOptions: PropTypes.arrayOf(PropTypes.object).isRequired,
        searchModifiers: PropTypes.arrayOf(PropTypes.object).isRequired,
        searchSuggestions: PropTypes.arrayOf(PropTypes.object).isRequired,
        setSearchOptions: PropTypes.func.isRequired,
        setSearchModifiers: PropTypes.func.isRequired,
        setSearchSuggestions: PropTypes.func.isRequired,
        isViewFiltered: PropTypes.bool.isRequired
    };

    constructor(props) {
        super(props);

        this.state = {
            editingPolicy: null,
            addingPolicy: false,
            policyDryRun: null,
            showPreviewPolicy: false,
            showConfirmationDialog: false
        };
    }

    onSubmit = policy => {
        if (this.state.addingPolicy) this.createPolicy(policy);
        else this.savePolicy(policy);
    };

    getPolicyDryRun = policy => {
        let filteredPolicy = removeEmptyFields(policy);
        filteredPolicy = postFormatWhitelistField(filteredPolicy);
        filteredPolicy = postFormatScopeField(filteredPolicy);
        this.update('EDIT_POLICY', { policy: filteredPolicy });
        const data = Object.assign({}, filteredPolicy);
        // set disabled to false for dryrun so that we can see what deployments the policy will affect
        data.disabled = false;
        getDryRun(data)
            .then(response => {
                if (!response.data) return;
                const policyDryRun = response.data;
                this.setState({ policyDryRun, showPreviewPolicy: true });
            })
            .catch(error => {
                console.error(error);
                if (error.response) toast(error.response.data.error);
            });
    };

    getSelectedPolicy = () => {
        if (this.props.match.params.id && this.props.policies.length !== 0) {
            return this.props.policies.find(policy => policy.id === this.props.match.params.id);
        }
        return null;
    };

    updateSelectedPolicy = policy => {
        const urlSuffix = policy && policy.id ? `/${policy.id}` : '';
        this.props.history.push({
            pathname: `/main/policies${urlSuffix}`
        });
    };

    preSubmit = policy => {
        let newPolicy = removeEmptyFields(policy);
        newPolicy = postFormatScopeField(newPolicy);
        newPolicy = postFormatWhitelistField(newPolicy);
        return newPolicy;
    };

    reassessPolicies = () => {
        reassessPolicies()
            .then(() => {
                toast('Policies were reassessed');
                this.policyTable.clearSelectedRows();
            })
            .catch(error => {
                console.error(error);
                if (error.response) toast(error.response.data.error);
            });
    };

    deletePolicies = () => {
        const promises = [];
        this.policyTable.getSelectedRows().forEach(row => {
            // close the view panel if that policy is being deleted
            if (row.id === this.props.match.params.id) {
                this.unselectPolicy();
            }
            const promise = deletePolicy(row.id);
            promises.push(promise);
        });
        Promise.all(promises).then(() => {
            this.policyTable.clearSelectedRows();
            this.hideConfirmationDialog();
            this.props.fetchPolicies();
        });
    };

    addPolicy = () => {
        this.update('ADD_POLICY', { policy: {} });
    };

    selectPolicy = policy => {
        this.update('SELECT_POLICY');
        this.updateSelectedPolicy(policy);
    };

    unselectPolicy = () => {
        this.update('UNSELECT_POLICY');
        this.updateSelectedPolicy();
    };

    editPolicy = policy => {
        this.update('EDIT_POLICY', { policy });
        this.updateSelectedPolicy(policy);
    };

    createPolicy = policy => {
        createPolicy(policy)
            .then(response => {
                const createdPolicy = response.data;
                this.selectPolicy(createdPolicy);
            })
            .catch(error => {
                console.error(error);
                if (error.response) toast(error.response.data.error);
            });
    };

    updatePolicy = policy =>
        savePolicy(policy)
            .then(() => {
                this.props.fetchPolicies();
            })
            .catch(error => {
                console.error(error);
                if (error.response) toast(error.response.data.error);
                return error;
            });

    savePolicy = policy => {
        this.updatePolicy(policy).then(error => {
            if (error !== undefined) return;
            this.selectPolicy(policy);
        });
    };

    toggleEnabledDisabledPolicy = policy => {
        const newPolicy = Object.assign({}, policy);
        newPolicy.disabled = !policy.disabled;
        this.updatePolicy(newPolicy);
    };

    update = (action, nextState) => {
        this.setState(prevState => reducer(action, prevState, nextState));
    };

    closePreviewPanel = () => {
        this.setState({ showPreviewPolicy: false });
        return this.unselectPolicy();
    };

    closeEditPanel = () => {
        const newPolicy = this.state.addingPolicy;
        if (newPolicy) return this.unselectPolicy();
        return this.unselectPolicy();
    };

    showConfirmationDialog = () => {
        this.setState({ showConfirmationDialog: true });
    };

    hideConfirmationDialog = () => {
        this.setState({ showConfirmationDialog: false });
    };

    renderTablePanel = () => {
        const header = `${this.props.policies.length} Policies`;
        const buttons = [
            {
                renderIcon: () => <Icon.Trash2 className="h-4 w-4" />,
                text: 'Delete Policies',
                className: 'btn-danger',
                onClick: this.showConfirmationDialog,
                disabled: this.state.editingPolicy !== null
            },
            {
                renderIcon: () => <Icon.FileText className="h-4 w-4" />,
                text: 'Reassess Policies',
                className: 'btn-success',
                onClick: this.reassessPolicies,
                disabled: this.state.editingPolicy !== null,
                tooltip: 'Manually enrich external data'
            },
            {
                renderIcon: () => <Icon.Plus className="h-4 w-4" />,
                text: 'Add',
                className: 'btn-success',
                onClick: this.addPolicy,
                disabled: this.state.editingPolicy !== null
            }
        ];
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
                sortMethod: sortSeverity
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
        const rows = this.props.policies;
        return (
            <Panel header={header} buttons={buttons}>
                <Table
                    columns={columns}
                    rows={rows}
                    onRowClick={this.selectPolicy}
                    actions={actions}
                    checkboxes
                    ref={table => {
                        this.policyTable = table;
                    }}
                />
            </Panel>
        );
    };

    renderViewPanel = () => {
        const policy = this.getSelectedPolicy();
        const hide = policy === null || policy === undefined || this.state.editingPolicy !== null;
        if (hide) return '';
        const header = policy.name;
        const buttons = [
            {
                renderIcon: () => <Icon.Edit className="h-4 w-4" />,
                text: 'Edit',
                className: 'btn-success',
                onClick: () => {
                    this.editPolicy(policy);
                }
            }
        ];
        return (
            <Panel header={header} buttons={buttons} onClose={this.unselectPolicy} width="w-2/3">
                <PolicyView />
            </Panel>
        );
    };

    renderEditPanel = () => {
        const policy = this.state.editingPolicy;
        const hide = policy === null;
        if (hide || this.state.showPreviewPolicy) return '';
        const header = this.state.editingPolicy.name;
        const buttons = [
            {
                renderIcon: () => <Icon.ArrowRight className="h-4 w-4" />,
                text: 'Next',
                className: 'btn-primary',
                onClick: () => {
                    this.getPolicyDryRun(this.formApi.values);
                }
            }
        ];
        return (
            <Panel header={header} buttons={buttons} onClose={this.closeEditPanel} width="w-2/3">
                <Form onSubmit={this.onSubmit} preSubmit={this.preSubmit}>
                    {formApi => (
                        <PolicyCreationForm
                            policy={policy}
                            formApi={formApi}
                            ref={() => {
                                this.formApi = formApi;
                            }}
                        />
                    )}
                </Form>
            </Panel>
        );
    };

    renderPreviewPanel = () => {
        if (!this.state.showPreviewPolicy) return '';
        const policy = this.state.editingPolicy;
        const hide = policy === null;
        if (hide) return '';
        const header = this.state.editingPolicy.name;
        const buttons = [
            {
                renderIcon: () => <Icon.ArrowLeft className="h-4 w-4" />,
                text: 'Previous',
                className: 'btn-primary',
                onClick: () => {
                    this.setState({ showPreviewPolicy: false });
                }
            },
            {
                renderIcon: () => <Icon.Save className="h-4 w-4" />,
                text: 'Save',
                className: 'btn-success',
                onClick: () => {
                    this.setState({ showPreviewPolicy: false });
                    this.onSubmit(this.state.editingPolicy);
                }
            }
        ];
        return (
            <Panel header={header} buttons={buttons} onClose={this.closePreviewPanel} width="w-2/3">
                <PoliciesPreview
                    dryrun={this.state.policyDryRun}
                    policyDisabled={this.state.editingPolicy.disabled || false}
                />
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
                <ToastContainer
                    toastClassName="font-sans text-base-600 text-white font-600 bg-black"
                    hideProgressBar
                    autoClose={3000}
                />
                <div>
                    <PageHeader header="Policies" subHeader={subHeader}>
                        <SearchInput
                            id="risk"
                            searchOptions={this.props.searchOptions}
                            searchModifiers={this.props.searchModifiers}
                            searchSuggestions={this.props.searchSuggestions}
                            setSearchOptions={this.props.setSearchOptions}
                            setSearchModifiers={this.props.setSearchModifiers}
                            setSearchSuggestions={this.props.setSearchSuggestions}
                        />
                    </PageHeader>
                </div>
                <div className="flex flex-1 bg-base-100">
                    <div className="flex flex-row w-full h-full bg-white rounded-sm shadow">
                        {this.renderTablePanel()}
                        {this.renderViewPanel()}
                        {this.renderEditPanel()}
                        {this.renderPreviewPanel()}
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

const mapStateToProps = createStructuredSelector({
    policies: selectors.getPolicies,
    searchOptions: selectors.getPoliciesSearchOptions,
    searchModifiers: selectors.getPoliciesSearchModifiers,
    searchSuggestions: selectors.getPoliciesSearchSuggestions,
    isViewFiltered
});

const mapDispatchToProps = dispatch => ({
    fetchPolicies: () => dispatch(policiesActions.fetchPolicies.request()),
    setSearchOptions: searchOptions =>
        dispatch(policiesActions.setPoliciesSearchOptions(searchOptions)),
    setSearchModifiers: searchModifiers =>
        dispatch(policiesActions.setPoliciesSearchModifiers(searchModifiers)),
    setSearchSuggestions: searchSuggestions =>
        dispatch(policiesActions.setPoliciesSearchSuggestions(searchSuggestions))
});

export default connect(mapStateToProps, mapDispatchToProps)(PoliciesPage);
