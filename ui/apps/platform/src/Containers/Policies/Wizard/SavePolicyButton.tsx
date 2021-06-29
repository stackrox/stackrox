import React, { ReactElement } from 'react';
import { connect } from 'react-redux';
import { useHistory, useRouteMatch } from 'react-router-dom';
import { Save } from 'react-feather';

import { selectors } from 'reducers';
import { createStructuredSelector } from 'reselect';
import { actions as wizardActions } from 'reducers/policies/wizard';
import wizardStages from 'Containers/Policies/Wizard/wizardStages';
import PanelButton from 'Components/PanelButton';

type SavePolicyButtonProps = {
    wizardPolicyIsNew: boolean;
    setWizardStage: (string) => void;
};

function SavePolicyButton({
    wizardPolicyIsNew,
    setWizardStage,
}: SavePolicyButtonProps): ReactElement {
    const history = useHistory();
    const match = useRouteMatch();
    function onSubmit() {
        if (wizardPolicyIsNew) {
            setWizardStage(wizardStages.create);
        } else {
            setWizardStage(wizardStages.save);
            history.push(`/main/policies/${match.params.policyId as string}`);
        }
    }
    return (
        <PanelButton
            icon={<Save className="h-4 w-4" />}
            className="btn btn-success mr-2"
            onClick={onSubmit}
            tooltip="Save policy"
        >
            Save
        </PanelButton>
    );
}

const mapStateToProps = createStructuredSelector({
    wizardPolicyIsNew: selectors.getWizardIsNew,
});

const mapDispatchToProps = {
    setWizardStage: wizardActions.setWizardStage,
};

export default connect(mapStateToProps, mapDispatchToProps)(SavePolicyButton);
