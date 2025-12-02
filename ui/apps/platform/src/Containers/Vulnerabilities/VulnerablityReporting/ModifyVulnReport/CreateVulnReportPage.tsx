import { useNavigate } from 'react-router-dom-v5-compat';
import {
    Breadcrumb,
    BreadcrumbItem,
    Divider,
    Flex,
    FlexItem,
    PageSection,
    Title,
} from '@patternfly/react-core';

import { vulnerabilityConfigurationReportsPath } from 'routePaths';
import PageTitle from 'Components/PageTitle';
import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';

import useCreateReport from '../api/useCreateReport';
import useReportFormValues from '../forms/useReportFormValues';
import ReportFormErrorAlert from './ReportFormErrorAlert';
import ReportFormWizard from './ReportFormWizard';

function CreateVulnReportPage() {
    const navigate = useNavigate();

    const formik = useReportFormValues();
    const { isLoading, error, createReport } = useCreateReport({
        onCompleted: () => {
            formik.resetForm();
            navigate(vulnerabilityConfigurationReportsPath);
        },
    });

    function onCreate() {
        createReport(formik.values);
    }

    return (
        <>
            <PageTitle title="Create vulnerability report" />
            <ReportFormErrorAlert error={error} />
            <PageSection hasBodyWrapper={false} className="pf-v6-u-py-md">
                <Breadcrumb>
                    <BreadcrumbItemLink to={vulnerabilityConfigurationReportsPath}>
                        Vulnerability reporting
                    </BreadcrumbItemLink>
                    <BreadcrumbItem isActive>Create report</BreadcrumbItem>
                </Breadcrumb>
            </PageSection>
            <Divider component="div" />
            <PageSection hasBodyWrapper={false} padding={{ default: 'noPadding' }}>
                <Flex direction={{ default: 'column' }} className="pf-v6-u-py-lg pf-v6-u-px-lg">
                    <FlexItem>
                        <Title headingLevel="h1">Create report</Title>
                    </FlexItem>
                    <FlexItem>
                        Configure reports, define collections, and assign delivery destinations to
                        report on vulnerabilities across the organization.
                    </FlexItem>
                </Flex>
            </PageSection>
            <Divider component="div" />
            <PageSection hasBodyWrapper={false} padding={{ default: 'noPadding' }} isCenterAligned>
                <ReportFormWizard formik={formik} onSave={onCreate} isSaving={isLoading} />
            </PageSection>
        </>
    );
}

export default CreateVulnReportPage;
