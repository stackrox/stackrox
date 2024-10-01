import React from 'react';
import { Spinner } from '@patternfly/react-core';

import { ReportSnapshot } from 'services/ReportsService.types';
import ReportJobStatus from 'Containers/Vulnerabilities/VulnerablityReporting/ViewVulnReport/ReportJobStatus';
import { onDownloadReport } from 'Components/ReportJob/utils';

type MyLastReportJobStatusProps = {
    reportSnapshot: ReportSnapshot | null | undefined;
    isLoadingReportSnapshots: boolean;
    currentUserId: string;
    baseDownloadURL: string;
};

function MyLastReportJobStatus({
    reportSnapshot,
    isLoadingReportSnapshots,
    currentUserId,
    baseDownloadURL,
}: MyLastReportJobStatusProps) {
    // reportSnapshot is undefined when initially fetching reportSnapshots
    if (isLoadingReportSnapshots && reportSnapshot === undefined) {
        return <Spinner size="md" aria-label="Fetching my last job status" />;
    }

    if (!reportSnapshot) {
        return 'None';
    }

    const onDownloadHandler = () => {
        const { completedAt } = reportSnapshot.reportStatus;
        const { name } = reportSnapshot;
        const { reportJobId } = reportSnapshot;
        return onDownloadReport({ reportJobId, name, completedAt, baseDownloadURL });
    };

    return (
        <ReportJobStatus
            reportStatus={reportSnapshot.reportStatus}
            isDownloadAvailable={reportSnapshot.isDownloadAvailable}
            areDownloadActionsDisabled={currentUserId !== reportSnapshot.user.id}
            onDownload={onDownloadHandler}
        />
    );
}

export default MyLastReportJobStatus;
