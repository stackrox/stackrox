import React from 'react';
import { useNavigate } from 'react-router-dom-v5-compat';
import { Button, Modal, Wizard, WizardStep } from '@patternfly/react-core';
import { FormikProps } from 'formik';
import isEmpty from 'lodash/isEmpty';

import useModal from 'hooks/useModal';
import { vulnerabilityConfigurationReportsPath } from 'routePaths';

import DeliveryDestinationsForm from '../forms/DeliveryDestinationsForm';
import ReportParametersForm from '../forms/ReportParametersForm';
import ReportReviewForm from '../forms/ReportReviewForm';
import { ReportFormValues } from '../forms/useReportFormValues';

const wizardStepNames = [
    'Configure report parameters',
    'Configure delivery destinations',
    'Review',
];

export type ReportFormWizardProps = {
    formik: FormikProps<ReportFormValues>;
    onSave: () => void;
    isSaving: boolean;
};

function ReportFormWizard({ formik, onSave, isSaving }: ReportFormWizardProps) {
    const navigate = useNavigate();

    const { isModalOpen, openModal, closeModal } = useModal();

    function onClose() {
        navigate(vulnerabilityConfigurationReportsPath);
    }

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
        <>
            <Wizard navAriaLabel="Vulnerability report configuration steps" onSave={onSave}>
                <WizardStep
                    name={wizardStepNames[0]}
                    id={wizardStepNames[0]}
                    key={wizardStepNames[0]}
                    body={{ hasNoPadding: true }}
                    footer={{
                        isNextDisabled: isStepDisabled(wizardStepNames[1]),
                        onClose: openModal,
                    }}
                >
                    <ReportParametersForm title={wizardStepNames[0]} formik={formik} />
                </WizardStep>
                <WizardStep
                    name={wizardStepNames[1]}
                    id={wizardStepNames[1]}
                    key={wizardStepNames[1]}
                    body={{ hasNoPadding: true }}
                    isDisabled={isStepDisabled(wizardStepNames[1])}
                    footer={{
                        isNextDisabled: isStepDisabled(wizardStepNames[2]),
                        onClose: openModal,
                    }}
                >
                    <DeliveryDestinationsForm title={wizardStepNames[1]} formik={formik} />
                </WizardStep>
                <WizardStep
                    name={wizardStepNames[2]}
                    id={wizardStepNames[2]}
                    key={wizardStepNames[2]}
                    body={{ hasNoPadding: true }}
                    isDisabled={isStepDisabled(wizardStepNames[2])}
                    footer={{
                        nextButtonProps: { isLoading: isSaving },
                        nextButtonText: 'Save',
                        onClose: openModal,
                    }}
                >
                    <ReportReviewForm title={wizardStepNames[2]} formValues={formik.values} />
                </WizardStep>
            </Wizard>
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
                    Are you sure you want to cancel? Any unsaved changes will be lost. You will be
                    taken back to the list of reports.
                </p>
            </Modal>
        </>
    );
}

export default ReportFormWizard;
