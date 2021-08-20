import React, { ReactElement } from 'react';
import { connect } from 'react-redux';
import { ArrowLeft, ArrowRight } from 'react-feather';
import { createStructuredSelector } from 'reselect';
import { formValueSelector } from 'redux-form';

import { actions as wizardActions } from 'reducers/policies/wizard';
import wizardStages from 'Containers/Policies/Wizard/wizardStages';
import PanelButton from 'Components/PanelButton';
import SavePolicyButton from '../SavePolicyButton';

type PreviewButtonsProps = {
    hasAuditLogEventSource: boolean;
    setWizardStage: (string) => void;
};

function PreviewButtons({
    hasAuditLogEventSource,
    setWizardStage,
}: PreviewButtonsProps): ReactElement {
    function goBackToEditBPL() {
        setWizardStage(wizardStages.editBPL);
    }

    function goToEnforcement() {
        setWizardStage(wizardStages.enforcement);
    }

    const skipEnforcement = hasAuditLogEventSource;

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
            {skipEnforcement ? (
                <SavePolicyButton />
            ) : (
                <PanelButton
                    icon={<ArrowRight className="h-4 w-4" />}
                    className="btn btn-base mr-2"
                    onClick={goToEnforcement}
                    tooltip="Go to next step"
                >
                    Next
                </PanelButton>
            )}
        </>
    );
}

const mapStateToProps = createStructuredSelector({
    hasAuditLogEventSource: (state) => {
        // TODO: Adding @types/redux-form causes this "state" to be an error. I didn't want to get
        // side tracked, so adding this disable for now. We should come back and properly type out the state
        // eslint-disable-next-line @typescript-eslint/ban-ts-comment
        // @ts-ignore
        const eventSourceValue = formValueSelector('policyCreationForm')(state, 'eventSource');
        return eventSourceValue === 'AUDIT_LOG_EVENT';
    },
});

const mapDispatchToProps = {
    setWizardStage: wizardActions.setWizardStage,
};

export default connect(mapStateToProps, mapDispatchToProps)(PreviewButtons);
