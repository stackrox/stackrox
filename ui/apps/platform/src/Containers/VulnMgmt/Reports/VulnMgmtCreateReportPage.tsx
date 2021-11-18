import React, { ReactElement } from 'react';
import {
    Breadcrumb,
    BreadcrumbItem,
    Text,
    TextContent,
    Title,
    PageSection,
} from '@patternfly/react-core';

import { vulnManagementReportsPath } from 'routePaths';
import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';

function VulnMgmtCreateReportPage(): ReactElement {
    return (
        <>
            <PageSection variant="light">
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
            <PageSection variant="light" isFilled>
                new report form goes here
            </PageSection>
        </>
    );
}

export default VulnMgmtCreateReportPage;
