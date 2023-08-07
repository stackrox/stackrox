import React from 'react';
import { Button, WizardContextConsumer, WizardFooter, WizardStep } from '@patternfly/react-core';

export type ReportFormWizardStepsProps = {
    wizardSteps: WizardStep[];
    saveText: string;
    onSave: () => void;
    isSaving: boolean;
    isStepDisabled: (stepName: string) => boolean;
};

function ReportFormWizardFooter({
    wizardSteps,
    saveText,
    onSave,
    isSaving,
    isStepDisabled,
}: ReportFormWizardStepsProps) {
    return (
        <WizardFooter>
            <WizardContextConsumer>
                {({ activeStep, onNext, onBack, onClose }) => {
                    const firstStepName = wizardSteps[0].name;
                    const lastStepName = wizardSteps[wizardSteps.length - 1].name;
                    const activeStepIndex = wizardSteps.findIndex(
                        (wizardStep) => wizardStep.name === activeStep.name
                    );
                    const nextStepName =
                        activeStepIndex === wizardSteps.length - 1
                            ? undefined
                            : wizardSteps[activeStepIndex + 1].name;
                    const isNextDisabled = isStepDisabled(nextStepName as string);

                    return (
                        <>
                            {activeStep.name !== lastStepName ? (
                                <Button
                                    variant="primary"
                                    type="submit"
                                    onClick={onNext}
                                    isDisabled={isNextDisabled}
                                >
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
