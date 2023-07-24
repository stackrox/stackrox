import React, { useEffect } from 'react';
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
    Alert,
    AlertVariant,
    Button,
    WizardFooter,
    WizardContextConsumer,
} from '@patternfly/react-core';

import { vulnerabilityReportsPath } from 'routePaths';
import useReportFormValues from 'Containers/Vulnerabilities/VulnerablityReporting/forms/useReportFormValues';

import PageTitle from 'Components/PageTitle';
import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import ReportParametersForm from 'Containers/Vulnerabilities/VulnerablityReporting/forms/ReportParametersForm';
import DeliveryDestinationsForm from 'Containers/Vulnerabilities/VulnerablityReporting/forms/DeliveryDestinationsForm';
import ReportReviewForm from 'Containers/Vulnerabilities/VulnerablityReporting/forms/ReportReviewForm';
import useCreateReport from '../api/useCreateReport';

const wizardStepNames = [
    'Configure report parameters',
    'Configure delivery destinations (Optional)',
    'Review and create',
];

function VulnReportsPage() {
    const alertRef = React.useRef<HTMLInputElement>(null);
    const history = useHistory();

    const { formValues, setFormFieldValue, clearFormValues } = useReportFormValues();
    const { data, isLoading, error, createReport } = useCreateReport();

    // When report is created, navigate to the vuln reports page
    // @TODO: We want to change this in the future to navigate to the read-only report page to verify the details
    useEffect(() => {
        if (data) {
            clearFormValues();
            history.push(vulnerabilityReportsPath);
        }
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [data]);

    // When an error occurs, scroll the message into view
    useEffect(() => {
        if (error && alertRef.current) {
            alertRef.current?.scrollIntoView({
                behavior: 'smooth',
            });
        }
    }, [error]);

    function onSave() {
        createReport(formValues);
    }

    return (
        <>
            <PageTitle title="Create vulnerability report" />
            {error && (
                <div ref={alertRef}>
                    <Alert
                        isInline
                        variant={AlertVariant.danger}
                        title={error}
                        className="pf-u-mb-sm"
                    />
                </div>
            )}
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
                    // @TODO: Make the navAriaLabel dynamic based on whether you're creating, editing, or cloning
                    navAriaLabel="Report creation steps"
                    // @TODO: Make the mainAriaLabel dynamic based on whether you're creating, editing, or cloning
                    mainAriaLabel="Report creation content"
                    hasNoBodyPadding
                    steps={[
                        {
                            name: wizardStepNames[0],
                            component: (
                                <ReportParametersForm
                                    formValues={formValues}
                                    setFormFieldValue={setFormFieldValue}
                                />
                            ),
                        },
                        {
                            name: wizardStepNames[1],
                            component: (
                                <DeliveryDestinationsForm
                                    formValues={formValues}
                                    setFormFieldValue={setFormFieldValue}
                                />
                            ),
                        },
                        {
                            name: wizardStepNames[2],
                            component: <ReportReviewForm formValues={formValues} />,
                            nextButtonText: 'Create',
                        },
                    ]}
                    onSave={onSave}
                    footer={
                        <WizardFooter>
                            <WizardContextConsumer>
                                {({ activeStep, onNext, onBack, onClose }) => {
                                    const lastStepName =
                                        wizardStepNames[wizardStepNames.length - 1];
                                    return (
                                        <>
                                            {activeStep.name !== lastStepName ? (
                                                <Button
                                                    variant="primary"
                                                    type="submit"
                                                    onClick={onNext}
                                                >
                                                    Next
                                                </Button>
                                            ) : (
                                                <Button
                                                    variant="primary"
                                                    type="submit"
                                                    onClick={onSave}
                                                    isLoading={isLoading}
                                                >
                                                    Create
                                                </Button>
                                            )}
                                            <Button
                                                variant="secondary"
                                                onClick={onBack}
                                                isDisabled={activeStep.name !== lastStepName}
                                            >
                                                Back
                                            </Button>
                                            <Button variant="link" onClick={onClose}>
                                                Cancel
                                            </Button>
                                        </>
                                    );
                                }}
                            </WizardContextConsumer>
                        </WizardFooter>
                    }
                />
            </PageSection>
        </>
    );
}

export default VulnReportsPage;
