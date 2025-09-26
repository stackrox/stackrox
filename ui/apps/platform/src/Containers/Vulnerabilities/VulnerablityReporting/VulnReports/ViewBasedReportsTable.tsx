import React, { useState } from 'react';
import { Button, Modal } from '@patternfly/react-core';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import { differenceInDays } from 'date-fns';

import type { ViewBasedReportSnapshot } from 'services/ReportsService.types';
import useAuthStatus from 'hooks/useAuthStatus';
import { GetSortParams } from 'hooks/useURLSort';
import useModal from 'hooks/useModal';
import { getDateTime } from 'utils/dateUtils';
import { TableUIState } from 'utils/getTableUIState';
import ReportJobStatus from 'Components/ReportJob/ReportJobStatus';
import TbodyUnified from 'Components/TableStateTemplates/TbodyUnified';
import { downloadReportByJobId } from 'services/ReportsService';
import useAnalytics, {
    VIEW_BASED_REPORT_DOWNLOAD_ATTEMPTED,
    VIEW_BASED_REPORT_JOB_DETAILS_VIEWED,
} from 'hooks/useAnalytics';
import ViewBasedReportJobDetails from './ViewBasedReportJobDetails';

export type ViewBasedReportsTableProps<T> = {
    tableState: TableUIState<T>;
    getSortParams: GetSortParams;
    onClearFilters: () => void;
};

function ViewBasedReportsTable<T extends ViewBasedReportSnapshot>({
    tableState,
    getSortParams,
    onClearFilters,
}: ViewBasedReportsTableProps<T>) {
    const { currentUser } = useAuthStatus();
    const { analyticsTrack } = useAnalytics();
    const { isModalOpen, openModal, closeModal } = useModal();
    const [selectedJobDetails, setSelectedJobDetails] = useState<ViewBasedReportSnapshot | null>(
        null
    );

    const onDownload = (snapshot: ViewBasedReportSnapshot) => () => {
        const { reportJobId, name, reportStatus } = snapshot;
        const { completedAt } = reportStatus;
        const filename = `${name}-${completedAt}`;

        // Calculate report age
        const reportAgeInDays = completedAt
            ? differenceInDays(new Date(), new Date(completedAt))
            : undefined;

        return downloadReportByJobId({
            reportJobId,
            filename,
            fileExtension: 'zip',
        })
            .then(({ fileSizeBytes }) => {
                // Track successful download
                analyticsTrack({
                    event: VIEW_BASED_REPORT_DOWNLOAD_ATTEMPTED,
                    properties: {
                        success: 1,
                        reportAgeInDays,
                        fileSizeBytes,
                    },
                });
            })
            .catch((error) => {
                // Track failed download
                analyticsTrack({
                    event: VIEW_BASED_REPORT_DOWNLOAD_ATTEMPTED,
                    properties: {
                        success: 0,
                        reportAgeInDays,
                        errorType: 'download_failed',
                    },
                });
                throw error; // Re-throw to maintain original behavior
            });
    };

    const onJobDetailsView = (snapshot: ViewBasedReportSnapshot) => {
        setSelectedJobDetails(snapshot);
        openModal();

        // Track job details view
        analyticsTrack({
            event: VIEW_BASED_REPORT_JOB_DETAILS_VIEWED,
            properties: {
                reportStatus: snapshot.reportStatus.runState || 'UNKNOWN',
                isOwnReport: currentUser.userId === snapshot.user.id ? 1 : 0,
            },
        });
    };

    return (
        <>
            <Table aria-label="View-based reports table">
                <Thead>
                    <Tr>
                        <Th width={15}>Request name</Th>
                        <Th>Requester</Th>
                        <Th>Job status</Th>
                        <Th sort={getSortParams('Report Completion Time')}>Completed</Th>
                    </Tr>
                </Thead>
                <TbodyUnified
                    tableState={tableState}
                    colSpan={4}
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
                                                onClick={() => onJobDetailsView(snapshot)}
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
