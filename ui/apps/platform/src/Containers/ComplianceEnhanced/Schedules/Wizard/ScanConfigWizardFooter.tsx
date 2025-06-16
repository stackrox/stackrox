import React from 'react';
import { Button, Modal } from '@patternfly/react-core';
import { WizardContextConsumer, WizardFooter, WizardStep } from '@patternfly/react-core/deprecated';
import useModal from 'hooks/useModal';

export type ScanConfigWizardFooterProps = {
    wizardSteps: WizardStep[];
    onSave: () => void;
    isSaving: boolean;
    proceedToNextStepIfValid: (nextFunction: () => void, stepId: string) => void;
    disableClusterNext: boolean;
};

function ScanConfigWizardFooter({
    wizardSteps,
    onSave,
    isSaving,
    proceedToNextStepIfValid,
    disableClusterNext,
}: ScanConfigWizardFooterProps) {
    const { isModalOpen, openModal, closeModal } = useModal();
    const firstStepId = wizardSteps[0].id;
    const lastStepId = wizardSteps[wizardSteps.length - 1].id;

    return (
        <WizardFooter>
            <WizardContextConsumer>
                {({ activeStep, onNext, onBack, onClose }) => (
                    <>
                        {activeStep.id !== lastStepId ? (
                            <Button
                                variant="primary"
                                type="submit"
                                isDisabled={
                                    disableClusterNext && activeStep.id === wizardSteps[1].id
                                }
                                onClick={() =>
                                    proceedToNextStepIfValid(onNext, String(activeStep.id))
                                }
                            >
                                Next
                            </Button>
                        ) : (
                            <Button
                                variant="primary"
                                type="submit"
                                isDisabled={isSaving}
                                onClick={onSave}
                                isLoading={isSaving}
                            >
                                Save
                            </Button>
                        )}
                        <Button
                            variant="secondary"
                            onClick={onBack}
                            isDisabled={activeStep.id === firstStepId}
                        >
                            Back
                        </Button>
                        <Button variant="link" onClick={openModal}>
                            Cancel
                        </Button>
                        <Modal
                            variant="small"
                            title="Confirm cancel"
                            isOpen={isModalOpen}
                            onClose={closeModal}
                            actions={[
                                <Button key="confirm" variant="primary" onClick={onClose}>
                                    Confirm
                                </Button>,
                                <Button key="cancel" variant="secondary" onClick={closeModal}>
                                    Cancel
                                </Button>,
                            ]}
                        >
                            <p>
                                Are you sure you want to cancel? Any unsaved changes will be lost.
                                You will be taken back to the list of scan configurations.
                            </p>
                        </Modal>
                    </>
                )}
            </WizardContextConsumer>
        </WizardFooter>
    );
}

export default ScanConfigWizardFooter;
