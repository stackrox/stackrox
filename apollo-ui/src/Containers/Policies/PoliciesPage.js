import React, { Component } from 'react';
import Table from 'Components/Table';
import Panel from 'Components/Panel';
import { ToastContainer, toast } from 'react-toastify';
import { Form } from 'react-form';
import PolicyCreationForm from 'Containers/Policies/PolicyCreationForm';
import PolicyView from 'Containers/Policies/PoliciesView';

import axios from 'axios';
import * as Icon from 'react-feather';
import isEqual from 'lodash/isEqual';

const reducer = (action, prevState, nextState) => {
    switch (action) {
        case 'UPDATE_POLICIES':
            return { policies: nextState.policies };
        case 'SELECT_POLICY':
            return { selectedPolicy: nextState.policy, editingPolicy: null, addingPolicy: false };
        case 'UNSELECT_POLICY':
            return { selectedPolicy: null, editingPolicy: null, addingPolicy: false };
        case 'EDIT_POLICY':
            return { editingPolicy: nextState.policy, addingPolicy: false };
        case 'ADD_POLICY':
            return { editingPolicy: nextState.policy, addingPolicy: true };
        case 'CANCEL_EDIT_POLICY':
            return { editingPolicy: null, addingPolicy: false };
        case 'CANCEL_ADD_POLICY':
            return { selectedPolicy: null, editingPolicy: null, addingPolicy: false };
        default:
            return prevState;
    }
};

class PoliciesPage extends Component {
    constructor(props) {
        super(props);

        this.state = {
            policies: [],
            selectedPolicy: null,
            editingPolicy: null,
            addingPolicy: false
        };
    }

    componentDidMount() {
        this.pollImagesPolicies();
    }

    componentWillUnmount() {
        if (this.pollTimeoutId) {
            clearTimeout(this.pollTimeoutId);
            this.pollTimeoutId = null;
        }
    }

    onSubmit = (policy) => {
        console.log('onSubmit', policy);
        if (this.state.addingPolicy) this.createPolicy(policy);
        else this.savePolicy(policy);
    }

    getImagesPolicies = () => axios.get('/v1/policies', { params: this.params }).then((response) => {
        if (!response.data.policies ||
            isEqual(this.state.policies, response.data.policies)) return;
        const { policies } = response.data;
        this.update('UPDATE_POLICIES', { policies });
    });

    pollImagesPolicies = () => {
        this.getImagesPolicies().then(() => {
            this.pollTimeoutId = setTimeout(this.pollImagesPolicies, 5000);
        });
    }

    preSubmit = policy => this.policyCreationForm.preSubmit(policy);

    addPolicy = () => {
        this.update('ADD_POLICY', { policy: {} });
    }

    selectPolicy = (policy) => {
        this.update('SELECT_POLICY', { policy });
    }

    unselectPolicy = () => {
        this.update('UNSELECT_POLICY');
    }

    editPolicy = (policy) => {
        this.update('EDIT_POLICY', { policy });
    }

    cancelAddingPolicy = () => {
        this.update('CANCEL_ADD_POLICY');
    }

    cancelEditingPolicy = () => {
        this.update('CANCEL_EDIT_POLICY');
    }

    createPolicy = (policy) => {
        axios.post('/v1/policies', policy).then(() => {
            this.cancelAddingPolicy();
            this.getImagesPolicies();
            this.selectPolicy(policy);
        }).catch((error) => {
            console.error(error);
            if (error.response) toast(error.response.data.error);
        });
    }

    savePolicy = (policy) => {
        console.log(policy);
        axios.put(`/v1/policies/${policy.id}`, policy).then(() => {
            this.cancelEditingPolicy();
            this.getImagesPolicies();
            this.selectPolicy(policy);
        }).catch((error) => {
            console.error(error);
            if (error.response) toast(error.response.data.error);
        });
    }

    update = (action, nextState) => {
        this.setState(prevState => reducer(action, prevState, nextState));
    }

