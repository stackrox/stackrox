import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { withRouter } from 'react-router-dom';
import { connect } from 'react-redux';
import { selectors } from 'reducers';
import { actions as backendActions } from 'reducers/policies/backend';
import { actions as dialogueActions } from 'reducers/network/dialogue';
import { createStructuredSelector } from 'reselect';

import Modal from 'Components/Modal';
import * as Icon from 'react-feather';
import dialogueStages from '../Network/Dialogue/dialogueStages';
import NotifyOne from '../Network/Dialogue/NotifyOne';
import NotifyMany from '../Network/Dialogue/NotifyMany';

class NotifierDialogue extends Component {
    static propTypes = {
        dialogueStage: PropTypes.string.isRequired,
        notifiers: PropTypes.arrayOf(
            PropTypes.shape({
                id: PropTypes.string.isRequired
            })
        ).isRequired,
        selectedNotifierIds: PropTypes.arrayOf(PropTypes.string),
        selectedPolicyIds: PropTypes.arrayOf(PropTypes.string).isRequired,
        enablePoliciesNotification: PropTypes.func.isRequired,

        setNetworkNotifiers: PropTypes.func.isRequired,
        setDialogueStage: PropTypes.func.isRequired
    };

    static defaultProps = {
        selectedNotifierIds: []
    };

    onEnableNotification = () => {
        const selectedNotifiers =
            this.props.notifiers.length === 1
                ? this.props.notifiers.map(notifier => notifier.id)
                : this.props.selectedNotifierIds;
        this.props.enablePoliciesNotification(this.props.selectedPolicyIds, selectedNotifiers);
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
        const { notifiers, selectedNotifierIds } = this.props;
        const notifyDisabled =
            notifiers.length !== 1 && (!selectedNotifierIds || selectedNotifierIds.length === 0);
        return (
            <Modal isOpen onRequestClose={this.onClose}>
                <div className="flex items-center w-full p-3 bg-primary-700 text-xl uppercase text-base-100 uppercase">
                    <div className="flex flex-1">Enable Notification</div>
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
                        onClick={this.onEnableNotification}
                        disabled={notifyDisabled}
                    >
                        Enable
                    </button>
                </div>
            </Modal>
        );
    }
}

const mapStateToProps = createStructuredSelector({
    selectedPolicyIds: selectors.getSelectedPolicyIds,
    notifiers: selectors.getNotifiers,
    selectedNotifierIds: selectors.getNetworkNotifiers,
    dialogueStage: selectors.getNetworkDialogueStage
});
const mapDispatchToProps = {
    enablePoliciesNotification: backendActions.enablePoliciesNotification,
    setNetworkNotifiers: dialogueActions.setNetworkNotifiers,
    setDialogueStage: dialogueActions.setNetworkDialogueStage
};

export default withRouter(
    connect(
        mapStateToProps,
        mapDispatchToProps
    )(NotifierDialogue)
);
