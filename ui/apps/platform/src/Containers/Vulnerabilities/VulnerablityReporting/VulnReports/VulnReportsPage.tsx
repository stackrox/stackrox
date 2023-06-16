import React from 'react';
import {
    PageSection,
    Title,
    Divider,
    Flex,
    FlexItem,
    Button,
    Card,
    CardBody,
} from '@patternfly/react-core';
import { TableComposable, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import { ReportConfiguration } from 'types/reportConfigurationService.proto';
import { ReportStatus } from 'types/report.proto';
import { Report } from 'Containers/Vulnerabilities/VulnerablityReporting/types';
import usePermissions from 'hooks/usePermissions';

import PageTitle from 'Components/PageTitle';
import HelpIconTh from './HelpIconTh';
import LastRunStatusState from './LastRunStatusState';
import LastRunState from './LastRunState';

const reportConfigurations: ReportConfiguration[] = [
    {
        id: '1',
        name: 'bob-report-1',
        description: '',
        type: 'VULNERABILITY',
        vulnReportFilters: {
            fixability: 'FIXABLE',
            severities: ['CRITICAL_VULNERABILITY_SEVERITY', 'IMPORTANT_VULNERABILITY_SEVERITY'],
            imageTypes: ['DEPLOYED', 'WATCHED'],
            allVuln: true,
        },
        emailConfig: {
            notifierId: 'notifier-1',
            mailingLists: ['bob@example.com', 'alice@example.com'],
        },
        resourceScope: {
            collectionScope: {
                collectionId: 'collection-1',
                collectionName: 'bob-collection-1',
            },
        },
        schedule: {
            intervalType: 'WEEKLY',
            hour: 10,
            minute: 0,
            daysOfWeek: {
                days: ['0', '1', '2', '3', '4', '5', '6'],
            },
        },
    },
];

const reportStatus: ReportStatus = {
    runState: 'SUCCESS',
    runTime: '2023-06-20T10:59:46.383433891Z',
    errorMsg: '',
    reportMethod: 'ON_DEMAND',
    reportNotificationMethod: 'DOWNLOAD',
};

const reportLastRunStatus: ReportStatus = {
    runState: 'FAILURE',
    runTime: '2023-06-20T10:59:46.383433891Z',
    errorMsg: 'Failed to generate download, please try again',
    reportMethod: 'ON_DEMAND',
    reportNotificationMethod: 'DOWNLOAD',
};

function VulnReportsPage() {
    const { hasReadWriteAccess, hasReadAccess } = usePermissions();

    const hasWorkflowAdministrationWriteAccess = hasReadWriteAccess('WorkflowAdministration');
    const hasImageReadAccess = hasReadAccess('Image');
    const hasAccessScopeReadAccess = hasReadAccess('Access');
    const hasNotifierIntegrationReadAccess = hasReadAccess('Integration');
    const canCreateReports =
        hasWorkflowAdministrationWriteAccess &&
        hasImageReadAccess &&
        hasAccessScopeReadAccess &&
        hasNotifierIntegrationReadAccess;

    const reports = reportConfigurations.map((reportConfiguration): Report => {
        return {
            ...reportConfiguration,
            reportStatus,
            reportLastRunStatus,
        };
    });
    return (
        <>
            <PageTitle title="Vulnerability reporting" />
            <Divider component="div" />
            <PageSection variant="light" padding={{ default: 'noPadding' }}>
                <Flex
                    direction={{ default: 'row' }}
                    alignItems={{ default: 'alignItemsCenter' }}
                    className="pf-u-py-lg pf-u-px-lg"
                >
                    <FlexItem flex={{ default: 'flex_1' }}>
                        <Flex direction={{ default: 'column' }}>
                            <FlexItem>
                                <Title headingLevel="h1">Vulnerability reporting</Title>
                            </FlexItem>
                            <FlexItem>
                                Configure reports, define report scopes, and assign distribution
                                lists to report on vulnerabilities across the organization.
                            </FlexItem>
                        </Flex>
                    </FlexItem>
                    <FlexItem>
                        {canCreateReports && (
                            <Button variant="primary" onClick={() => {}}>
                                Create report
                            </Button>
                        )}
                    </FlexItem>
                </Flex>
            </PageSection>
            <PageSection padding={{ default: 'noPadding' }}>
                <PageSection isCenterAligned>
                    <Card>
                        <CardBody className="pf-u-p-0">
                            <TableComposable borders={false}>
                                <Thead noWrap>
                                    <Tr>
                                        <Th>Report</Th>
                                        <HelpIconTh tooltip="A set of user-configured rules for selecting deployments as part of the report scope">
                                            Collection
                                        </HelpIconTh>
                                        <Th>Last run status</Th>
                                        <HelpIconTh tooltip="The report that was last run by a schedule or an on-demand action including 'send report now' and 'generate a downloadable report'">
                                            Last run
                                        </HelpIconTh>
                                    </Tr>
                                </Thead>
                                {reports.map((report) => {
                                    return (
                                        <Tbody
                                            key={report.id}
                                            style={{
                                                borderBottom:
                                                    '1px solid var(--pf-c-table--BorderColor)',
                                            }}
                                        >
                                            <Tr>
                                                <Td>{report.name}</Td>
                                                <Td>
                                                    {
                                                        report.resourceScope.collectionScope
                                                            .collectionName
                                                    }
                                                </Td>
                                                <Td>
                                                    <LastRunStatusState
                                                        reportStatus={report.reportLastRunStatus}
                                                    />
                                                </Td>
                                                <Td>
                                                    <LastRunState
                                                        reportStatus={report.reportStatus}
                                                    />
                                                </Td>
                                            </Tr>
                                        </Tbody>
                                    );
                                })}
                            </TableComposable>
                        </CardBody>
                    </Card>
                </PageSection>
            </PageSection>
        </>
    );
}

export default VulnReportsPage;
