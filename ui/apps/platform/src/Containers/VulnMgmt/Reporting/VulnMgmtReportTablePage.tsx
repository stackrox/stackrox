/* eslint-disable @typescript-eslint/no-unused-vars */
import React, { ReactElement } from 'react';
import { PageSection, PageSectionVariants, Text, TextContent, Title } from '@patternfly/react-core';

import ACSEmptyState from 'Components/ACSEmptyState';
import { ReportConfiguration } from 'types/report.proto';
import VulnMgmtReportTablePanel from './VulnMgmtReportTablePanel';
import VulnMgmtReportTableColumnDescriptor from './VulnMgmtReportTableColumnDescriptor';

function ReportTablePage(): ReactElement {
    const columns = VulnMgmtReportTableColumnDescriptor;
    const reports: ReportConfiguration[] = [
        {
            id: 'fcba6900-d066-4baf-b141-c3676591cf9a',
            name: 'snowman-reporting',
            description: 'This report checks for fixable and unfixable cves on cluster "jon-snow"',
            type: 'VULNERABILITY',
            scopeId: 'fb03376c-af92-460b-bce5-54e4ee543dd4',
            filter: {
                fixability: 'BOTH',
                sinceLastReport: false,
                severities: ['IMPORTANT_VULNERABILITY_SEVERITY', 'CRITICAL_VULNERABILITY_SEVERITY'],
            },
            notifierConfig: {
                emailConfig: {
                    notifierId: 'dbfd901f-e075-4d1d-9d2a-96b0f3280b1a',
                    mailingLists: ['norville.rogers@mysterinc.com', 'fred.jones@mysteryinc.com'],
                },
            },
            schedule: {
                intervalType: 'WEEKLY',
                hour: 1,
                minute: 0,
                interval: { day: 1 },
            },
            runStatus: {
                reportStatus: 'SUCCESS',
                lastTimeRun: '2021-11-08T20:10:22z',
                errorMsg: '',
            },
        },
        {
            id: '934e7835-1ad9-4eda-a89d-be23313a46b1',
            name: 'fireman-reporting',
            description: 'This report checks for new fixable cves on cluster "flameon"',
            type: 'VULNERABILITY',
            scopeId: '395160a8-add0-48a0-a397-2801d586555d',
            filter: {
                fixability: 'FIXABLE',
                sinceLastReport: false,
                severities: ['CRITICAL_VULNERABILITY_SEVERITY'],
            },
            notifierConfig: {
                emailConfig: {
                    notifierId: 'e8a9a12f-5438-4c09-be03-bad13c80b6eb',
                    mailingLists: ['velma.dinkley@mysterinc.com', 'scooby.doo@mysteryinc.com'],
                },
            },
            schedule: {
                intervalType: 'DAILY',
                hour: 1,
                minute: 0,
                interval: { day: 1 },
            },
            runStatus: {
                reportStatus: 'FAILURE',
                lastTimeRun: '2021-11-09T20:10:22z',
                errorMsg: 'Error',
            },
        },
    ];

    return (
        <>
            <PageSection variant={PageSectionVariants.light}>
                <TextContent>
                    <Title headingLevel="h1">Vulnerability reporting</Title>
                    <Text component="p">
                        Configure reports, define resource scopes, and assign distribution lists to
                        report on vulnerabilities across the organization.
                    </Text>
                </TextContent>
            </PageSection>
            <PageSection variant={PageSectionVariants.light}>
                {reports.length > 0 ? (
                    <VulnMgmtReportTablePanel
                        reports={reports}
                        reportCount={0}
                        currentPage={0}
                        setCurrentPage={function (page: number): void {
                            throw new Error('Function not implemented.');
                        }}
                        perPage={0}
                        setPerPage={function (perPage: number): void {
                            throw new Error('Function not implemented.');
                        }}
                        activeSortIndex={0}
                        setActiveSortIndex={function (idx: number): void {
                            throw new Error('Function not implemented.');
                        }}
                        activeSortDirection="desc"
                        setActiveSortDirection={function (dir: string): void {
                            throw new Error('Function not implemented.');
                        }}
                        columns={columns}
                    />
                ) : (
                    <ACSEmptyState title="No reports are currently configured." />
                )}
            </PageSection>
        </>
    );
}

export default ReportTablePage;
