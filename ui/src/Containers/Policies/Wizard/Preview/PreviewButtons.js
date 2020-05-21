import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { ArrowLeft, ArrowRight } from 'react-feather';

import { actions as wizardActions } from 'reducers/policies/wizard';
import wizardStages from 'Containers/Policies/Wizard/wizardStages';
import PanelButton from 'Components/PanelButton';

function PreviewButtons({ setWizardStage }) {
    function goBackToEdit() {
        setWizardStage(wizardStages.edit);
    }

    function goToEnforcement() {
        setWizardStage(wizardStages.enforcement);
    }

    return (
        <>
            <PanelButton
                icon={<ArrowLeft className="h-4 w-4" />}
                className="btn btn-base mr-2"
                onClick={goBackToEdit}
                tooltip="Back to previous step"
            >
                Previous
            </PanelButton>
            <PanelButton
                icon={<ArrowRight className="h-4 w-4" />}
                className="btn btn-base mr-2"
                onClick={goToEnforcement}
                tooltip="Go to next step"
            >
                Next
            </PanelButton>
        </>
    );
}

PreviewButtons.propTypes = {
    setWizardStage: PropTypes.func.isRequired,
};

const mapDispatchToProps = {
    setWizardStage: wizardActions.setWizardStage,
};

export default connect(null, mapDispatchToProps)(PreviewButtons);
