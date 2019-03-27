import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import * as Icon from 'react-feather';
import { selectors } from 'reducers';
import { actions as backendActions } from 'reducers/network/backend';
import { actions as dialogueActions } from 'reducers/network/dialogue';

import Modal from 'Components/Modal';
import dialogueStages from './dialogueStages';

// ConfirmationDialogue is the pop-up that displays when deleting policies from the table.
class ApplyModification extends Component {
    static propTypes = {
        dialogueStage: PropTypes.string.isRequired,

        applyModification: PropTypes.func.isRequired,
        setDialogueStage: PropTypes.func.isRequired
    };

    onConfirm = () => {
        this.props.setDialogueStage(dialogueStages.closed);
        this.props.applyModification();
    };

    onClose = () => {
        this.props.setDialogueStage(dialogueStages.closed);
    };

    render() {
        const { dialogueStage } = this.props;
        if (dialogueStage !== dialogueStages.application) return null;

        return (
            <Modal isOpen onRequestClose={this.onClose}>
                <div className="flex items-center w-full p-3 bg-primary-700 text-xl uppercase text-base-100">
                    <div className="flex flex-1">Apply Network Policies</div>
                    <Icon.X className="h-4 w-4 cursor-pointer" onClick={this.onClose} />
                </div>
                <div className="leading-normal p-3 border-b border-base-300 bg-base-100">
                    Applying network policies can impact running deployments.
                    <br />
                    Do you wish to continue?
                </div>
                <div className="flex m-3 justify-end">
                    <button type="button" className="btn btn-base" onClick={this.onClose}>
                        Cancel
                    </button>
                    <button
                        type="button"
                        className="btn ml-3 bg-primary-600 text-base-100 h-9 hover:bg-primary-700"
                        onClick={this.onConfirm}
                    >
                        Apply
                    </button>
                </div>
            </Modal>
        );
    }
}

const mapStateToProps = createStructuredSelector({
    dialogueStage: selectors.getNetworkDialogueStage
});

const mapDispatchToProps = {
    applyModification: backendActions.applyNetworkPolicyModification.request,
    setDialogueStage: dialogueActions.setNetworkDialogueStage
};

export default connect(
    mapStateToProps,
    mapDispatchToProps
)(ApplyModification);
