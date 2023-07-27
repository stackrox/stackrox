import React, { useEffect } from 'react';
import { useHistory, useParams } from 'react-router-dom';
import {
    PageSection,
    Title,
    Divider,
    Flex,
    FlexItem,
    Breadcrumb,
    BreadcrumbItem,
    Wizard,
    Bullseye,
    Spinner,
} from '@patternfly/react-core';

import { vulnerabilityReportsPath } from 'routePaths';
import useReportFormValues from 'Containers/Vulnerabilities/VulnerablityReporting/forms/useReportFormValues';
import useCreateReport from 'Containers/Vulnerabilities/VulnerablityReporting/api/useCreateReport';
import useFetchReport from 'Containers/Vulnerabilities/VulnerablityReporting/api/useFetchReport';

import PageTitle from 'Components/PageTitle';
import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import ReportParametersForm from 'Containers/Vulnerabilities/VulnerablityReporting/forms/ReportParametersForm';
import DeliveryDestinationsForm from 'Containers/Vulnerabilities/VulnerablityReporting/forms/DeliveryDestinationsForm';
import ReportReviewForm from 'Containers/Vulnerabilities/VulnerablityReporting/forms/ReportReviewForm';
import NotFoundMessage from 'Components/NotFoundMessage/NotFoundMessage';
import { getReportFormValuesFromConfiguration } from '../utils';
import ReportFormErrorAlert from './ReportFormErrorAlert';
import ReportFormWizardFooter from './ReportFormWizardFooter';

const wizardStepNames = [
    'Configure report parameters',
    'Configure delivery destinations (Optional)',
    'Review and create',
];

function CloneVulnReportPage() {
    const history = useHistory();
    const { reportId } = useParams();

    const { reportConfiguration, isLoading, error } = useFetchReport(reportId);
    const { formValues, setFormValues, setFormFieldValue, clearFormValues } = useReportFormValues();
    const {
        isLoading: isCreating,
        error: createError,
        createReport,
    } = useCreateReport({
        onCompleted: () => {
            clearFormValues();
            history.push(vulnerabilityReportsPath);
        },
    });

    // We fetch the report configuration for the edittable report and then populate the form values
    useEffect(() => {
        if (reportConfiguration) {
            const reportFormValues = getReportFormValuesFromConfiguration(reportConfiguration);
            // We need to clear the reportId and modify the name
            reportFormValues.reportId = '';
            reportFormValues.reportParameters.reportName = `${reportFormValues.reportParameters.reportName} (copy)`;
            setFormValues(reportFormValues);
        }
    }, [reportConfiguration, setFormValues]);

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

    if (error) {
        return (
            <NotFoundMessage
                title="Error fetching the report configuration"
                message={error}
                actionText="Go to reports"
                url={vulnerabilityReportsPath}
            />
        );
    }

    if (isLoading) {
        return (
            <Bullseye>
                <Spinner isSVG />
            </Bullseye>
        );
    }

    return (
        <>
            <PageTitle title="Create vulnerability report" />
            <ReportFormErrorAlert error={createError} />
            <PageSection variant="light" className="pf-u-py-md">
                <Breadcrumb>
                    <BreadcrumbItemLink to={vulnerabilityReportsPath}>
                        Vulnerability reporting
                    </BreadcrumbItemLink>
                    <BreadcrumbItem isActive>Clone report</BreadcrumbItem>
                </Breadcrumb>
            </PageSection>
            <Divider component="div" />
            <PageSection variant="light" padding={{ default: 'noPadding' }}>
                <Flex direction={{ default: 'column' }} className="pf-u-py-lg pf-u-px-lg">
                    <FlexItem>
                        <Title headingLevel="h1">Clone report</Title>
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
                    navAriaLabel="Report clone steps"
                    mainAriaLabel="Report clone content"
                    hasNoBodyPadding
                    steps={wizardSteps}
                    onSave={onCreate}
                    footer={
                        <ReportFormWizardFooter
                            wizardSteps={wizardSteps}
                            saveText="Create"
                            onSave={onCreate}
                            isSaving={isCreating}
                        />
                    }
                />
            </PageSection>
        </>
    );
}

export default CloneVulnReportPage;
