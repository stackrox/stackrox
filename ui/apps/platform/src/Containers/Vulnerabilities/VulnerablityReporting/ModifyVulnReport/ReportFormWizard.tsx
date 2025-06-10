import React from 'react';
import { useNavigate } from 'react-router-dom';
import { Wizard, WizardStep } from '@patternfly/react-core/deprecated';
import { FormikProps } from 'formik';
import isEmpty from 'lodash/isEmpty';

import { vulnerabilityConfigurationReportsPath } from 'routePaths';

import DeliveryDestinationsForm from '../forms/DeliveryDestinationsForm';
import ReportParametersForm from '../forms/ReportParametersForm';
import ReportReviewForm from '../forms/ReportReviewForm';
import ReportFormWizardFooter from './ReportFormWizardFooter';
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
                <ReportFormWizardFooter
                    wizardSteps={wizardSteps}
                    saveText={finalStepNextButtonText}
                    onSave={onSave}
                    isSaving={isSaving}
                    isStepDisabled={isStepDisabled}
                />
            }
        />
    );
}

export default ReportFormWizard;
