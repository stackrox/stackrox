import React from 'react';
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
import { Snapshot } from 'types/reportJob';
import { saveFile } from 'services/DownloadService';
import { getDateTime } from 'utils/dateUtils';
import ReportJobStatus from 'Components/ReportJob/ReportJobStatus';
import { GetSortParams } from 'hooks/useURLSort';
import { TableUIState } from 'utils/getTableUIState';
import TbodyUnified from 'Components/TableStateTemplates/TbodyUnified';
import { sanitizeFilename } from 'utils/fileUtils';

export type ReportJobsTableProps<T> = {
    tableState: TableUIState<T>;
    getSortParams: GetSortParams;
    getJobId: (data: T) => string;
    getConfigName: (data: T) => string;
    onClearFilters: () => void;
    onDeleteDownload: (reportJobId: string) => void;
    renderExpandableRowContent: (snapshot: T) => React.ReactNode;
};

const onDownload = (snapshot: Snapshot, jobId: string, configName: string) => () => {
    const { completedAt } = snapshot.reportStatus;
    const filename = `${configName}-${completedAt}`;
    const sanitizedFilename = sanitizeFilename(filename);
    return saveFile({
        method: 'get',
        url: `/v2/compliance/scan/configurations/reports/download?id=${jobId}`,
        data: null,
        timeout: 300000,
        name: `${sanitizedFilename}.zip`,
    });
};

function ReportJobsTable<T extends Snapshot>({
    tableState,
    getSortParams,
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
                    <Th width={25} sort={getSortParams('Compliance Report Completed Time')}>
                        Completed
                    </Th>
                    <Th>Status</Th>
                    <Th>Requester</Th>
                    <Th>
                        <span className="pf-v5-screen-reader">Row actions</span>
                    </Th>
                </Tr>
            </Thead>
            <TbodyUnified
                tableState={tableState}
                colSpan={5}
                emptyProps={{
                    title: 'No report jobs found',
                    message:
                        'Send a report now or generate a downloadable report to trigger a job.',
                }}
                filteredEmptyProps={{ onClearFilters }}
                renderer={({ data }) =>
                    data.map((snapshot, rowIndex) => {
                        const { user, reportStatus, isDownloadAvailable } = snapshot;
                        const jobId = getJobId(snapshot);
                        const configName = getConfigName(snapshot);
                        const isExpanded = expandedRowSet.has(jobId);
                        const areDownloadActionsDisabled = currentUser.userId !== user.id;

                        const rowActions = [
                            {
                                title: (
                                    <span className="pf-v5-u-danger-color-100">
                                        Delete download
                                    </span>
                                ),
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
                    })
                }
            />
        </Table>
    );
}

export default ReportJobsTable;
