import React from 'react';
import { Bullseye, Button, Text } from '@patternfly/react-core';
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

import useSet from 'hooks/useSet';
import useAuthStatus from 'hooks/useAuthStatus';
import { ReportSnapshot } from 'services/ReportsService.types';
import { saveFile } from 'services/DownloadService';
import { getDateTime } from 'utils/dateUtils';
import ReportJobStatus from 'Containers/Vulnerabilities/VulnerablityReporting/ViewVulnReport/ReportJobStatus';
import { ComplianceScanSnapshot } from 'services/ComplianceScanConfigurationService';
import EmptyStateTemplate from './EmptyStateTemplate';

export type ReportJobsTableProps<T> = {
    snapshots: T[];
    getJobId: (data: T) => string;
    getConfigName: (data: T) => string;
    onClearFilters: () => void;
    onDeleteDownload: (reportId) => void;
    renderExpandableRowContent: (snapshot: T) => React.ReactNode;
};

type Snapshot = ReportSnapshot | ComplianceScanSnapshot;

const filenameSanitizerRegex = new RegExp('(:)|(/)|(\\s)', 'gi');

const onDownload = (snapshot: Snapshot, jobId: string, configName: string) => () => {
    const { completedAt } = snapshot.reportStatus;
    const filename = `${configName}-${completedAt}`;
    const sanitizedFilename = filename.replaceAll(filenameSanitizerRegex, '_');
    return saveFile({
        method: 'get',
        // @TODO: We may need to allow passing specific endpoints depending on backend
        url: `/api/reports/jobs/download?id=${jobId}`,
        data: null,
        timeout: 300000,
        name: `${sanitizedFilename}.zip`,
    });
};

function ReportJobsTable<T extends Snapshot>({
    snapshots,
    getJobId,
    getConfigName,
    onClearFilters,
    onDeleteDownload,
    renderExpandableRowContent,
}: ReportJobsTableProps<T>) {
    const { currentUser } = useAuthStatus();
    const expandedRowSet = useSet<string>();

    return (
        <Table aria-label="Jobs table" variant="compact">
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
            {snapshots.length === 0 && (
                <Tbody>
                    <Tr>
                        <Td colSpan={5}>
                            <Bullseye>
                                <EmptyStateTemplate title="No report jobs found" headingLevel="h2">
                                    <Text>Clear any search value and try again</Text>
                                    <Button variant="link" onClick={onClearFilters}>
                                        Clear filters
                                    </Button>
                                </EmptyStateTemplate>
                            </Bullseye>
                        </Td>
                    </Tr>
                </Tbody>
            )}
            {snapshots.map((snapshot, rowIndex) => {
                const { user, reportStatus, isDownloadAvailable } = snapshot;
                const jobId = getJobId(snapshot);
                const configName = getConfigName(snapshot);
                const isExpanded = expandedRowSet.has(jobId);
                const areDownloadActionsDisabled = currentUser.userId !== user.id;

                const rowActions = [
                    {
                        title: <span className="pf-v5-u-danger-color-100">Delete download</span>,
                        onClick: (event) => {
                            event.preventDefault();
                            onDeleteDownload(jobId);
                        },
                    },
                ];

                return (
                    <Tbody key={jobId} isExpanded={isExpanded}>
                        <Tr>
                            <Td
                                expand={{
                                    rowIndex,
                                    isExpanded,
                                    onToggle: () => expandedRowSet.toggle(jobId),
                                }}
                            />
                            <Td dataLabel="Completed">
                                {reportStatus.completedAt
                                    ? getDateTime(reportStatus.completedAt)
                                    : '-'}
                            </Td>
                            <Td dataLabel="Status">
                                <ReportJobStatus
                                    reportStatus={reportStatus}
                                    isDownloadAvailable={isDownloadAvailable}
                                    areDownloadActionsDisabled={areDownloadActionsDisabled}
                                    onDownload={onDownload(snapshot, jobId, configName)}
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
                                    {renderExpandableRowContent(snapshot)}
                                </ExpandableRowContent>
                            </Td>
                        </Tr>
                    </Tbody>
                );
            })}
        </Table>
    );
}

export default ReportJobsTable;
