import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { withRouter } from 'react-router-dom';
import { selectors } from 'reducers';
import { createSelector, createStructuredSelector } from 'reselect';

import PolicyCreationForm from 'Containers/Policies/PolicyCreationForm';
import PoliciesPreview from 'Containers/Policies/PoliciesPreview';
import { preFormatPolicyFields } from 'Containers/Policies/policyFormUtils';

/* PolicyCreationWizard component
 * 
 * wizardState.current holds a string representing the current state
 * `` -> `EDIT` -> `PRE_PREVIEW` -> `PREVIEW` -> `SAVE` || `CREATE` -> ''
 * 
 * wizardState.dryrun holds an object that has the policy dryrun preview data
 * 
 * wizardState.policy holds the enabled version of the policy to send to dryrun
 * 
 * wizardState.disabled holds a bool that says whether this policy is enabled/disabled
 * 
 * wizardState.isNew holds a bool that says whether this is a new policy or not 
 * to determine POST vs PUT request
 */

class PolicyCreationWizard extends Component {
    static propTypes = {
        policy: PropTypes.shape({}),
        wizardState: PropTypes.shape({}).isRequired
    };

    static defaultProps = {
        policy: null
    };

    renderEditPanel = () => {
        const { wizardState } = this.props;
        const show = wizardState.current === 'EDIT' || wizardState.current === 'PRE_PREVIEW';
        if (!show) return '';
        const policy = { ...this.props.policy };
        if (wizardState.disabled) policy.disabled = wizardState.disabled;
        return (
            <PolicyCreationForm
                initialValues={preFormatPolicyFields(policy)}
                submitForm={this.closeEditPanel}
            />
        );
    };

    renderPreviewPanel = () => {
        const { wizardState } = this.props;
        const show = wizardState.current === 'PREVIEW';
        if (!show) return '';
        return (
            <PoliciesPreview
                dryrun={wizardState.dryrun}
                policyDisabled={wizardState.disabled || false}
            />
        );
    };

    render() {
        return (
            <div className="flex flex-1 flex-col bg-base-200">
                {this.renderEditPanel()}
                {this.renderPreviewPanel()}
            </div>
        );
    }
}

const getPolicyId = (state, props) => props.match.params.policyId;

const getPolicy = createSelector(
    [selectors.getPolicyWizardState, selectors.getPoliciesById, getPolicyId],
    (wizardState, policiesById, policyId) => {
        const selectedPolicy = policiesById[policyId];
        if (wizardState.policy) return Object.assign({}, selectedPolicy, wizardState.policy);
        return selectedPolicy;
    }
);

const mapStateToProps = createStructuredSelector({
    wizardState: selectors.getPolicyWizardState,
    policy: getPolicy
});

export default withRouter(connect(mapStateToProps)(PolicyCreationWizard));
