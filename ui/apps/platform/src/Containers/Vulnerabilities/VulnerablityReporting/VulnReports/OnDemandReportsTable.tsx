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

import useAuthStatus from 'hooks/useAuthStatus';
import { saveFile } from 'services/DownloadService';
import { getDateTime } from 'utils/dateUtils';
import ReportJobStatus from 'Components/ReportJob/ReportJobStatus';
import { GetSortParams } from 'hooks/useURLSort';
import { TableUIState } from 'utils/getTableUIState';
import TbodyUnified from 'Components/TableStateTemplates/TbodyUnified';
import { sanitizeFilename } from 'utils/fileUtils';
import { OnDemandReportSnapshot } from 'services/ReportsService.types';

export type OnDemandReportsTableProps<T> = {
    tableState: TableUIState<T>;
    getSortParams: GetSortParams;
    onClearFilters: () => void;
};

const onDownload = (snapshot: OnDemandReportSnapshot) => () => {
    const { requestName, reportJobId } = snapshot;
    const { completedAt } = snapshot.reportStatus;
    const filename = `${requestName}-${completedAt}`;
    const sanitizedFilename = sanitizeFilename(filename);
    return saveFile({
        method: 'get',
        url: `/v2/compliance/scan/configurations/reports/download?id=${reportJobId}`,
        data: null,
        timeout: 300000,
        name: `${sanitizedFilename}.zip`,
    });
};

function OnDemandReportsTable<T extends OnDemandReportSnapshot>({
    tableState,
    getSortParams,
    onClearFilters,
}: OnDemandReportsTableProps<T>) {
    const { currentUser } = useAuthStatus();

    return (
        <Table aria-label="On-demand reports table">
            <Thead>
                <Tr>
                    <Th width={15}>Request name</Th>
                    <Th>Requester</Th>
                    <Th>Job status</Th>
                    <Th>Expiration</Th>
                    <Th sort={getSortParams('Compliance Report Completed Time')}>Completed</Th>
                </Tr>
            </Thead>
            <TbodyUnified
                tableState={tableState}
                colSpan={5}
                emptyProps={{
                    title: 'No on-demand reports found',
                    message: '', // Figure out what to put as the call-to-action
                }}
                filteredEmptyProps={{ onClearFilters }}
                renderer={({ data }) =>
                    data.map((snapshot) => {
                        const {
                            user,
                            reportStatus,
                            isDownloadAvailable,
                            reportJobId,
                            requestName,
                        } = snapshot;
                        const areDownloadActionsDisabled = currentUser.userId !== user.id;

                        return (
                            <Tbody key={reportJobId}>
                                <Tr>
                                    <Td dataLabel="Request name">{requestName}</Td>
                                    <Td dataLabel="Requester">{user.name}</Td>
                                    <Td dataLabel="Job status">
                                        <ReportJobStatus
                                            reportStatus={reportStatus}
                                            isDownloadAvailable={isDownloadAvailable}
                                            areDownloadActionsDisabled={areDownloadActionsDisabled}
                                            onDownload={onDownload(snapshot)}
                                        />
                                    </Td>
                                    <Td dataLabel="Expiration">7 days</Td>
                                    <Td dataLabel="Completed">
                                        {reportStatus.completedAt
                                            ? getDateTime(reportStatus.completedAt)
                                            : '-'}
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

export default OnDemandReportsTable;
