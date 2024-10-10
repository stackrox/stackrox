import React from 'react';
import {
    Card,
    CardBody,
    Divider,
    Pagination,
    Toolbar,
    ToolbarContent,
    ToolbarItem,
} from '@patternfly/react-core';

import {
    ComplianceScanConfigurationStatus,
    ComplianceReportSnapshot,
} from 'services/ComplianceScanConfigurationService';
import JobDetails from 'Containers/Vulnerabilities/VulnerablityReporting/ViewVulnReport/JobDetails';
import ReportJobsTable from 'Components/ReportJob/ReportJobsTable';
import useURLPagination from 'hooks/useURLPagination';
import useURLSearch from 'hooks/useURLSearch';
import { ensureBoolean, ensureStringArray } from 'utils/ensure';
import useURLStringUnion from 'hooks/useURLStringUnion';
import { RunState } from 'types/reportJob';
import useAnalytics from 'hooks/useAnalytics';
import ConfigDetails from './ConfigDetails';
import ReportRunStatesFilter, { ensureReportRunStates } from './ReportRunStatesFilter';
import MyJobsFilter from './MyJobsFilter';

function createMockData(scanConfig: ComplianceScanConfigurationStatus) {
    const snapshots: ComplianceReportSnapshot[] = [
        {
            reportJobId: 'ab1c03ae-9707-43d1-932d-f948afb67b53',
            scanConfigId: scanConfig.id,
            name: scanConfig.scanName,
            description: scanConfig.scanConfig.description,
            reportStatus: {
                completedAt: '2024-08-27T00:01:40.569402380Z',
                errorMsg:
                    "Error sending email notifications:  error: Error sending email for notifier 'fc99e179-57c1-4ba2-8e59-45dbf184c78c': Connection failed",
                reportNotificationMethod: 'EMAIL',
                reportRequestType: 'SCHEDULED',
                runState: 'FAILURE',
            },
            reportData: scanConfig,
            user: {
                id: 'sso:3e30efee-45f0-49d3-aec1-2861fcb3faf6:c02da449-f1c9-4302-afc7-3cbf450f2e0c',
                name: 'Test User',
            },
            isDownloadAvailable: false,
        },
    ];
    return snapshots;
}

function getJobId(snapshot: ComplianceReportSnapshot) {
    return snapshot.scanConfigId;
}

function getConfigName(snapshot: ComplianceReportSnapshot) {
    return snapshot.name;
}

type ReportJobsProps = {
    scanConfig: ComplianceScanConfigurationStatus | undefined;
    isComplianceReportingEnabled: boolean;
};

function ReportJobs({ scanConfig, isComplianceReportingEnabled }: ReportJobsProps) {
    const { analyticsTrack } = useAnalytics();

    const { page, perPage, setPage, setPerPage } = useURLPagination(10);
    const { searchFilter, setSearchFilter } = useURLSearch();
    const [isViewingOnlyMyJobs, setIsViewingOnlyMyJobs] = useURLStringUnion('viewOnlyMyJobs', [
        'false',
        'true',
    ]);

    const filteredReportRunStates = ensureStringArray(searchFilter['Report State']);

    const onReportStatesFilterChange = (_checked: boolean, selectedStatus: RunState) => {
        const isStatusIncluded = filteredReportRunStates.includes(selectedStatus);
        if (isStatusIncluded) {
            const newFilters = ensureReportRunStates(
                filteredReportRunStates.filter((status) => status !== selectedStatus)
            );
            analyticsTrack({
                event: 'Compliance Report Run State Filtered',
                properties: {
                    value: newFilters,
                },
            });
            setSearchFilter({
                ...searchFilter,
                'Report State': newFilters,
            });
        } else {
            const newFilters = ensureReportRunStates([...filteredReportRunStates, selectedStatus]);
            analyticsTrack({
                event: 'Compliance Report Run State Filtered',
                properties: {
                    value: newFilters,
                },
            });
            setSearchFilter({
                ...searchFilter,
                'Report State': newFilters,
            });
        }
        setPage(1);
    };

    const onMyJobsFilterChange = (checked: boolean) => {
        analyticsTrack({
            event: 'Compliance Report Jobs View Toggled',
            properties: {
                view: 'My jobs',
                state: checked,
            },
        });
        setIsViewingOnlyMyJobs(String(checked));
        setPage(1);
    };

    // @TODO: We will eventually make an API request using the scan config id to get the job history
    const complianceScanSnapshots = scanConfig ? createMockData(scanConfig) : [];

    return (
        <>
            <Toolbar>
                <ToolbarContent>
                    <ToolbarItem alignItems="center">
                        <ReportRunStatesFilter
                            reportRunStates={ensureReportRunStates(filteredReportRunStates)}
                            onChange={onReportStatesFilterChange}
                        />
                    </ToolbarItem>
                    <ToolbarItem className="pf-v5-u-flex-grow-1" alignSelf="center">
                        <MyJobsFilter
                            isViewingOnlyMyJobs={ensureBoolean(isViewingOnlyMyJobs)}
                            onMyJobsFilterChange={onMyJobsFilterChange}
                        />
                    </ToolbarItem>
                    <ToolbarItem variant="pagination" align={{ default: 'alignRight' }}>
                        <Pagination
                            toggleTemplate={({ firstIndex, lastIndex }) => (
                                <span>
                                    <b>
                                        {firstIndex} - {lastIndex}
                                    </b>{' '}
                                    of <b>many</b>
                                </span>
                            )}
                            page={page}
                            perPage={perPage}
                            onSetPage={(_, newPage) => setPage(newPage)}
                            onPerPageSelect={(_, newPerPage) => setPerPage(newPerPage)}
                            isCompact
                        />
                    </ToolbarItem>
                </ToolbarContent>
            </Toolbar>
            <ReportJobsTable
                snapshots={complianceScanSnapshots}
                getJobId={getJobId}
                getConfigName={getConfigName}
                onClearFilters={() => {}}
                onDeleteDownload={() => {}}
                renderExpandableRowContent={(snapshot: ComplianceReportSnapshot) => {
                    return (
                        <>
                            <Card isFlat>
                                <CardBody>
                                    <JobDetails
                                        reportStatus={snapshot.reportStatus}
                                        isDownloadAvailable={snapshot.isDownloadAvailable}
                                    />
                                    <Divider component="div" className="pf-v5-u-my-md" />
                                    <ConfigDetails
                                        scanConfig={snapshot.reportData}
                                        isComplianceReportingEnabled={isComplianceReportingEnabled}
                                    />
                                </CardBody>
                            </Card>
                        </>
                    );
                }}
            />
        </>
    );
}

export default ReportJobs;
