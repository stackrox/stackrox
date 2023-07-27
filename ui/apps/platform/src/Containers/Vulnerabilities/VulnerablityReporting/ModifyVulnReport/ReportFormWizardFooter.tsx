import React from 'react';
import { Button, WizardContextConsumer, WizardFooter, WizardStep } from '@patternfly/react-core';

export type ReportFormWizardStepsProps = {
    wizardSteps: WizardStep[];
    saveText: string;
    onSave: () => void;
    isSaving: boolean;
};

function ReportFormWizardFooter({
    wizardSteps,
    saveText,
    onSave,
    isSaving,
}: ReportFormWizardStepsProps) {
    return (
        <WizardFooter>
            <WizardContextConsumer>
                {({ activeStep, onNext, onBack, onClose }) => {
                    const firstStepName = wizardSteps[0].name;
                    const lastStepName = wizardSteps[wizardSteps.length - 1].name;
                    return (
                        <>
                            {activeStep.name !== lastStepName ? (
                                <Button variant="primary" type="submit" onClick={onNext}>
                                    Next
                                </Button>
                            ) : (
                                <Button
                                    variant="primary"
                                    type="submit"
                                    onClick={onSave}
                                    isLoading={isSaving}
                                >
                                    {saveText}
                                </Button>
                            )}
                            <Button
                                variant="secondary"
                                onClick={onBack}
                                isDisabled={activeStep.name === firstStepName}
                            >
                                Back
                            </Button>
                            <Button variant="link" onClick={onClose}>
                                Cancel
                            </Button>
                        </>
                    );
                }}
            </WizardContextConsumer>
        </WizardFooter>
    );
}

export default ReportFormWizardFooter;
