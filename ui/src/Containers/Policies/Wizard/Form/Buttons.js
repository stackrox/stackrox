import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { selectors } from 'reducers';
import { actions as notificationActions } from 'reducers/notifications';
import { actions as wizardActions } from 'reducers/policies/wizard';
import { createStructuredSelector } from 'reselect';
import wizardStages from 'Containers/Policies/Wizard/wizardStages';

import { formValueSelector } from 'redux-form';
import * as Icon from 'react-feather';

import PanelButton from 'Components/PanelButton';
import { formatPolicyFields, getPolicyFormDataKeys } from 'Containers/Policies/Wizard/Form/utils';

class Buttons extends Component {
    static propTypes = {
        policies: PropTypes.arrayOf(PropTypes.object).isRequired,
        wizardPolicy: PropTypes.shape({
            id: PropTypes.string,
            enforcementActions: PropTypes.arrayOf(PropTypes.string)
        }).isRequired,
        formData: PropTypes.shape({
            name: PropTypes.string
        }),
        wizardPolicyIsNew: PropTypes.bool.isRequired,

        setWizardStage: PropTypes.func.isRequired,
        setWizardPolicy: PropTypes.func.isRequired,

        addToast: PropTypes.func.isRequired,
        removeToast: PropTypes.func.isRequired
    };

    static defaultProps = {
        formData: {
            name: ''
        }
    };

    goToPreview = () => {
        const dryRunOK = this.checkPreDryRun();
        if (dryRunOK) {
            // Format form data into the policy.
            const serverFormattedPolicy = formatPolicyFields(this.props.formData);
            const enabledPolicy = Object.assign({}, serverFormattedPolicy);

            // Need to add id and enforcement actions since those aren't in the form data.
            enabledPolicy.id = this.props.wizardPolicy.id;
            enabledPolicy.enforcementActions = this.props.wizardPolicy.enforcementActions;

            // Set the new policy information and proceed to preview.
            // (set prepreview so that dry run is picked up before preview panel)
            this.props.setWizardPolicy(enabledPolicy);
            this.props.setWizardStage(wizardStages.prepreview);
        }
    };

    checkPreDryRun = () => {
        if (!this.props.wizardPolicyIsNew) return true;

        const policyNames = this.props.policies.map(policy => policy.name);
        if (policyNames.find(name => name === this.props.formData.name)) {
            const error = `Could not add policy due to name validation: "${this.props.formData.name}
                " already exists`;
            this.showToast(error);
            return false;
        }
        return true;
    };

    showToast = error => {
        this.props.addToast(error);
        setTimeout(this.props.removeToast, 500);
    };

    render() {
        return (
            <PanelButton
                icon={<Icon.ArrowRight className="h-4 w-4" />}
                text="Next"
                className="btn btn-base"
                onClick={this.goToPreview}
            />
        );
    }
}

const getFormData = state =>
    formValueSelector('policyCreationForm')(state, ...getPolicyFormDataKeys());

const mapStateToProps = createStructuredSelector({
    policies: selectors.getFilteredPolicies,
    wizardPolicy: selectors.getWizardPolicy,
    formData: getFormData,
    wizardPolicyIsNew: selectors.getWizardIsNew
});

const mapDispatchToProps = {
    setWizardPolicy: wizardActions.setWizardPolicy,
    setWizardStage: wizardActions.setWizardStage,

    addToast: notificationActions.addNotification,
    removeToast: notificationActions.removeOldestNotification
};

export default connect(
    mapStateToProps,
    mapDispatchToProps
)(Buttons);
