import React, { ReactElement } from 'react';
import {
    Breadcrumb,
    BreadcrumbItem,
    Divider,
    PageSection,
    Text,
    TextContent,
    Title,
} from '@patternfly/react-core';

import { vulnManagementReportsPath } from 'routePaths';
import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import PageTitle from 'Components/PageTitle';
import { ReportConfiguration } from 'types/report.proto';
import VulnMgmtReportForm from '../VulnMgmtReportForm';

type VulnMgmtEditReportPageProps = {
    report: ReportConfiguration;
};

function VulnMgmtEditReportPage({ report }: VulnMgmtEditReportPageProps): ReactElement {
    return (
        <>
            <PageSection variant="light">
                <PageTitle title="Vulnerability Management - Edit" />
                <Breadcrumb className="pf-u-mb-md">
                    <BreadcrumbItemLink to={vulnManagementReportsPath}>
                        Vulnerability reporting
                    </BreadcrumbItemLink>
                    <BreadcrumbItem isActive>Create a vulnerability report</BreadcrumbItem>
                </Breadcrumb>
                <TextContent>
                    <Title headingLevel="h1">Create a vulnerability report</Title>
                    <Text component="p">
                        Configure reports, define reporting scopes, and assign distribution lists to
                        report on vulnerabilities across the organization.
                    </Text>
                </TextContent>
            </PageSection>
            <Divider component="div" />
            <VulnMgmtReportForm initialValues={report} />
        </>
    );
}

export default VulnMgmtEditReportPage;
