import React, { ReactElement } from 'react';
import { connect } from 'react-redux';
import { ArrowLeft } from 'react-feather';

import { actions as wizardActions } from 'reducers/policies/wizard';
import wizardStages from 'Containers/Policies/Wizard/wizardStages';
import PanelButton from 'Components/PanelButton';
import SavePolicyButton from '../SavePolicyButton';

type EnforcementButtonsProps = {
    setWizardStage: (string) => void;
};

function EnforcementButtons({ setWizardStage }: EnforcementButtonsProps): ReactElement {
    function goBackToPreview() {
        setWizardStage(wizardStages.preview);
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
            <SavePolicyButton />
        </>
    );
}

const mapDispatchToProps = {
    setWizardStage: wizardActions.setWizardStage,
};

export default connect(null, mapDispatchToProps)(EnforcementButtons);
