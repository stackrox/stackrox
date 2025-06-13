import React from 'react';
import { useNavigate } from 'react-router-dom';
import { Button, Modal } from '@patternfly/react-core';
import {
    Wizard,
    WizardContextConsumer,
    WizardFooter,
    WizardStep,
} from '@patternfly/react-core/deprecated';
import { FormikProps } from 'formik';
import isEmpty from 'lodash/isEmpty';

import useModal from 'hooks/useModal';
import { vulnerabilityConfigurationReportsPath } from 'routePaths';

import DeliveryDestinationsForm from '../forms/DeliveryDestinationsForm';
import ReportParametersForm from '../forms/ReportParametersForm';
import ReportReviewForm from '../forms/ReportReviewForm';
import { ReportFormValues } from '../forms/useReportFormValues';

export type ReportFormWizardProps = {
    formik: FormikProps<ReportFormValues>;
    navAriaLabel: string;
    mainAriaLabel: string;
    wizardStepNames: string[];
    finalStepNextButtonText: string;
    onSave: () => void;
    isSaving: boolean;
};

function ReportFormWizard({
    formik,
    navAriaLabel,
    mainAriaLabel,
    wizardStepNames,
    finalStepNextButtonText,
    onSave,
    isSaving,
}: ReportFormWizardProps) {
    const navigate = useNavigate();

    const { isModalOpen, openModal, closeModal } = useModal();

    function onClose() {
        navigate(vulnerabilityConfigurationReportsPath);
    }

    const wizardSteps: WizardStep[] = [
        {
            name: wizardStepNames[0],
            component: <ReportParametersForm title={wizardStepNames[0]} formik={formik} />,
        },
        {
            name: wizardStepNames[1],
            component: <DeliveryDestinationsForm title={wizardStepNames[1]} formik={formik} />,
            isDisabled: isStepDisabled(wizardStepNames[1]),
        },
        {
            name: wizardStepNames[2],
            component: <ReportReviewForm title={wizardStepNames[2]} formValues={formik.values} />,
            nextButtonText: finalStepNextButtonText,
            isDisabled: isStepDisabled(wizardStepNames[2]),
        },
    ];

    function isStepDisabled(stepName: string | undefined): boolean {
        if (stepName === wizardStepNames[0]) {
            return false;
        }
        if (stepName === wizardStepNames[1]) {
            return !isEmpty(formik.errors.reportParameters);
        }
        if (stepName === wizardStepNames[2]) {
            return (
                !isEmpty(formik.errors.reportParameters) ||
                !isEmpty(formik.errors.deliveryDestinations) ||
                !isEmpty(formik.errors.schedule)
            );
        }
        return false;
    }

    return (
        <Wizard
            navAriaLabel={navAriaLabel}
            mainAriaLabel={mainAriaLabel}
            hasNoBodyPadding
            steps={wizardSteps}
            onSave={onSave}
            onClose={onClose}
            footer={
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
                                            {finalStepNextButtonText}
                                        </Button>
                                    )}
                                    <Button
                                        variant="secondary"
                                        onClick={onBack}
                                        isDisabled={activeStep.name === firstStepName}
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
                                            <Button
                                                key="confirm"
                                                variant="primary"
                                                onClick={onClose}
                                            >
                                                Confirm
                                            </Button>,
                                            <Button
                                                key="cancel"
                                                variant="secondary"
                                                onClick={closeModal}
                                            >
                                                Cancel
                                            </Button>,
                                        ]}
                                    >
                                        <p>
                                            Are you sure you want to cancel? Any unsaved changes
                                            will be lost. You will be taken back to the list of
                                            reports.
                                        </p>
                                    </Modal>
                                </>
                            );
                        }}
                    </WizardContextConsumer>
                </WizardFooter>
            }
        />
    );
}

export default ReportFormWizard;
