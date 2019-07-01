import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { withRouter } from 'react-router-dom';
import ReactRouterPropTypes from 'react-router-prop-types';
import { connect } from 'react-redux';
import { selectors } from 'reducers';
import { actions as backendActions } from 'reducers/policies/backend';
import { actions as dialogueActions } from 'reducers/policies/notifier';
import { actions as pageActions } from 'reducers/policies/page';
import { actions as tableActions } from 'reducers/policies/table';
import { createStructuredSelector } from 'reselect';

import CustomDialogue from 'Components/CustomDialogue';
import uniq from 'lodash/uniq';
import policyBulkActions from './policyBulkActions';
import DialogueNotifiers from './DialogueNotifiers';

// PolicyBulkActionDialogue is the pop-up that displays when performing bulk policy operations.
class PolicyBulkActionDialogue extends Component {
    static propTypes = {
        policiesAction: PropTypes.string.isRequired,
        policies: PropTypes.arrayOf(PropTypes.object).isRequired,
        selectedPolicyIds: PropTypes.arrayOf(PropTypes.string).isRequired,

        selectPolicyIds: PropTypes.func.isRequired,
        deletePolicies: PropTypes.func.isRequired,

        notifiers: PropTypes.arrayOf(
            PropTypes.shape({
                id: PropTypes.string.isRequired
            })
        ).isRequired,
        selectedNotifierIds: PropTypes.arrayOf(PropTypes.string),
        setPolicyNotifiers: PropTypes.func.isRequired,
        enablePoliciesNotification: PropTypes.func.isRequired,
        disablePoliciesNotification: PropTypes.func.isRequired,

        closeWizard: PropTypes.func.isRequired,
        closeDialogue: PropTypes.func.isRequired,

        match: ReactRouterPropTypes.match.isRequired
    };

    static defaultProps = {
        selectedNotifierIds: []
    };

    selectedPolicyNotifiers = () => {
        return uniq(
            this.props.policies
                .filter(
                    policy =>
                        this.props.selectedPolicyIds.find(id => id === policy.id) &&
                        policy.notifiers.length > 0
                )
                .flatMap(policy => policy.notifiers)
        );
    };

    deletePolicies = () => {
        const policyIds = this.props.selectedPolicyIds;
        policyIds.forEach(rowId => {
            // close the view panel if that policy is being deleted
            if (rowId === this.props.match.params.policyId) {
                this.props.closeWizard();
            }
        });
        // Remove selected policies, close dialogue, then begin deletion.
        this.props.selectPolicyIds([]);
        this.props.closeDialogue();
        this.props.deletePolicies(policyIds);
    };

    enableNotification = () => {
        const selectedNotifiers =
            this.props.notifiers.length === 1
                ? this.props.notifiers.map(notifier => notifier.id)
                : this.props.selectedNotifierIds;
        this.props.enablePoliciesNotification(this.props.selectedPolicyIds, selectedNotifiers);
        this.props.setPolicyNotifiers([]);
        this.props.closeDialogue();
    };

    disableNotification = () => {
        const selectedPolicyNotifiers = this.selectedPolicyNotifiers();
        const selectedNotifiers =
            selectedPolicyNotifiers.length === 1
                ? selectedPolicyNotifiers
                : this.props.selectedNotifierIds;
        this.props.disablePoliciesNotification(this.props.selectedPolicyIds, selectedNotifiers);
        this.props.setPolicyNotifiers([]);
        this.props.closeDialogue();
    };

    getDialogueTitle = () => {
        return `${this.props.policiesAction}`;
    };

    getDialogueText = () => {
        const numSelectedRows = this.props.selectedPolicyIds.length;
        const selectedPolicyNotifiers = this.selectedPolicyNotifiers();
        const suffix = `${numSelectedRows === 1 ? 'policy' : 'policies'}`;

        switch (this.props.policiesAction) {
            case policyBulkActions.deletePolicies:
                return `Are you sure you want to delete ${numSelectedRows} ${suffix}?`;
            case policyBulkActions.enableNotification:
                if (this.props.notifiers.length === 0) {
                    return `No notifiers configured!`;
                }
                return '';
            case policyBulkActions.disableNotification:
                if (selectedPolicyNotifiers.length === 0) {
                    return `No notifiers configured for selected ${suffix}!`;
                }
                return `Are you sure you want to disable notification for ${numSelectedRows} ${suffix}?`;
            default:
                return '';
        }
    };

    getChildren = () => {
        switch (this.props.policiesAction) {
            case policyBulkActions.enableNotification:
            case policyBulkActions.disableNotification:
                return <DialogueNotifiers />;
            default:
                return null;
        }
    };

    onConfirm = () => {
        switch (this.props.policiesAction) {
            case policyBulkActions.deletePolicies:
                return this.deletePolicies();
            case policyBulkActions.enableNotification:
                return this.enableNotification();
            case policyBulkActions.disableNotification:
                return this.disableNotification();
            default:
                return null;
        }
    };

    getConfirmText = () => {
        if (this.props.policiesAction === policyBulkActions.enableNotification) {
            return 'Enable';
        }
        return 'Confirm';
    };

    confirmDisabled = () => {
        const { selectedNotifierIds } = this.props;
        const selectedPolicyNotifiers = this.selectedPolicyNotifiers();
        switch (this.props.policiesAction) {
            case policyBulkActions.enableNotification:
                return this.props.notifiers.length !== 1 && selectedNotifierIds.length === 0;
            case policyBulkActions.disableNotification:
                return selectedPolicyNotifiers.length !== 1 && selectedNotifierIds.length === 0;
            default:
                return false;
        }
    };

    onClose = () => {
        this.props.setPolicyNotifiers([]);
        this.props.closeDialogue();
    };

    render() {
        if (!this.props.policiesAction || this.props.selectedPolicyIds.length === 0) return null;

        return (
            <CustomDialogue
                title={this.getDialogueTitle()}
                text={this.getDialogueText()}
                onConfirm={this.onConfirm}
                confirmText={this.getConfirmText()}
                confirmDisabled={this.confirmDisabled()}
                onCancel={this.onClose}
            >
                {this.getChildren()}
            </CustomDialogue>
        );
    }
}

const mapStateToProps = createStructuredSelector({
    policies: selectors.getFilteredPolicies,
    policiesAction: selectors.getPoliciesAction,
    selectedPolicyIds: selectors.getSelectedPolicyIds,
    notifiers: selectors.getNotifiers,
    selectedNotifierIds: selectors.getPolicyNotifiers
});

const mapDispatchToProps = {
    selectPolicyIds: tableActions.selectPolicyIds,
    deletePolicies: backendActions.deletePolicies,

    enablePoliciesNotification: backendActions.enablePoliciesNotification,
    disablePoliciesNotification: backendActions.disablePoliciesNotification,
    setPolicyNotifiers: dialogueActions.setPolicyNotifiers,

    closeWizard: pageActions.closeWizard,
    closeDialogue: pageActions.closeDialogue
};

export default withRouter(
    connect(
        mapStateToProps,
        mapDispatchToProps
    )(PolicyBulkActionDialogue)
);
