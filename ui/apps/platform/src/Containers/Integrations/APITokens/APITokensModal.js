import React, { Component } from 'react';
import { connect } from 'react-redux';
import PropTypes from 'prop-types';
import * as Icon from 'react-feather';
import { createStructuredSelector } from 'reselect';

import { actions } from 'reducers/apitokens';
import { selectors } from 'reducers';

import Modal from 'Components/Modal';
import Dialog from 'Components/Dialog';
import { PanelNew, PanelBody, PanelHead, PanelHeadEnd, PanelTitle } from 'Components/Panel';
import PanelButton from 'Components/PanelButton';

import APITokenForm from './APITokenForm';
import APITokenDetails from './APITokenDetails';

class APITokensModal extends Component {
    static propTypes = {
        tokens: PropTypes.arrayOf(
            PropTypes.shape({
                id: PropTypes.string.isRequired,
                name: PropTypes.string.isRequired,
                roles: PropTypes.arrayOf(PropTypes.string).isRequired,
            })
        ).isRequired,
        onRequestClose: PropTypes.func.isRequired,
        startTokenGenerationWizard: PropTypes.func.isRequired,
        closeTokenGenerationWizard: PropTypes.func.isRequired,
        generateAPIToken: PropTypes.func.isRequired,
        revokeAPITokens: PropTypes.func.isRequired,
        currentGeneratedToken: PropTypes.string,
        currentGeneratedTokenMetadata: PropTypes.shape({
            name: PropTypes.string.isRequired,
            roles: PropTypes.arrayOf(PropTypes.string).isRequired,
        }),
        selectedIntegration: PropTypes.shape({
            id: PropTypes.string,
        }),
    };

    static defaultProps = {
        currentGeneratedToken: '',
        currentGeneratedTokenMetadata: null,
        selectedIntegration: null,
    };

    constructor(props) {
        super(props);

        this.state = {
            selectedTokenId: null,
            showConfirmationDialog: false,
            selection: [],
        };
    }

    componentDidMount() {
        this.props.startTokenGenerationWizard();
        if (this.props.selectedIntegration) {
            this.setState({ selectedTokenId: this.props.selectedIntegration.id });
        }
    }

    onRowClick = (row) => {
        this.setState({ selectedTokenId: row.id });
    };

    onRevokeHandler = (token) => (e) => {
        e.stopPropagation();
        this.revokeTokens(token);
    };

    onSubmit = () => {
        this.props.generateAPIToken();
    };

    revokeTokens = ({ id }) => {
        if (id) {
            this.props.revokeAPITokens([id]);
        } else {
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

    showConfirmationDialog = () => {
        this.setState({ showConfirmationDialog: true });
    };

    hideConfirmationDialog = () => {
        this.setState({ showConfirmationDialog: false });
    };

    showTokenGenerationDetails = () =>
        this.props.currentGeneratedToken && this.props.currentGeneratedTokenMetadata;

    renderForm = () => {
        if (this.showTokenGenerationDetails() || this.state.selectedTokenId) {
            return null;
        }

        const buttons = (
            <PanelButton
                icon={<Icon.Save className="h-4 w-4" />}
                className="btn btn-success mr-3"
                onClick={this.onSubmit}
                tooltip="Generate"
            >
                Generate
            </PanelButton>
        );

        return (
            <PanelNew testid="panel">
                <PanelHead>
                    <PanelTitle isUpperCase testid="panel-header" text="Generate API Token" />
                    <PanelHeadEnd>{buttons}</PanelHeadEnd>
                </PanelHead>
                <PanelBody>
                    <APITokenForm />
                </PanelBody>
            </PanelNew>
        );
    };

    renderDetails = () => {
        if (this.showTokenGenerationDetails()) {
            const { currentGeneratedToken, currentGeneratedTokenMetadata } = this.props;
            return (
                <PanelNew testid="panel">
                    <PanelHead>
                        <PanelTitle isUpperCase testid="panel-header" text="Generated Token" />
                    </PanelHead>
                    <PanelBody>
                        <APITokenDetails
                            token={currentGeneratedToken}
                            metadata={currentGeneratedTokenMetadata}
                        />
                    </PanelBody>
                </PanelNew>
            );
        }
        if (this.state.selectedTokenId) {
            const selectedTokenMetadata = this.props.tokens.find(
                ({ id }) => id === this.state.selectedTokenId
            );
            if (selectedTokenMetadata) {
                return (
                    <PanelNew testid="panel">
                        <PanelHead>
                            <PanelTitle isUpperCase testid="panel-header" text="Token Details" />
                        </PanelHead>
                        <PanelBody>
                            <APITokenDetails metadata={selectedTokenMetadata} />
                        </PanelBody>
                    </PanelNew>
                );
            }
        }
        return null;
    };

    render() {
        const { selection, showConfirmationDialog } = this.state;
        return (
            <Modal isOpen onRequestClose={this.props.onRequestClose} className="max-w-184">
                <header className="flex items-center p-4 bg-primary-500 text-base-100 uppercase">
                    <span className="flex flex-1">Configure API Tokens</span>
                    <Icon.X className="h-4 w-4 cursor-pointer" onClick={this.closeModal} />
                </header>
                <div className="flex flex-1 relative bg-base-100">
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
    currentGeneratedTokenMetadata: selectors.getCurrentGeneratedTokenMetadata,
});

const mapDispatchToProps = {
    startTokenGenerationWizard: actions.startTokenGenerationWizard,
    closeTokenGenerationWizard: actions.closeTokenGenerationWizard,
    generateAPIToken: actions.generateAPIToken.request,
    revokeAPITokens: actions.revokeAPITokens,
};

export default connect(mapStateToProps, mapDispatchToProps)(APITokensModal);
