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
import { ReportConfigurationMappedValues } from 'types/report.proto';
import VulnMgmtReportForm from './VulnMgmtReportForm';

function VulnMgmtCreateReportPage(): ReactElement {
    const emptyReportValues: ReportConfigurationMappedValues = {
        id: '',
        name: '',
        description: '',
        type: 'VULNERABILITY',
        vulnReportFiltersMappedValues: {
            fixabilityMappedValues: [],
            sinceLastReport: false,
            severities: [],
        },
        scopeId: '',
        notifierConfig: {
            emailConfig: {
                notifierId: '',
                mailingLists: [],
            },
        },
        schedule: {
            intervalType: 'WEEKLY',
            hour: 0,
            minute: 0,
            interval: {
                days: [],
            },
        },
    };

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
            <Divider component="div" />
            <VulnMgmtReportForm initialValues={emptyReportValues} />
        </>
    );
}

export default VulnMgmtCreateReportPage;
