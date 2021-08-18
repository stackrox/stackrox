import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import cloneDeep from 'lodash/cloneDeep';
import { Copy, Download, Edit } from 'react-feather';
import { initialize } from 'redux-form';

import { selectors } from 'reducers';
import { actions as notificationActions } from 'reducers/notifications';
import { actions as wizardActions } from 'reducers/policies/wizard';
import wizardStages from 'Containers/Policies/Wizard/wizardStages';
import PanelButton from 'Components/PanelButton';
import { exportPolicies } from 'services/PoliciesService';

function DetailsButtons({
    wizardPolicy,
    setWizardStage,
    setWizardPolicy,
    addToast,
    removeToast,
    initializeForm,
}) {
    function goToEdit() {
        setWizardStage(wizardStages.edit);
    }

    function exportOnePolicy() {
        const policiesToExport = [wizardPolicy.id];
        exportPolicies(policiesToExport).catch((err) => {
            addToast(`Could not export the policy: ${err.message}`);
            setTimeout(removeToast, 5000);
        });
    }

    function onPolicyClone() {
        const newPolicy = cloneDeep(wizardPolicy);
        /*
         * Default policies will have the "criteriaLocked" and "mitreVectorsLocked" fields set to true.
         * When we clone these policies, we'll need to set them to false to allow users to edit
         * both the policy criteria and mitre attack vectors
         */
        newPolicy.mitreVectorsLocked = false;
        newPolicy.criteriaLocked = false;
        newPolicy.id = '';
        newPolicy.name += ' (COPY)';
        setWizardPolicy(newPolicy);
        initializeForm('policyCreationForm', newPolicy);
        setWizardStage(wizardStages.edit);
    }

    return (
        <>
            <PanelButton
                icon={<Copy className="h-4 w-4" />}
                className="btn btn-base mr-2"
                onClick={onPolicyClone}
                tooltip="Clone policy"
            >
                Clone
            </PanelButton>
            <PanelButton
                icon={<Download className="h-4 w-4" />}
                className="btn btn-base mr-2"
                onClick={exportOnePolicy}
                tooltip="Export policy"
                dataTestId="single-policy-export"
            >
                Export
            </PanelButton>
            <PanelButton
                icon={<Edit className="h-4 w-4" />}
                className="btn btn-base mr-2"
                onClick={goToEdit}
                tooltip="Edit policy"
            >
                Edit
            </PanelButton>
        </>
    );
}

DetailsButtons.propTypes = {
    wizardPolicy: PropTypes.shape({
        id: PropTypes.string,
    }).isRequired,
    setWizardStage: PropTypes.func.isRequired,
    setWizardPolicy: PropTypes.func.isRequired,
    addToast: PropTypes.func.isRequired,
    removeToast: PropTypes.func.isRequired,
    initializeForm: PropTypes.func.isRequired,
};

const mapStateToProps = createStructuredSelector({
    wizardPolicy: selectors.getWizardPolicy,
});

const mapDispatchToProps = {
    setWizardStage: wizardActions.setWizardStage,
    setWizardPolicy: wizardActions.setWizardPolicy,
    addToast: notificationActions.addNotification,
    removeToast: notificationActions.removeOldestNotification,
    initializeForm: initialize,
};

export default connect(mapStateToProps, mapDispatchToProps)(DetailsButtons);
