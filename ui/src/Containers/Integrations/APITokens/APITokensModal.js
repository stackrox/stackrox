import React, { Component } from 'react';
import { connect } from 'react-redux';
import PropTypes from 'prop-types';
import * as Icon from 'react-feather';
import Tooltip from 'rc-tooltip';
import { createStructuredSelector } from 'reselect';

import { actions } from 'reducers/apitokens';
import { selectors } from 'reducers';

import CheckboxTable from 'Components/CheckboxTable';
import { rtTrActionsClassName } from 'Components/Table';
import { toggleRow, toggleSelectAll } from 'utils/checkboxUtils';
import Modal from 'Components/Modal';
import Dialog from 'Components/Dialog';
import Panel from 'Components/Panel';
import PanelButton from 'Components/PanelButton';
import NoResultsMessage from 'Components/NoResultsMessage';

import APITokenForm from './APITokenForm';
import APITokenDetails from './APITokenDetails';

class APITokensModal extends Component {
    static propTypes = {
        tokens: PropTypes.arrayOf(
            PropTypes.shape({
                id: PropTypes.string.isRequired,
                name: PropTypes.string.isRequired,
                role: PropTypes.string.isRequired
            })
        ).isRequired,
        tokenGenerationWizardOpen: PropTypes.bool.isRequired,
        onRequestClose: PropTypes.func.isRequired,
        startTokenGenerationWizard: PropTypes.func.isRequired,
        closeTokenGenerationWizard: PropTypes.func.isRequired,
        generateAPIToken: PropTypes.func.isRequired,
        revokeAPITokens: PropTypes.func.isRequired,
        currentGeneratedToken: PropTypes.string,
        currentGeneratedTokenMetadata: PropTypes.shape({
            name: PropTypes.string.isRequired,
            role: PropTypes.string.isRequired
        })
    };

    static defaultProps = {
        currentGeneratedToken: '',
        currentGeneratedTokenMetadata: null
    };

    state = {
        selectedTokenId: null,
        showConfirmationDialog: false,
        selection: []
    };

    onRowClick = row => {
        this.setState({ selectedTokenId: row.id });
    };

    onRevokeHandler = token => e => {
        e.stopPropagation();
        this.revokeTokens(token);
    };

    onSubmit = () => {
        this.props.generateAPIToken();
    };

    revokeTokens = ({ id }) => {
        if (id) this.props.revokeAPITokens([id]);
        else {
            this.props.revokeAPITokens(this.state.selection);
            this.hideConfirmationDialog();
            this.clearSelection();
        }
    };

    unSelectRow = () => {
        this.setState({ selectedTokenId: null });
    };

    closeModal = () => {
        this.props.closeTokenGenerationWizard();
        this.props.onRequestClose();
    };

    openForm = () => {
        this.props.startTokenGenerationWizard();
    };

    closeForm = () => {
        this.props.closeTokenGenerationWizard();
    };

    clearSelection = () => this.setState({ selection: [] });

    showConfirmationDialog = () => {
        this.setState({ showConfirmationDialog: true });
    };

    hideConfirmationDialog = () => {
        this.setState({ showConfirmationDialog: false });
    };

    showModalView = () => {
        if (!this.props.tokens || !this.props.tokens.length)
            return <NoResultsMessage message="No API Tokens Generated" />;

        const columns = [
            { accessor: 'name', Header: 'Name' },
            { accessor: 'role', Header: 'Role' },
            {
                Header: '',
                accessor: '',
                headerClassName: 'hidden',
                className: rtTrActionsClassName,
                Cell: ({ original }) => this.renderRowActionButtons(original)
            }
        ];

        return (
            <CheckboxTable
                ref={table => {
                    this.apiTokenModalTable = table;
                }}
                rows={this.props.tokens}
                columns={columns}
                onRowClick={this.onRowClick}
                toggleRow={this.toggleRow}
                toggleSelectAll={this.toggleSelectAll}
                selection={this.state.selection}
                selectedRowId={this.state.selectedTokenId}
                noDataText="No API Tokens Generated"
                minRows={20}
            />
        );
    };

    toggleRow = id => {
        const selection = toggleRow(id, this.state.selection);
        this.updateSelection(selection);
    };

    toggleSelectAll = () => {
        const rowsLength = this.props.tokens.length;
        const tableRef = this.apiTokenModalTable.reactTable;
        const selection = toggleSelectAll(rowsLength, this.state.selection, tableRef);
        this.updateSelection(selection);
    };

    showTokenGenerationDetails = () =>
        this.props.currentGeneratedToken && this.props.currentGeneratedTokenMetadata;

    updateSelection = selection => this.setState({ selection });

