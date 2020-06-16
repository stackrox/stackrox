import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { ArrowLeft, ArrowRight } from 'react-feather';
import { createSelector, createStructuredSelector } from 'reselect';

import { selectors } from 'reducers';
import { actions as wizardActions } from 'reducers/policies/wizard';
import wizardStages from 'Containers/Policies/Wizard/wizardStages';
import PanelButton from 'Components/PanelButton';
import { knownBackendFlags, isBackendFeatureFlagEnabled } from 'utils/featureFlags';

function PreviewButtons({ setWizardStage, BPLisEnabled }) {
    function goBackToEditBPL() {
        if (BPLisEnabled) {
            setWizardStage(wizardStages.editBPL);
        } else {
            setWizardStage(wizardStages.edit);
        }
    }

    function goToEnforcement() {
        setWizardStage(wizardStages.enforcement);
    }

    return (
        <>
            <PanelButton
                icon={<ArrowLeft className="h-4 w-4" />}
                className="btn btn-base mr-2"
                onClick={goBackToEditBPL}
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
    BPLisEnabled: PropTypes.bool.isRequired,
};

const getBPLisEnabled = createSelector([selectors.getFeatureFlags], (featureFlags) => {
    return isBackendFeatureFlagEnabled(
        featureFlags,
        knownBackendFlags.ROX_BOOLEAN_POLICY_LOGIC,
        false
    );
});

const mapStateToProps = createStructuredSelector({
    BPLisEnabled: getBPLisEnabled,
});

const mapDispatchToProps = {
    setWizardStage: wizardActions.setWizardStage,
};

export default connect(mapStateToProps, mapDispatchToProps)(PreviewButtons);
