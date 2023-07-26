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
} from '@patternfly/react-core';

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
    'Configure delivery destinations (Optional)',
    'Review and create',
];

function CreateVulnReportPage() {
    const history = useHistory();

    const { formValues, setFormFieldValue, clearFormValues } = useReportFormValues();
    const { isLoading, error, createReport } = useCreateReport({
        onCompleted: () => {
            clearFormValues();
            history.push(vulnerabilityReportsPath);
        },
    });

    function onCreate() {
        createReport(formValues);
    }

    const wizardSteps = [
        {
            name: wizardStepNames[0],
            component: (
                <ReportParametersForm
                    title={wizardStepNames[0]}
                    formValues={formValues}
                    setFormFieldValue={setFormFieldValue}
                />
            ),
        },
        {
            name: wizardStepNames[1],
            component: (
                <DeliveryDestinationsForm
                    title={wizardStepNames[1]}
                    formValues={formValues}
                    setFormFieldValue={setFormFieldValue}
                />
            ),
        },
        {
            name: wizardStepNames[2],
            component: <ReportReviewForm title={wizardStepNames[2]} formValues={formValues} />,
            nextButtonText: 'Create',
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
                        />
                    }
                />
            </PageSection>
        </>
    );
}

export default CreateVulnReportPage;
