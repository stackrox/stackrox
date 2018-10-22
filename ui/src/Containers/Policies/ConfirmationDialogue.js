import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { withRouter } from 'react-router';
import ReactRouterPropTypes from 'react-router-prop-types';
import { connect } from 'react-redux';
import { selectors } from 'reducers';
import { actions as backendActions } from 'reducers/policies/backend';
import { actions as pageActions } from 'reducers/policies/page';
import { actions as tableActions } from 'reducers/policies/table';
import { createStructuredSelector } from 'reselect';

import Dialog from 'Components/Dialog';

// ConfirmationDialogue is the pop-up that displays when deleting policies from the table.
class ConfirmationDialogue extends Component {
    static propTypes = {
        dialogueOpen: PropTypes.bool.isRequired,
        selectedPolicyIds: PropTypes.arrayOf(PropTypes.string).isRequired,

        selectPolicyIds: PropTypes.func.isRequired,
        deletePolicies: PropTypes.func.isRequired,

        closeWizard: PropTypes.func.isRequired,
        closeDialogue: PropTypes.func.isRequired,

        match: ReactRouterPropTypes.match.isRequired
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

    render() {
        if (!this.props.dialogueOpen) return null;

        const numSelectedRows = this.props.selectedPolicyIds.length;
        return (
            <Dialog
                isOpen={!this.props.dialogueOpen || numSelectedRows !== 0}
                text={`Are you sure you want to delete ${numSelectedRows} ${
                    numSelectedRows === 1 ? 'policy' : 'policies'
                }?`}
                onConfirm={this.deletePolicies}
                onCancel={this.props.closeDialogue}
            />
        );
    }
}

const mapStateToProps = createStructuredSelector({
    dialogueOpen: selectors.getDialogueOpen,
    selectedPolicyIds: selectors.getSelectedPolicyIds
});

const mapDispatchToProps = {
    selectPolicyIds: tableActions.selectPolicyIds,
    deletePolicies: backendActions.deletePolicies,

    closeWizard: pageActions.closeWizard,
    closeDialogue: pageActions.closeDialogue
};

export default withRouter(
    connect(
        mapStateToProps,
        mapDispatchToProps
    )(ConfirmationDialogue)
);
