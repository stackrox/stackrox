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
    ComplianceScanSnapshot,
} from 'services/ComplianceScanConfigurationService';
import JobDetails from 'Containers/Vulnerabilities/VulnerablityReporting/ViewVulnReport/JobDetails';
import ReportJobsTable from 'Components/ReportJob/ReportJobsTable';
import { RunState } from 'services/ReportsService.types';
import useURLPagination from 'hooks/useURLPagination';
import useURLSearch from 'hooks/useURLSearch';
import { ensureBoolean, ensureStringArray } from 'utils/ensure';
import ConfigDetails from './ConfigDetails';
import ReportStatesFilter, { ensureReportStates } from './ReportStatusFilter';
import MyJobsFilter from './MyJobsFilter';

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
    const { page, perPage, setPage, setPerPage } = useURLPagination(10);
    const { searchFilter, setSearchFilter } = useURLSearch();

    const filteredReportStates = ensureStringArray(searchFilter['Report state']);
    const viewOnlyMyJobs = ensureBoolean(searchFilter['View only my jobs']);

    const onReportStatesFilterChange = (_checked: boolean, selectedStatus: RunState) => {
        const isStatusIncluded = filteredReportStates.includes(selectedStatus);
        if (isStatusIncluded) {
            setSearchFilter({
                ...searchFilter,
                'Report state': filteredReportStates.filter((status) => status !== selectedStatus),
            });
        } else {
            setSearchFilter({
                ...searchFilter,
                'Report state': [...filteredReportStates, selectedStatus],
            });
        }
        setPage(1);
    };

    const onMyJobsFilterChange = (checked: boolean) => {
        setSearchFilter({
            ...searchFilter,
            'View only my jobs': String(checked),
        });
        setPage(1);
    };

    // @TODO: We will eventually make an API request using the scan config id to get the job history
    const complianceScanSnapshots = scanConfig ? createMockData(scanConfig) : [];

    return (
        <>
            <Toolbar>
                <ToolbarContent>
                    <ToolbarItem alignItems="center">
                        <ReportStatesFilter
                            reportStates={ensureReportStates(filteredReportStates)}
                            onChange={onReportStatesFilterChange}
                        />
                    </ToolbarItem>
                    <ToolbarItem className="pf-v5-u-flex-grow-1" alignSelf="center">
                        <MyJobsFilter
                            viewOnlyMyJobs={viewOnlyMyJobs}
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
        </>
    );
}

export default ReportJobs;
