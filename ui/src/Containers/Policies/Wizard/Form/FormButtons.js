import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { getFormValues } from 'redux-form';
import { createStructuredSelector } from 'reselect';
import { ArrowRight } from 'react-feather';

import { selectors } from 'reducers';
import wizardStages from 'Containers/Policies/Wizard/wizardStages';
import { actions as notificationActions } from 'reducers/notifications';
import { actions as wizardActions } from 'reducers/policies/wizard';
import PanelButton from 'Components/PanelButton';
import { formatPolicyFields } from 'Containers/Policies/Wizard/Form/utils';

function FormButtons({
    policies,
    wizardPolicy,
    formData,
    wizardPolicyIsNew,
    setWizardStage,
    setWizardPolicy,
    addToast,
    removeToast
}) {
    function goToPreview() {
        const dryRunOK = checkPreDryRun();
        if (dryRunOK) {
            // Format form data into the policy.
            const serverFormattedPolicy = formatPolicyFields(formData);
            const enabledPolicy = Object.assign({}, serverFormattedPolicy);

            // Need to add id and enforcement actions since those aren't in the form data.
            enabledPolicy.id = wizardPolicy.id;
            enabledPolicy.enforcementActions = wizardPolicy.enforcementActions;

            // Set the new policy information and proceed to preview.
            // (set prepreview so that dry run is picked up before preview panel)
            setWizardPolicy(enabledPolicy);
            setWizardStage(wizardStages.prepreview);
        }
    }

    function checkPreDryRun() {
        if (!wizardPolicyIsNew) return true;

        const policyNames = policies.map(policy => policy.name);
        if (policyNames.find(name => name === formData.name)) {
            const error = `Could not add policy due to name validation: "${formData.name}
                " already exists`;
            showToast(error);
            return false;
        }
        return true;
    }

    function showToast(error) {
        addToast(error);
        setTimeout(removeToast, 500);
    }

    return (
        <PanelButton
            icon={<ArrowRight className="h-4 w-4" />}
            className="btn btn-base"
            onClick={goToPreview}
            tooltip="Go to next step"
        >
            Next
        </PanelButton>
    );
}

FormButtons.propTypes = {
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

FormButtons.defaultProps = {
    formData: {
        name: ''
    }
};

const mapStateToProps = createStructuredSelector({
    policies: selectors.getFilteredPolicies,
    wizardPolicy: selectors.getWizardPolicy,
    formData: getFormValues('policyCreationForm'),
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
)(FormButtons);
