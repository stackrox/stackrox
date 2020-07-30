import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import ReactRouterPropTypes from 'react-router-prop-types';
import { withRouter } from 'react-router-dom';
import { ArrowLeft, Save } from 'react-feather';

import { selectors } from 'reducers';
import { createStructuredSelector } from 'reselect';
import { actions as wizardActions } from 'reducers/policies/wizard';
import wizardStages from 'Containers/Policies/Wizard/wizardStages';
import PanelButton from 'Components/PanelButton';

function EnforcementButtons({ history, match, wizardPolicyIsNew, setWizardStage }) {
    function goBackToPreview() {
        setWizardStage(wizardStages.preview);
    }

    function onSubmit() {
        if (wizardPolicyIsNew) {
            setWizardStage(wizardStages.create);
        } else {
            setWizardStage(wizardStages.save);
            history.push(`/main/policies/${match.params.policyId}`);
        }
    }
    return (
        <>
            <PanelButton
                icon={<ArrowLeft className="h-4 w-4" />}
                className="btn btn-base mr-2"
                onClick={goBackToPreview}
                tooltip="Back to previous step"
            >
                Previous
            </PanelButton>
            <PanelButton
                icon={<Save className="h-4 w-4" />}
                className="btn btn-success mr-2"
                onClick={onSubmit}
                tooltip="Save policy"
            >
                Save
            </PanelButton>
        </>
    );
}

EnforcementButtons.propTypes = {
    history: ReactRouterPropTypes.location.isRequired,
    match: ReactRouterPropTypes.location.isRequired,
    wizardPolicyIsNew: PropTypes.bool.isRequired,
    setWizardStage: PropTypes.func.isRequired,
};

const mapStateToProps = createStructuredSelector({
    wizardPolicyIsNew: selectors.getWizardIsNew,
});

const mapDispatchToProps = {
    setWizardStage: wizardActions.setWizardStage,
};

export default withRouter(connect(mapStateToProps, mapDispatchToProps)(EnforcementButtons));
