import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { withRouter } from 'react-router-dom';
import { connect } from 'react-redux';
import { actions } from 'reducers/integrations';
import { createStructuredSelector } from 'reselect';
import * as Icon from 'react-feather';
import { selectors } from 'reducers';

import Dialog from 'Components/Dialog';
import Form from 'Containers/Integrations/Form';
import Modal from 'Components/Modal';
import Table from 'Containers/Integrations/Table';

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
        deleteIntegrations: PropTypes.func.isRequired,
        isCreating: PropTypes.bool,
        setCreateState: PropTypes.func.isRequired
    };

    static defaultProps = {
        isCreating: false
    };

    constructor(props) {
        super(props);

        this.state = {
            selectedIntegration: null,
            showConfirmationDialog: false
        };
    }

    componentWillUnmount() {
        this.props.setCreateState(false);
    }

    onTableDelete = () => {
        this.showConfirmationDialog();
    };

    onTableAdd = () => {
        this.props.setCreateState(true);
    };

    onTableRowClick = integration => {
        this.setState({
            selectedIntegration: integration
        });
        this.props.setCreateState(false);
    };

    setTableRef = table => {
        this.integrationTable = table;
    };

    getSelectedIntegrationId = () =>
        this.state.selectedIntegration ? this.state.selectedIntegration.id : '';

    // determines whether the form panel is open based on selected integration and creation state
    formIsOpen = () => this.props.isCreating || !!this.state.selectedIntegration;

    hideConfirmationDialog = () => {
        this.setState({ showConfirmationDialog: false });
    };

    showConfirmationDialog = () => {
        this.setState({ showConfirmationDialog: true });
    };

    closeIntegrationForm = () => {
        this.setState({
            selectedIntegration: null
        });
        this.props.setCreateState(false);
    };

    activateAuthIntegration = integration => () => {
        if (integration !== null && integration.loginUrl !== null && !integration.validated) {
            window.location = integration.loginUrl;
        }
    };

    deleteTableSelectedIntegrations = () => {
        if (!this.integrationTable) return;
        const { selection } = this.integrationTable.state;
        const { source, type } = this.props;
        this.props.deleteIntegrations(source, type, selection);
        this.integrationTable.clearSelectedRows();
        this.hideConfirmationDialog();
    };

    renderHeader = () => {
        const { source, type } = this.props;
        return (
            <header className="flex items-center w-full p-4 bg-primary-500 text-white uppercase">
                <span className="flex flex-1">{`Configure ${type} ${SOURCE_LABELS[source]}`}</span>
                <Icon.X className="h-4 w-4 cursor-pointer" onClick={this.props.onRequestClose} />
            </header>
        );
    };

    renderTable = () => (
        <Table
            integrations={this.props.integrations}
            source={this.props.source}
            type={this.props.type}
            buttonsEnabled={!this.formIsOpen()}
            onRowClick={this.onTableRowClick}
            onActivate={this.activateAuthIntegration}
            onAdd={this.onTableAdd}
            onDelete={this.onTableDelete}
            setTable={this.setTableRef}
            selectedIntegrationId={this.getSelectedIntegrationId()}
        />
    );

    renderForm = () => {
        const { source, type } = this.props;
        if (!this.formIsOpen()) return null;
        return (
            <Form
                initialValues={this.state.selectedIntegration}
                enableReinitialize
                source={source}
                type={type}
                onClose={this.closeIntegrationForm}
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
            <Modal
                isOpen
                onRequestClose={this.props.onRequestClose}
                className="w-full lg:w-5/6 h-full"
            >
                {this.renderHeader()}
                <div className="flex flex-1 w-full bg-white">
                    {this.renderTable()}
                    {this.renderForm()}
                </div>
                {this.renderConfirmationDialog()}
            </Modal>
        );
    }
}

const mapStateToProps = createStructuredSelector({
    isCreating: selectors.getCreationState
});

const mapDispatchToProps = dispatch => ({
    deleteIntegrations: (source, sourceType, ids) =>
        dispatch(actions.deleteIntegrations(source, sourceType, ids)),
    setCreateState: state => dispatch(actions.setCreateState(state))
});

export default withRouter(connect(mapStateToProps, mapDispatchToProps)(IntegrationModal));
