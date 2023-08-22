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
import isEmpty from 'lodash/isEmpty';

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
    'Configure delivery destinations',
    'Review and save',
];

function EditVulnReportPage() {
    const history = useHistory();
    const { reportId } = useParams();

    const { reportConfiguration, isLoading, error } = useFetchReport(reportId);
    const formik = useReportFormValues();
    const { isSaving, saveError, saveReport } = useSaveReport({
        onCompleted: () => {
            formik.resetForm();
            history.push(vulnerabilityReportsPath);
        },
    });

    // We fetch the report configuration for the edittable report and then populate the form values
    useEffect(() => {
        if (reportConfiguration) {
            const reportFormValues = getReportFormValuesFromConfiguration(reportConfiguration);
            formik.setValues(reportFormValues);
        }
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [reportConfiguration, formik.setValues]);

    function onSave() {
        saveReport(reportId, formik.values);
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

    function onClose() {
        history.push(vulnerabilityReportsPath);
    }

    const wizardSteps = [
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
            nextButtonText: 'Save',
            isDisabled: isStepDisabled(wizardStepNames[2]),
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
                    onClose={onClose}
                    footer={
                        <ReportFormWizardFooter
                            wizardSteps={wizardSteps}
                            saveText="Save"
                            onSave={onSave}
                            isSaving={isSaving}
                            isStepDisabled={isStepDisabled}
                        />
                    }
                />
            </PageSection>
        </>
    );
}

export default EditVulnReportPage;
