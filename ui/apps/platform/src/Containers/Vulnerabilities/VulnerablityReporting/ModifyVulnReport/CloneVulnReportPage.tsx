import React, { useEffect } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import {
    PageSection,
    Title,
    Divider,
    Flex,
    FlexItem,
    Breadcrumb,
    BreadcrumbItem,
    Bullseye,
    Spinner,
} from '@patternfly/react-core';

import { vulnerabilityConfigurationReportsPath } from 'routePaths';
import useReportFormValues from 'Containers/Vulnerabilities/VulnerablityReporting/forms/useReportFormValues';
import useCreateReport from 'Containers/Vulnerabilities/VulnerablityReporting/api/useCreateReport';
import useFetchReport from 'Containers/Vulnerabilities/VulnerablityReporting/api/useFetchReport';

import PageTitle from 'Components/PageTitle';
import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import NotFoundMessage from 'Components/NotFoundMessage/NotFoundMessage';
import { getReportFormValuesFromConfiguration } from '../utils';
import ReportFormErrorAlert from './ReportFormErrorAlert';
import ReportFormWizard from './ReportFormWizard';

const wizardStepNames = [
    'Configure report parameters',
    'Configure delivery destinations',
    'Review and create',
];

function CloneVulnReportPage() {
    const navigate = useNavigate();
    const { reportId } = useParams() as { reportId: string };

    const { reportConfiguration, isLoading, error } = useFetchReport(reportId);
    const formik = useReportFormValues();
    const {
        isLoading: isCreating,
        error: createError,
        createReport,
    } = useCreateReport({
        onCompleted: () => {
            formik.resetForm();
            navigate(vulnerabilityConfigurationReportsPath);
        },
    });

    // We fetch the report configuration for the edittable report and then populate the form values
    useEffect(() => {
        if (reportConfiguration) {
            const reportFormValues = getReportFormValuesFromConfiguration(reportConfiguration);
            // We need to clear the reportId and modify the name
            reportFormValues.reportId = '';
            reportFormValues.reportParameters.reportName = `${reportFormValues.reportParameters.reportName} (copy)`;
            formik.setValues(reportFormValues);
        }
    }, [reportConfiguration, formik.setValues]);

    function onCreate() {
        createReport(formik.values);
    }

    if (error) {
        return (
            <NotFoundMessage
                title="Error fetching the report configuration"
                message={error}
                actionText="Go to reports"
                url={vulnerabilityConfigurationReportsPath}
            />
        );
    }

    if (isLoading) {
        return (
            <Bullseye>
                <Spinner />
            </Bullseye>
        );
    }

    return (
        <>
            <PageTitle title="Create vulnerability report" />
            <ReportFormErrorAlert error={createError} />
            <PageSection variant="light" className="pf-v5-u-py-md">
                <Breadcrumb>
                    <BreadcrumbItemLink to={vulnerabilityConfigurationReportsPath}>
                        Vulnerability reporting
                    </BreadcrumbItemLink>
                    <BreadcrumbItem isActive>Clone report</BreadcrumbItem>
                </Breadcrumb>
            </PageSection>
            <Divider component="div" />
            <PageSection variant="light" padding={{ default: 'noPadding' }}>
                <Flex direction={{ default: 'column' }} className="pf-v5-u-py-lg pf-v5-u-px-lg">
                    <FlexItem>
                        <Title headingLevel="h1">Clone report</Title>
                    </FlexItem>
                    <FlexItem>
                        Configure reports, define collections, and assign delivery destinations to
                        report on vulnerabilities across the organization.
                    </FlexItem>
                </Flex>
            </PageSection>
            <Divider component="div" />
            <PageSection padding={{ default: 'noPadding' }} isCenterAligned>
                <ReportFormWizard
                    formik={formik}
                    navAriaLabel="Report clone steps"
                    mainAriaLabel="Report clone content"
                    wizardStepNames={wizardStepNames}
                    finalStepNextButtonText={'Create'}
                    onSave={onCreate}
                    isSaving={isCreating}
                />
            </PageSection>
        </>
    );
}

export default CloneVulnReportPage;
