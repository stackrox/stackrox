import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { ArrowRight } from 'react-feather';

import wizardStages from 'Containers/Policies/Wizard/wizardStages';
import { actions as wizardActions } from 'reducers/policies/wizard';
import PanelButton from 'Components/PanelButton';

function FormButtons({ setWizardStage }) {
    function goToCriteria() {
        setWizardStage(wizardStages.editBPL);
    }

    return (
        <PanelButton
            icon={<ArrowRight className="h-4 w-4" />}
            className="btn btn-base mr-2"
            onClick={goToCriteria}
            tooltip="Go to policy criteria"
        >
            Next
        </PanelButton>
    );
}

FormButtons.propTypes = {
    setWizardStage: PropTypes.func.isRequired,
};

const mapDispatchToProps = {
    setWizardStage: wizardActions.setWizardStage,
};

export default connect(null, mapDispatchToProps)(FormButtons);