    renderTablePanel = () => {
        const header = `${this.state.policies.length} Policies`;
        const buttons = [
            {
                renderIcon: () => <Icon.Plus className="h-4 w-4" />,
                text: 'Add Policy',
                className: 'flex py-2 px-2 rounded-sm text-success-600 hover:text-white hover:bg-success-400 uppercase text-center text-sm items-center ml-2 bg-white border-2 border-success-400',
                onClick: () => this.addPolicy(),
                disabled: this.state.editingPolicy !== null
            }
        ];
        const columns = [
            { key: 'name', label: 'Name' },
            { key: 'description', label: 'Description' },
            {
                key: 'severity',
                keyValueFunc: item => item.split('_')[0].capitalizeFirstLetterOfWord(),
                label: 'Severity',
                classFunc: (item) => {
                    switch (item) {
                        case 'Low':
                            return 'text-low-500';
                        case 'Medium':
                            return 'text-medium-500';
                        case 'High':
                            return 'text-high-500';
                        case 'Critical':
                            return 'text-critical-500';
                        default:
                            return '';
                    }
                }
            },
        ];
        const rows = this.state.policies;
        return (
            <Panel header={header} buttons={buttons}>
                <Table columns={columns} rows={rows} onRowClick={this.selectPolicy} />
            </Panel>
        );
    }

    renderViewPanel = () => {
        const policy = this.state.selectedPolicy;
        const hide = this.state.selectedPolicy === null || this.state.editingPolicy !== null;
        if (hide) return '';
        const header = this.state.selectedPolicy.name;
        const buttons = [
            {
                renderIcon: () => <Icon.X className="h-4 w-4" />,
                className: 'flex py-2 px-2 rounded-sm text-primary-600 hover:text-white hover:bg-primary-400 uppercase text-center text-sm items-center ml-2 bg-white border-2 border-primary-400',
                onClick: this.unselectPolicy
            },
            {
                renderIcon: () => <Icon.Edit className="h-4 w-4" />,
                text: 'Edit Policy',
                className: 'flex py-2 px-2 rounded-sm text-success-600 hover:text-white hover:bg-success-400 uppercase text-center text-sm items-center ml-2 bg-white border-2 border-success-400',
                onClick: () => {
                    const { selectedPolicy } = this.state;
                    this.editPolicy(selectedPolicy);
                }
            }
        ];
        return (
            <Panel header={header} buttons={buttons} width="w-2/3">
                <PolicyView policy={policy} />
            </Panel>
        );
    }

    renderEditPanel = () => {
        const policy = this.state.editingPolicy;
        const hide = policy === null;
        if (hide) return '';
        const header = this.state.editingPolicy.name;
        const buttons = [
            {
                renderIcon: () => <Icon.X className="h-4 w-4" />,
                text: 'Cancel',
                className: 'flex py-2 px-2 rounded-sm text-primary-600 hover:text-white hover:bg-primary-400 uppercase text-center text-sm items-center ml-2 bg-white border-2 border-primary-400',
                onClick: () => {
                    const newPolicy = this.state.addingPolicy;
                    if (newPolicy) return this.cancelAddingPolicy();
                    return this.cancelEditingPolicy();
                }
            },
            {
                renderIcon: () => <Icon.Save className="h-4 w-4" />,
                text: 'Save Policy',
                className: 'flex py-2 px-2 rounded-sm text-success-600 hover:text-white hover:bg-success-400 uppercase text-center text-sm items-center ml-2 bg-white border-2 border-success-400',
                onClick: () => this.policyCreationForm.submitForm()
            }
        ];
        return (
            <Panel header={header} buttons={buttons} width="w-2/3">
                <Form onSubmit={this.onSubmit} preSubmit={this.preSubmit}>
                    {formApi => (
                        <PolicyCreationForm policy={policy} formApi={formApi} ref={(policyCreationForm) => { this.policyCreationForm = policyCreationForm; }} />
                    )}
                </Form>
            </Panel>
        );
    }

    render() {
        return (
            <section className="flex flex-1 h-full">
                <ToastContainer toastClassName="font-sans text-base-600 text-white font-600 bg-black" hideProgressBar autoClose={3000} />
                <div className="flex flex-1 border-t border-primary-300 bg-base-100">
                    <div className="flex flex-row w-full p-3 overflow-y-scroll bg-white rounded-sm shadow">
                        {this.renderTablePanel()}
                        {this.renderViewPanel()}
                        {this.renderEditPanel()}
                    </div>
                </div>
            </section>
        );
    }
}

export default PoliciesPage;
