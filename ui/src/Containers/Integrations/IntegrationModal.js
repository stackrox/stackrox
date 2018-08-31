import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { withRouter } from 'react-router-dom';
import * as Icon from 'react-feather';

import Dialog from 'Components/Dialog';
import Form from 'Containers/Integrations/Form';
import Modal from 'Components/Modal';
import Table from 'Containers/Integrations/Table';

import * as AuthService from 'services/AuthService';
import { deleteIntegration } from 'services/IntegrationsService';

const SOURCE_LABELS = Object.freeze({
    authProviders: 'authentication provider',
    dnrIntegrations: 'integrations', // type is set to D&R, so this will read as "D&R Integrations"
    imageIntegrations: 'image integrations',
    notifiers: 'plugin'
});

class IntegrationModal extends Component {
    static propTypes = {
        integrations: PropTypes.arrayOf(
            PropTypes.shape({
                type: PropTypes.string
            })
        ).isRequired,
        source: PropTypes.oneOf([
            'imageIntegrations',
            'dnrIntegrations',
            'notifiers',
            'authProviders'
        ]).isRequired,
        type: PropTypes.string.isRequired,
        onRequestClose: PropTypes.func.isRequired,
        onIntegrationsUpdate: PropTypes.func.isRequired,

        clusters: PropTypes.arrayOf(
            PropTypes.shape({
                name: PropTypes.string.isRequired,
                id: PropTypes.string.isRequired
            })
        )
    };

    static defaultProps = {
        clusters: []
    };

    constructor(props) {
        super(props);

        this.state = {
            formIsOpen: false,
            formValues: {},
            errorMessage: '',
            successMessage: '',
            showConfirmationDialog: false,
            selectedIntegrationId: null
        };
    }

    onRequestClose = isSuccessful => {
        this.clearMessages();
        this.props.onRequestClose(isSuccessful);
    };

    onTableDelete = () => {
        this.showConfirmationDialog();
    };

    onTableAdd = () => {
        this.setState({
            formIsOpen: true
        });
    };

    onTableRowClick = integration => {
        this.setState({
            formIsOpen: true,
            formValues: integration,
            selectedIntegrationId: integration.id
        });
    };

    onFormCancel = () => {
        this.closeIntegrationForm();
    };

    onFormRequest = () => {
        this.clearMessages();
    };

    onFormError = errorMessage => {
        this.setState({ errorMessage });
    };

    onFormSubmitSuccess = () => {
        this.props.onIntegrationsUpdate(this.props.source);
        this.closeIntegrationForm();
    };

    onFormTestSuccess = () => {
        this.setState({
            successMessage: 'Integration test was successful'
        });
    };

    setTableRef = table => {
        this.integrationTable = table;
    };

    hideConfirmationDialog = () => {
        this.setState({ showConfirmationDialog: false });
    };

    showConfirmationDialog = () => {
        this.setState({ showConfirmationDialog: true });
    };

    closeIntegrationForm = () => {
        this.setState({
            formIsOpen: false,
            formValues: {},
            selectedIntegrationId: null
        });
    };

    clearMessages = () => {
        this.setState({
            errorMessage: '',
            successMessage: ''
        });
    };

    activateAuthIntegration = integration => () => {
        if (integration !== null && integration.loginUrl !== null && !integration.validated) {
            window.location = integration.loginUrl;
        }
    };

    deleteTableSelectedIntegrations = () => {
        const promises = [];
        if (!this.integrationTable) return;
        const numSelectedRows = this.integrationTable.state.selection;
        numSelectedRows.forEach(id => {
            const promise =
                this.props.source === 'authProviders'
                    ? AuthService.deleteAuthProvider(id)
                    : deleteIntegration(this.props.source, id);
            promises.push(promise);
        });
        Promise.all(promises).then(() => {
            this.integrationTable.clearSelectedRows();
            this.hideConfirmationDialog();
            this.props.onIntegrationsUpdate(this.props.source);
        });
    };

    renderHeader = () => {
        const { source, type } = this.props;
        return (
            <header className="flex items-center w-full p-4 bg-primary-500 text-white uppercase">
                <span className="flex flex-1">{`Configure ${type} ${SOURCE_LABELS[source]}`}</span>
                <Icon.X className="h-4 w-4 cursor-pointer" onClick={this.onRequestClose} />
            </header>
        );
    };

    renderErrorMessage = () => {
        if (this.state.errorMessage !== '') {
            return (
                <div className="px-4 py-2 bg-high-500 text-white" data-test-id="integration-error">
                    {this.state.errorMessage}
                </div>
            );
        }
        return null;
    };

    renderSuccessMessage = () => {
        if (this.state.successMessage !== '') {
            return (
                <div className="px-4 py-2 bg-success-500 text-white">
                    {this.state.successMessage}
                </div>
            );
        }
        return null;
    };

    renderTable = () => (
        <Table
            clusters={this.props.clusters}
            integrations={this.props.integrations}
            source={this.props.source}
            type={this.props.type}
            buttonsEnabled={!this.state.formIsOpen}
            onRowClick={this.onTableRowClick}
            onActivate={this.activateAuthIntegration}
            onAdd={this.onTableAdd}
            onDelete={this.onTableDelete}
            setTable={this.setTableRef}
            selectedIntegrationId={this.state.selectedIntegrationId}
        />
    );

    renderForm = () => {
        if (!this.state.formIsOpen) {
            return null;
        }
        return (
            <Form
                initialValues={this.state.formValues}
                source={this.props.source}
                type={this.props.type}
                clusters={this.props.clusters}
                onCancel={this.onFormCancel}
                onSubmitRequest={this.onFormRequest}
                onSubmitSuccess={this.onFormSubmitSuccess}
                onSubmitError={this.onFormError}
                onTestRequest={this.onFormRequest}
                onTestSuccess={this.onFormTestSuccess}
                onTestError={this.onFormError}
            />
        );
    };

    renderConfirmationDialog = () => {
        const numSelectedRows = this.integrationTable
            ? this.integrationTable.state.selection.length
            : 0;
        return (
            <Dialog
                isOpen={this.state.showConfirmationDialog}
                text={`Are you sure you want to delete ${numSelectedRows} integration(s)?`}
                onConfirm={this.deleteTableSelectedIntegrations}
                onCancel={this.hideConfirmationDialog}
            />
        );
    };

    render() {
        return (
            <Modal isOpen onRequestClose={this.onRequestClose} className="w-full lg:w-5/6 h-full">
                {this.renderHeader()}
                {this.renderErrorMessage()}
                {this.renderSuccessMessage()}
                <div className="flex flex-1 w-full bg-white">
                    {this.renderTable()}
                    {this.renderForm()}
                </div>
                {this.renderConfirmationDialog()}
            </Modal>
        );
    }
}

export default withRouter(IntegrationModal);
