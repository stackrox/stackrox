import React, { useState } from 'react';
import { Button, Modal } from '@patternfly/react-core';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import type { ViewBasedReportSnapshot } from 'services/ReportsService.types';
import useAuthStatus from 'hooks/useAuthStatus';
import { GetSortParams } from 'hooks/useURLSort';
import useModal from 'hooks/useModal';
import { getDateTime } from 'utils/dateUtils';
import { TableUIState } from 'utils/getTableUIState';
import ReportJobStatus from 'Components/ReportJob/ReportJobStatus';
import TbodyUnified from 'Components/TableStateTemplates/TbodyUnified';
import { downloadReportByJobId } from 'services/ReportsService';
import ViewBasedReportJobDetails from './ViewBasedReportJobDetails';

export type ViewBasedReportsTableProps<T> = {
    tableState: TableUIState<T>;
    getSortParams: GetSortParams;
    onClearFilters: () => void;
};

const onDownload = (snapshot: ViewBasedReportSnapshot) => () => {
    const { reportJobId, name, reportStatus } = snapshot;
    const { completedAt } = reportStatus;
    const filename = `${name}-${completedAt}`;
    return downloadReportByJobId({
        reportJobId,
        filename,
        fileExtension: 'zip',
    });
};

function ViewBasedReportsTable<T extends ViewBasedReportSnapshot>({
    tableState,
    getSortParams,
    onClearFilters,
}: ViewBasedReportsTableProps<T>) {
    const { currentUser } = useAuthStatus();
    const { isModalOpen, openModal, closeModal } = useModal();
    const [selectedJobDetails, setSelectedJobDetails] = useState<ViewBasedReportSnapshot | null>(
        null
    );

    return (
        <>
            <Table aria-label="View-based reports table">
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
                        title: 'No view-based reports found',
                        message: '', // Figure out what to put as the call-to-action
                    }}
                    filteredEmptyProps={{ onClearFilters }}
                    renderer={({ data }) =>
                        data.map((snapshot) => {
                            const { user, reportStatus, isDownloadAvailable, reportJobId, name } =
                                snapshot;
                            const areDownloadActionsDisabled = currentUser.userId !== user.id;

                            return (
                                <Tbody key={reportJobId}>
                                    <Tr>
                                        <Td dataLabel="Request name">
                                            <Button
                                                variant="link"
                                                isInline
                                                onClick={() => {
                                                    setSelectedJobDetails(snapshot);
                                                    openModal();
                                                }}
                                            >
                                                {name}
                                            </Button>
                                        </Td>
                                        <Td dataLabel="Requester">{user.name}</Td>
                                        <Td dataLabel="Job status">
                                            <ReportJobStatus
                                                reportStatus={reportStatus}
                                                isDownloadAvailable={isDownloadAvailable}
                                                areDownloadActionsDisabled={
                                                    areDownloadActionsDisabled
                                                }
                                                onDownload={onDownload(snapshot)}
                                            />
                                        </Td>
                                        {/* @TODO: Show the difference between the retention period for view-based downloadable reports and the date when this was created */}
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
            {selectedJobDetails && (
                <Modal
                    variant="small"
                    title="Request parameters"
                    isOpen={isModalOpen}
                    onClose={() => {
                        setSelectedJobDetails(null);
                        closeModal();
                    }}
                >
                    <ViewBasedReportJobDetails reportSnapshot={selectedJobDetails} />
                </Modal>
            )}
        </>
    );
}

export default ViewBasedReportsTable;
