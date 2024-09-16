import React from 'react';
import { Card, CardBody, Divider } from '@patternfly/react-core';

import {
    ComplianceScanConfigurationStatus,
    ComplianceScanSnapshot,
} from 'services/ComplianceScanConfigurationService';
import JobDetails from 'Containers/Vulnerabilities/VulnerablityReporting/ViewVulnReport/JobDetails';
import ReportJobsTable from 'Components/ReportJobsTable';
import ConfigDetails from './ConfigDetails';

function createMockData(scanConfig: ComplianceScanConfigurationStatus) {
    const snapshots: ComplianceScanSnapshot[] = [
        {
            reportJobId: 'ab1c03ae-9707-43d1-932d-f948afb67b53',
            reportStatus: {
                completedAt: '2024-08-27T00:01:40.569402380Z',
                errorMsg:
                    "Error sending email notifications:  error: Error sending email for notifier 'fc99e179-57c1-4ba2-8e59-45dbf184c78c': Connection failed",
                reportNotificationMethod: 'EMAIL',
                reportRequestType: 'SCHEDULED',
                runState: 'FAILURE',
            },
            user: {
                id: 'sso:3e30efee-45f0-49d3-aec1-2861fcb3faf6:c02da449-f1c9-4302-afc7-3cbf450f2e0c',
                name: 'Test User',
            },
            isDownloadAvailable: false,
            scanConfig,
        },
    ];
    return snapshots;
}

function getJobId(snapshot: ComplianceScanSnapshot) {
    return snapshot.scanConfig.id;
}

function getConfigName(snapshot: ComplianceScanSnapshot) {
    return snapshot.scanConfig.scanName;
}

type ReportJobsProps = {
    scanConfig: ComplianceScanConfigurationStatus | undefined;
};

function ReportJobs({ scanConfig }: ReportJobsProps) {
    // @TODO: We will eventually make an API request using the scan config id to get the job history
    const complianceScanSnapshots = scanConfig ? createMockData(scanConfig) : [];

    return (
        <ReportJobsTable
            snapshots={complianceScanSnapshots}
            getJobId={getJobId}
            getConfigName={getConfigName}
            onClearFilters={() => {}}
            onDeleteDownload={() => {}}
            renderExpandableRowContent={(snapshot: ComplianceScanSnapshot) => {
                return (
                    <>
                        <Card isFlat>
                            <CardBody>
                                <JobDetails
                                    reportStatus={snapshot.reportStatus}
                                    isDownloadAvailable={snapshot.isDownloadAvailable}
                                />
                                <Divider component="div" className="pf-v5-u-my-md" />
                                <ConfigDetails scanConfig={snapshot.scanConfig} />
                            </CardBody>
                        </Card>
                    </>
                );
            }}
        />
    );
}

export default ReportJobs;