    renderRowActionButtons = token => (
        <div className="border-2 border-r-2 border-base-400 bg-base-100">
            <Tooltip placement="top" overlay={<div>Revoke token</div>} mouseLeaveDelay={0}>
                <button
                    type="button"
                    className="p-1 px-4 hover:bg-primary-200 text-primary-600 hover:text-primary-700"
                    onClick={this.onRevokeHandler(token)}
                >
                    <Icon.Trash2 className="mt-1 h-4 w-4" />
                </button>
            </Tooltip>
        </div>
    );

    renderPanelButtons = () => {
        const selectionCount = this.state.selection.length;
        return (
            <React.Fragment>
                {selectionCount !== 0 && (
                    <PanelButton
                        icon={<Icon.Slash className="h-4 w-4 ml-1" />}
                        text={`Revoke (${selectionCount})`}
                        className="btn btn-alert"
                        onClick={this.showConfirmationDialog}
                        disabled={this.state.selectedTokenId !== null}
                    />
                )}
                {selectionCount === 0 && (
                    <PanelButton
                        icon={<Icon.Plus className="h-4 w-4 ml-1" />}
                        text="Generate Token"
                        className="btn btn-base"
                        onClick={this.openForm}
                        disabled={
                            this.props.tokenGenerationWizardOpen ||
                            this.state.selectedTokenId !== null
                        }
                    />
                )}
            </React.Fragment>
        );
    };

    renderHeader = () => (
        <header className="flex items-center w-full p-4 bg-primary-500 text-base-100 uppercase">
            <span className="flex flex-1">Configure API Tokens</span>
            <Icon.X className="h-4 w-4 cursor-pointer" onClick={this.closeModal} />
        </header>
    );

    renderTable = () => {
        const selectionCount = this.state.selection.length;
        const tokenCount = this.props.tokens.length;
        const headerText =
            selectionCount !== 0
                ? `${selectionCount} API Token${selectionCount === 1 ? '' : 's'} Selected`
                : `${tokenCount} API Token${tokenCount === 1 ? '' : 's'}`;
        return (
            <Panel header={headerText} buttons={this.renderPanelButtons()}>
                {this.showModalView()}
            </Panel>
        );
    };

    renderForm = () => {
        if (!this.props.tokenGenerationWizardOpen) {
            return null;
        }
        if (this.showTokenGenerationDetails()) {
            return null;
        }

        const buttons = (
            <PanelButton
                icon={<Icon.Save className="h-4 w-4" />}
                text="Generate"
                className="btn btn-success"
                onClick={this.onSubmit}
            />
        );

        return (
            <Panel header="Generate API Token" onClose={this.closeForm} buttons={buttons}>
                <APITokenForm />
            </Panel>
        );
    };

    renderDetails = () => {
        if (this.showTokenGenerationDetails()) {
            const { currentGeneratedToken, currentGeneratedTokenMetadata } = this.props;
            return (
                <Panel header="Generated Token" onClose={this.closeForm}>
                    <APITokenDetails
                        token={currentGeneratedToken}
                        metadata={currentGeneratedTokenMetadata}
                    />
                </Panel>
            );
        }
        if (this.state.selectedTokenId) {
            const selectedTokenMetadata = this.props.tokens.find(
                ({ id }) => id === this.state.selectedTokenId
            );
            if (selectedTokenMetadata) {
                return (
                    <Panel header="Token Details" onClose={this.unSelectRow}>
                        <APITokenDetails metadata={selectedTokenMetadata} />
                    </Panel>
                );
            }
        }
        return null;
    };

    render() {
        const { selection, showConfirmationDialog } = this.state;
        return (
            <Modal
                isOpen
                onRequestClose={this.props.onRequestClose}
                className="w-full lg:w-5/6 h-full"
            >
                {this.renderHeader()}
                <div className="flex flex-1 w-full bg-base-100">
                    {this.renderTable()}
                    {this.renderForm()}
                    {this.renderDetails()}
                </div>
                <Dialog
                    isOpen={showConfirmationDialog}
                    text={`Are you sure you want to revoke ${selection.length} token${
                        selection.length === 1 ? '' : 's'
                    }?`}
                    onConfirm={this.revokeTokens}
                    onCancel={this.hideConfirmationDialog}
                />
            </Modal>
        );
    }
}

const mapStateToProps = createStructuredSelector({
    tokenGenerationWizardOpen: selectors.tokenGenerationWizardOpen,
    currentGeneratedToken: selectors.getCurrentGeneratedToken,
    currentGeneratedTokenMetadata: selectors.getCurrentGeneratedTokenMetadata
});

const mapDispatchToProps = {
    startTokenGenerationWizard: actions.startTokenGenerationWizard,
    closeTokenGenerationWizard: actions.closeTokenGenerationWizard,
    generateAPIToken: actions.generateAPIToken.request,
    revokeAPITokens: actions.revokeAPITokens
};

export default connect(
    mapStateToProps,
    mapDispatchToProps
)(APITokensModal);
