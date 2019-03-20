import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import * as Icon from 'react-feather';
import { selectors } from 'reducers';
import { actions as dialogueActions } from 'reducers/network/dialogue';

import Modal from 'Components/Modal';
import NotifyOne from './NotifyOne';
import NotifyMany from './NotifyMany';
import dialogueStages from './dialogueStages';

// ConfirmationDialogue is the pop-up that displays when deleting policies from the table.
class NotifyModification extends Component {
    static propTypes = {
        dialogueStage: PropTypes.string.isRequired,
        notifiers: PropTypes.arrayOf(
            PropTypes.shape({
                id: PropTypes.string.isRequired
            })
        ).isRequired,
        selectedNotifiers: PropTypes.arrayOf(PropTypes.string),

        setNetworkNotifiers: PropTypes.func.isRequired,
        notifyModification: PropTypes.func.isRequired,
        setDialogueStage: PropTypes.func.isRequired
    };

    static defaultProps = {
        selectedNotifiers: []
    };

    onNotify = () => {
        // If we only have one notifier, then assume any request is for that notifier.
        if (this.props.notifiers.length === 1) {
            this.props.setNetworkNotifiers(this.props.notifiers.map(notifier => notifier.id));
        }
        this.props.notifyModification();
        this.props.setDialogueStage(dialogueStages.closed);
        this.props.setNetworkNotifiers([]);
    };

    onClose = () => {
        this.props.setDialogueStage(dialogueStages.closed);
        this.props.setNetworkNotifiers([]);
    };

    render() {
        const { dialogueStage } = this.props;
        if (dialogueStage !== dialogueStages.notification) {
            return null;
        }

        // If we have one notifier, we can assume that is the notifier to send to, otherwise, some need to be selected.
        const { notifiers, selectedNotifiers } = this.props;
        const notifyDisabled =
            notifiers.length !== 1 && (!selectedNotifiers || selectedNotifiers.length === 0);
        return (
            <Modal isOpen onRequestClose={this.onClose}>
                <div className="flex items-center w-full p-3 bg-primary-700 text-xl uppercase text-base-100 uppercase">
                    <div className="flex flex-1">Share Network Policy YAML With Team</div>
                    <Icon.X className="ml-6 h-4 w-4 cursor-pointer" onClick={this.onClose} />
                </div>
                <NotifyOne />
                <NotifyMany />
                <div className="flex m-3 justify-end">
                    <button type="button" className="btn btn-base" onClick={this.onClose}>
                        Cancel
                    </button>
                    <button
                        type="button"
                        className="btn ml-3 bg-primary-600 text-base-100 h-9 hover:bg-primary-700"
                        onClick={this.onNotify}
                        disabled={notifyDisabled}
                    >
                        Notify
                    </button>
                </div>
            </Modal>
        );
    }
}

const mapStateToProps = createStructuredSelector({
    notifiers: selectors.getNotifiers,
    selectedNotifiers: selectors.getNetworkNotifiers,
    dialogueStage: selectors.getNetworkDialogueStage
});

const mapDispatchToProps = {
    notifyModification: dialogueActions.notifyNetworkPolicyModification,
    setNetworkNotifiers: dialogueActions.setNetworkNotifiers,
    setDialogueStage: dialogueActions.setNetworkDialogueStage
};

export default connect(
    mapStateToProps,
    mapDispatchToProps
)(NotifyModification);
