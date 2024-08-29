import React from 'react';
import { Bullseye, Button, Card, CardBody, Text } from '@patternfly/react-core';
import {
    ActionsColumn,
    ExpandableRowContent,
    Table,
    Tbody,
    Td,
    Th,
    Thead,
    Tr,
} from '@patternfly/react-table';

import EmptyStateTemplate from 'Components/EmptyStateTemplate';
import { ComplianceScanConfigurationStatus } from 'services/ComplianceScanConfigurationService';
import useSet from 'hooks/useSet';
import { getDateTime } from 'utils/dateUtils';
import ReportJobStatus from 'Containers/Vulnerabilities/VulnerablityReporting/ViewVulnReport/ReportJobStatus';
import useAuthStatus from 'hooks/useAuthStatus';
import JobDetails from 'Containers/Vulnerabilities/VulnerablityReporting/ViewVulnReport/JobDetails';
import ConfigDetails from './ConfigDetails';

type ReportJobsProps = {
    scanConfig: ComplianceScanConfigurationStatus | undefined;
};

function createMockData(scanConfig: ComplianceScanConfigurationStatus) {
    const snapshots = [
        {
            scanScheduleConfigId: scanConfig.id,
            scanSchedulReportJobId: 'ab1c03ae-9707-43d1-932d-f948afb67b53',
            name: scanConfig.scanName,
            description: scanConfig.scanConfig.description,
            schedule: scanConfig.scanConfig.scanSchedule,
            reportStatus: {
                completedAt: '2024-08-27T00:01:40.569402380Z',
                errorMsg:
                    "Error sending email notifications:  error: Error sending email for notifier 'fc99e179-57c1-4ba2-8e59-45dbf184c78c': Connection failed",
                reportNotificationMethod: 'EMAIL',
                reportRequestType: 'SCHEDULED',
                runState: 'FAILURE',
            },
            notifiers: [],
            user: {
                id: 'sso:3e30efee-45f0-49d3-aec1-2861fcb3faf6:c02da449-f1c9-4302-afc7-3cbf450f2e0c',
                name: 'Test User',
            },
            isDownloadAvailable: false,
            clusters: [], // @TODO: Figure this out
            profiles: [], // @TODO: Figure this out
        },
    ] as const;
    return snapshots;
}

function ReportJobs({ scanConfig }: ReportJobsProps) {
    const { currentUser } = useAuthStatus();
    const expandedRowSet = useSet<string>();

    const scanConfigSnapshots = scanConfig ? createMockData(scanConfig) : [];

    return (
        <Table aria-label="Scan schedule report jobs table" variant="compact">
            <Thead>
                <Tr>
                    <Th>
                        <span className="pf-v5-screen-reader">Row expansion</span>
                    </Th>
                    <Th width={25}>Completed</Th>
                    <Th width={25}>Status</Th>
                    <Th width={50}>Requestor</Th>
                    <Th>
                        <span className="pf-v5-screen-reader">Row actions</span>
                    </Th>
                </Tr>
            </Thead>
            {scanConfigSnapshots.length === 0 && (
                <Tbody>
                    <Tr>
                        <Td colSpan={5}>
                            <Bullseye>
                                <EmptyStateTemplate title="No report jobs found" headingLevel="h2">
                                    <Text>Clear any search value and try again</Text>
                                    <Button variant="link" onClick={() => {}}>
                                        Clear filters
                                    </Button>
                                </EmptyStateTemplate>
                            </Bullseye>
                        </Td>
                    </Tr>
                </Tbody>
            )}
            {scanConfigSnapshots.map((scanConfigSnapshot, rowIndex) => {
                const { scanScheduleReportJobId, reportStatus, user, isDownloadAvailable } =
                    scanConfigSnapshot;
                const isExpanded = expandedRowSet.has(scanScheduleReportJobId);
                const areDownloadActionsDisabled = currentUser.userId !== user.id;

                function onDownload() {
                    // TODO: Download logic
                }

                const rowActions = [
                    {
                        title: <span className="pf-v5-u-danger-color-100">Delete download</span>,
                        onClick: (event) => {
                            event.preventDefault();
                            // @TODO: Delete logic
                        },
                    },
                ];

                return (
                    <Tbody key={scanScheduleReportJobId} isExpanded={isExpanded}>
                        <Tr>
                            <Td
                                expand={{
                                    rowIndex,
                                    isExpanded,
                                    onToggle: () => expandedRowSet.toggle(scanScheduleReportJobId),
                                }}
                            />
                            <Td dataLabel="Completed">
                                {reportStatus.completedAt
                                    ? getDateTime(reportStatus.completedAt)
                                    : '-'}
                            </Td>
                            <Td dataLabel="Status">
                                <ReportJobStatus
                                    reportStatus={scanConfigSnapshot.reportStatus}
                                    isDownloadAvailable={scanConfigSnapshot.isDownloadAvailable}
                                    areDownloadActionsDisabled={areDownloadActionsDisabled}
                                    onDownload={onDownload}
                                />
                            </Td>
                            <Td dataLabel="Requester">{user.name}</Td>
                            <Td isActionCell>
                                {isDownloadAvailable && (
                                    <ActionsColumn
                                        items={rowActions}
                                        isDisabled={areDownloadActionsDisabled}
                                    />
                                )}
                            </Td>
                        </Tr>
                        <Tr isExpanded={isExpanded}>
                            <Td colSpan={5}>
                                <ExpandableRowContent>
                                    <Card>
                                        <CardBody>
                                            <JobDetails
                                                reportStatus={reportStatus}
                                                isDownloadAvailable={isDownloadAvailable}
                                            />
                                        </CardBody>
                                    </Card>
                                    <ConfigDetails scanConfig={scanConfig} />
                                </ExpandableRowContent>
                            </Td>
                        </Tr>
                    </Tbody>
                );
            })}
        </Table>
    );
}

export default ReportJobs;
