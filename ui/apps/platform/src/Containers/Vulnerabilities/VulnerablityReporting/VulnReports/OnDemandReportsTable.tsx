import React from 'react';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import { OnDemandReportSnapshot } from 'services/ReportsService.types';
import useAuthStatus from 'hooks/useAuthStatus';
import { getDateTime } from 'utils/dateUtils';
import { TableUIState } from 'utils/getTableUIState';
import ReportJobStatus from 'Components/ReportJob/ReportJobStatus';
import TbodyUnified from 'Components/TableStateTemplates/TbodyUnified';
import { GetSortParams } from 'hooks/useURLSort';

export type OnDemandReportsTableProps<T> = {
    tableState: TableUIState<T>;
    getSortParams: GetSortParams;
    onClearFilters: () => void;
};

// eslint-disable-next-line @typescript-eslint/no-unused-vars
const onDownload = (snapshot: OnDemandReportSnapshot) => () => {
    // @TODO: Add download logic here
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
                    <Th sort={getSortParams('Report Completed Time')}>Completed</Th>
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
                                    {/* @TODO: Show the difference between the retention period for on-demand downloadable reports and the date when this was created */}
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
