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
    imageIntegrations: 'image integrations',
    notifiers: 'plugin'
});

const reducer = (action, prevState, nextState) => {
    switch (action) {
        case 'EDIT_INTEGRATION':
            return { editIntegration: nextState.editIntegration };
        case 'CLEAR_MESSAGES':
            return { errorMessage: '', successMessage: '' };
        case 'ERROR_MESSAGE':
            return { errorMessage: nextState.errorMessage };
        case 'SUCCESS_MESSAGE':
            return { successMessage: nextState.successMessage };
        default:
            return prevState;
    }
};

class IntegrationModal extends Component {
    static propTypes = {
        integrations: PropTypes.arrayOf(
            PropTypes.shape({
                type: PropTypes.string.isRequired
            })
        ).isRequired,
        source: PropTypes.oneOf(['imageIntegrations', 'notifiers', 'authProviders']).isRequired,
        type: PropTypes.string.isRequired,
        onRequestClose: PropTypes.func.isRequired,
        onIntegrationsUpdate: PropTypes.func.isRequired
    };

    constructor(props) {
        super(props);

        this.state = {
            editIntegration: null,
            errorMessage: '',
            successMessage: '',
            showConfirmationDialog: false
        };
    }

    onRequestClose = isSuccessful => {
        this.update('CLEAR_MESSAGES');
        this.props.onRequestClose(isSuccessful);
    };

    onTableDelete = () => {
        this.showConfirmationDialog();
    };

    onTableRowClick = integration => {
        this.update('EDIT_INTEGRATION', { editIntegration: integration });
    };

    onTableActivate = () => {
        this.update('EDIT_INTEGRATION', { editIntegration: {} });
    };

    onTableAdd = () => {
        this.update('EDIT_INTEGRATION', { editIntegration: {} });
    };

    onFormCancel = () => {
        this.update('EDIT_INTEGRATION', { editIntegration: null });
    };

    onFormRequest = () => {
        this.update('CLEAR_MESSAGES');
    };

    onFormError = errorMessage => {
        this.update('ERROR_MESSAGE', { errorMessage });
    };

    onFormSubmitSuccess = () => {
        this.props.onIntegrationsUpdate(this.props.source);
        this.update('EDIT_INTEGRATION', { editIntegration: null });
    };

    onFormTestSuccess = () => {
        this.update('SUCCESS_MESSAGE', {
            successMessage: 'Integration test was successful'
        });
    };

    setTableRef = table => {
        this.integrationTable = table;
    };

    update = (action, nextState) => {
        this.setState(prevState => reducer(action, prevState, nextState));
    };

    hideConfirmationDialog = () => {
        this.setState({ showConfirmationDialog: false });
    };

    showConfirmationDialog = () => {
        this.setState({ showConfirmationDialog: true });
    };

    activateAuthIntegration = integration => {
        if (integration !== null && integration.loginUrl !== null && !integration.validated) {
            window.location = integration.loginUrl;
        }
    };

    deleteTableSelectedIntegrations = () => {
        const promises = [];
        this.integrationTable.getSelectedRows().forEach(id => {
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
            integrations={this.props.integrations}
            source={this.props.source}
            type={this.props.type}
            buttonsEnabled={this.state.editIntegration === null}
            onRowClick={this.onTableRowClick}
            onActivate={this.activateAuthIntegration}
            onAdd={this.onTableAdd}
            onDelete={this.onTableDelete}
            setTable={this.setTableRef}
        />
    );

    renderForm = () => (
        <Form
            initialValues={this.state.editIntegration}
            source={this.props.source}
            type={this.props.type}
            onCancel={this.onFormCancel}
            onSubmitRequest={this.onFormRequest}
            onSubmitSuccess={this.onFormSubmitSuccess}
            onSubmitError={this.onFormError}
            onTestRequest={this.onFormRequest}
            onTestSuccess={this.onFormTestSuccess}
            onTestError={this.onFormError}
        />
    );

    renderConfirmationDialog = () => {
        const numSelectedRows = this.integrationTable
            ? this.integrationTable.getSelectedRows().length
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
