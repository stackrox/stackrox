import React from 'react';
import { useHistory } from 'react-router-dom';
import {
    PageSection,
    Title,
    Divider,
    Flex,
    FlexItem,
    Breadcrumb,
    BreadcrumbItem,
    Wizard,
    WizardStep,
} from '@patternfly/react-core';
import isEmpty from 'lodash/isEmpty';

import { vulnerabilityReportsPath } from 'routePaths';
import useReportFormValues from 'Containers/Vulnerabilities/VulnerablityReporting/forms/useReportFormValues';

import PageTitle from 'Components/PageTitle';
import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import ReportParametersForm from 'Containers/Vulnerabilities/VulnerablityReporting/forms/ReportParametersForm';
import DeliveryDestinationsForm from 'Containers/Vulnerabilities/VulnerablityReporting/forms/DeliveryDestinationsForm';
import ReportReviewForm from 'Containers/Vulnerabilities/VulnerablityReporting/forms/ReportReviewForm';
import useCreateReport from '../api/useCreateReport';
import ReportFormWizardFooter from './ReportFormWizardFooter';
import ReportFormErrorAlert from './ReportFormErrorAlert';

const wizardStepNames = [
    'Configure report parameters',
    'Configure delivery destinations',
    'Review and create',
];

function CreateVulnReportPage() {
    const history = useHistory();

    const formik = useReportFormValues();
    const { isLoading, error, createReport } = useCreateReport({
        onCompleted: () => {
            formik.resetForm();
            history.push(vulnerabilityReportsPath);
        },
    });

    function onCreate() {
        createReport(formik.values);
    }

    // @TODO: This is reused in the Edit and Clone components so we can try to refactor this soon
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
            nextButtonText: 'Create',
            isDisabled: isStepDisabled(wizardStepNames[2]),
        },
    ];

    return (
        <>
            <PageTitle title="Create vulnerability report" />
            <ReportFormErrorAlert error={error} />
            <PageSection variant="light" className="pf-u-py-md">
                <Breadcrumb>
                    <BreadcrumbItemLink to={vulnerabilityReportsPath}>
                        Vulnerability reporting
                    </BreadcrumbItemLink>
                    <BreadcrumbItem isActive>Create report</BreadcrumbItem>
                </Breadcrumb>
            </PageSection>
            <Divider component="div" />
            <PageSection variant="light" padding={{ default: 'noPadding' }}>
                <Flex direction={{ default: 'column' }} className="pf-u-py-lg pf-u-px-lg">
                    <FlexItem>
                        <Title headingLevel="h1">Create report</Title>
                    </FlexItem>
                    <FlexItem>
                        Configure reports, define report scopes, and assign delivery destinations to
                        report on vulnerabilities across the organization.
                    </FlexItem>
                </Flex>
            </PageSection>
            <Divider component="div" />
            <PageSection padding={{ default: 'noPadding' }} isCenterAligned>
                <Wizard
                    navAriaLabel="Report creation steps"
                    mainAriaLabel="Report creation content"
                    hasNoBodyPadding
                    steps={wizardSteps}
                    onSave={onCreate}
                    footer={
                        <ReportFormWizardFooter
                            wizardSteps={wizardSteps}
                            saveText="Create"
                            onSave={onCreate}
                            isSaving={isLoading}
                            isStepDisabled={isStepDisabled}
                        />
                    }
                />
            </PageSection>
        </>
    );
}

export default CreateVulnReportPage;
