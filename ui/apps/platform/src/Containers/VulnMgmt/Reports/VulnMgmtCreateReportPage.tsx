import React, { ReactElement } from 'react';
import {
    Breadcrumb,
    BreadcrumbItem,
    Divider,
    PageSection,
    PageSectionVariants,
    Text,
    TextContent,
    Title,
} from '@patternfly/react-core';

import { vulnManagementReportsPath } from 'routePaths';
import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import PageTitle from 'Components/PageTitle';
import VulnMgmtReportForm from './VulnMgmtReportForm';
import { emptyReportValues } from './VulnMgmtReport.utils';

function VulnMgmtCreateReportPage(): ReactElement {
    return (
        <>
            <PageSection variant={PageSectionVariants.light}>
                <PageTitle title="Vulnerability Management - Create report" />
                <Breadcrumb className="pf-u-mb-md">
                    <BreadcrumbItemLink to={vulnManagementReportsPath}>
                        Vulnerability reporting
                    </BreadcrumbItemLink>
                    <BreadcrumbItem isActive>Create an image vulnerability report</BreadcrumbItem>
                </Breadcrumb>
                <TextContent>
                    <Title headingLevel="h1">Create an image vulnerability report</Title>
                    <Text component="p">
                        Configure reports, define reporting scopes, and assign distribution lists to
                        report on vulnerabilities across the organization.
                    </Text>
                </TextContent>
            </PageSection>
            <Divider component="div" />
            <VulnMgmtReportForm initialValues={emptyReportValues} initialReportScope={null} />
        </>
    );
}

export default VulnMgmtCreateReportPage;
