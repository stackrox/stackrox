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
import useSaveReport from 'Containers/Vulnerabilities/VulnerablityReporting/api/useSaveReport';
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
    'Review and save',
];

function EditVulnReportPage() {
    const history = useHistory();
    const { reportId } = useParams();

    const { reportConfiguration, isLoading, error } = useFetchReport(reportId);
    const { formValues, setFormValues, setFormFieldValue, clearFormValues } = useReportFormValues();
    const { data, isSaving, saveError, saveReport } = useSaveReport();

    // When report is created, navigate to the vuln reports page
    // @TODO: We want to change this in the future to navigate to the read-only report page to verify the details
    useEffect(() => {
        if (data) {
            clearFormValues();
            history.push(vulnerabilityReportsPath);
        }
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [data]);

    // We fetch the report configuration for the edittable report and then populate the form values
    useEffect(() => {
        if (reportConfiguration) {
            const reportFormValues = getReportFormValuesFromConfiguration(reportConfiguration);
            setFormValues(reportFormValues);
        }
    }, [reportConfiguration, setFormValues]);

    function onSave() {
        saveReport(reportId, formValues);
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
            nextButtonText: 'Save',
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
            <ReportFormErrorAlert error={saveError} />
            <PageSection variant="light" className="pf-u-py-md">
                <Breadcrumb>
                    <BreadcrumbItemLink to={vulnerabilityReportsPath}>
                        Vulnerability reporting
                    </BreadcrumbItemLink>
                    <BreadcrumbItem isActive>Edit report</BreadcrumbItem>
                </Breadcrumb>
            </PageSection>
            <Divider component="div" />
            <PageSection variant="light" padding={{ default: 'noPadding' }}>
                <Flex direction={{ default: 'column' }} className="pf-u-py-lg pf-u-px-lg">
                    <FlexItem>
                        <Title headingLevel="h1">Edit report</Title>
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
                    navAriaLabel="Report edit steps"
                    mainAriaLabel="Report edit content"
                    hasNoBodyPadding
                    steps={wizardSteps}
                    onSave={onSave}
                    footer={
                        <ReportFormWizardFooter
                            wizardSteps={wizardSteps}
                            saveText="Save"
                            onSave={onSave}
                            isSaving={isSaving}
                        />
                    }
                />
            </PageSection>
        </>
    );
}

export default EditVulnReportPage;
