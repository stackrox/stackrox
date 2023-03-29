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
import { ReportScope } from 'hooks/useFetchReport';
import { ReportConfiguration } from 'types/report.proto';
import VulnMgmtReportForm from '../VulnMgmtReportForm';

type VulnMgmtEditReportPageProps = {
    report: ReportConfiguration;
    reportScope: ReportScope | null;
    refreshQuery: () => void;
};

function VulnMgmtEditReportPage({
    report,
    reportScope,
    refreshQuery,
}: VulnMgmtEditReportPageProps): ReactElement {
    const { id, name } = report;

    return (
        <>
            <PageSection variant="light">
                <PageTitle title="Vulnerability Management - Edit" />
                <Breadcrumb className="pf-u-mb-md">
                    <BreadcrumbItemLink to={vulnManagementReportsPath}>
                        Vulnerability reporting
                    </BreadcrumbItemLink>
                    <BreadcrumbItemLink to={`${vulnManagementReportsPath}/${id}`}>
                        {name}
                    </BreadcrumbItemLink>
                    <BreadcrumbItem isActive>Edit</BreadcrumbItem>
                </Breadcrumb>
                <TextContent>
                    <Title headingLevel="h1">Edit vulnerability report</Title>
                    <Text component="p">
                        Configure reports, define reporting scopes, and assign distribution lists to
                        report on vulnerabilities across the organization.
                    </Text>
                </TextContent>
            </PageSection>
            <Divider component="div" />
            <VulnMgmtReportForm
                initialValues={report}
                initialReportScope={reportScope}
                refreshQuery={refreshQuery}
            />
        </>
    );
}

export default VulnMgmtEditReportPage;
