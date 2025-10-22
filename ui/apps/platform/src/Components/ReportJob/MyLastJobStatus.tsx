import React from 'react';
import { Spinner } from '@patternfly/react-core';

import ReportJobStatus from 'Components/ReportJob/ReportJobStatus';
import { onDownloadReport } from 'Components/ReportJob/utils';
import type { Snapshot } from 'types/reportJob';

type MyLastJobStatusProps = {
    snapshot: Snapshot | null | undefined;
    isLoadingSnapshots: boolean;
    currentUserId: string;
    baseDownloadURL: string;
};

function MyLastJobStatus({
    snapshot,
    isLoadingSnapshots,
    currentUserId,
    baseDownloadURL,
}: MyLastJobStatusProps) {
    // reportSnapshot is undefined when initially fetching reportSnapshots
    if (isLoadingSnapshots && snapshot === undefined) {
        return <Spinner size="md" aria-label="Fetching my last job status" />;
    }

    if (!snapshot) {
        return 'None';
    }

    const onDownloadHandler = () => {
        const { completedAt } = snapshot.reportStatus;
        const { reportJobId, name } = snapshot;
        return onDownloadReport({ reportJobId, name, completedAt, baseDownloadURL });
    };

    return (
        <ReportJobStatus
            reportStatus={snapshot.reportStatus}
            isDownloadAvailable={snapshot.isDownloadAvailable}
            areDownloadActionsDisabled={currentUserId !== snapshot.user.id}
            onDownload={onDownloadHandler}
        />
    );
}

export default MyLastJobStatus;
