import React from 'react';
import {
    Button,
    Modal,
    WizardContextConsumer,
    WizardFooter,
    WizardStep,
} from '@patternfly/react-core';
import useModal from 'hooks/useModal';

export type ScanConfigWizardStepsProps = {
    wizardSteps: WizardStep[];
    onSave: () => void;
    isSaving: boolean;
    proceedToNextStepIfValid: (nextFunction: () => void, stepId: string) => void;
};

function ScanConfigWizardFooter({
    wizardSteps,
    onSave,
    isSaving,
    proceedToNextStepIfValid,
}: ScanConfigWizardStepsProps) {
    const { isModalOpen, openModal, closeModal } = useModal();
    const firstStepId = wizardSteps[0].id;
    const lastStepId = wizardSteps[wizardSteps.length - 1].id;

    function renderButtons(activeStepId, onNext, onBack) {
        return (
            <>
                {activeStepId !== lastStepId ? (
                    <Button
                        variant="primary"
                        type="submit"
                        onClick={() => proceedToNextStepIfValid(onNext, activeStepId)}
                    >
                        Next
                    </Button>
                ) : (
                    <Button variant="primary" type="submit" onClick={onSave} isLoading={isSaving}>
                        Create
                    </Button>
                )}
                <Button
                    variant="secondary"
                    onClick={onBack}
                    isDisabled={activeStepId === firstStepId}
                >
                    Back
                </Button>
                <Button variant="link" onClick={openModal}>
                    Cancel
                </Button>
            </>
        );
    }

    function renderModal() {
        return (
            <Modal
                variant="small"
                title="Confirm cancel"
                isOpen={isModalOpen}
                onClose={closeModal}
                actions={[
                    <Button key="confirm" variant="primary" onClick={closeModal}>
                        Confirm
                    </Button>,
                    <Button key="cancel" variant="secondary" onClick={closeModal}>
                        Cancel
                    </Button>,
                ]}
            >
                <p>
                    Are you sure you want to cancel? Any unsaved changes will be lost. You will be
                    taken back to the list of scan configurations.
                </p>
            </Modal>
        );
    }

    return (
        <WizardFooter>
            <WizardContextConsumer>
                {({ activeStep, onNext, onBack }) => (
                    <>
                        {renderButtons(activeStep.id, onNext, onBack)}
                        {renderModal()}
                    </>
                )}
            </WizardContextConsumer>
        </WizardFooter>
    );
}

export default ScanConfigWizardFooter;
