import { useCallback, useState } from 'react';

import useAnalytics, {
    VULNERABILITY_REPORT_DOWNLOAD_GENERATED,
    VULNERABILITY_REPORT_SENT_MANUALLY,
} from 'hooks/useAnalytics';
import { runReportRequest } from 'services/ReportsService';
import { ReportNotificationMethod } from 'services/ReportsService.types';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

export type UseSaveReportProps = {
    onCompleted: (context: { reportNotificationMethod: ReportNotificationMethod }) => void;
};

type Result = {
    isRunCompleted: boolean;
    isRunning: boolean;
    runError: string | null;
};

type SaveReportResult = {
    runReport: (reportConfigId: string, reportNotificationMethod: ReportNotificationMethod) => void;
} & Result;

const defaultResult = {
    isRunCompleted: false,
    isRunning: false,
    runError: null,
};

function useRunReport({ onCompleted }: UseSaveReportProps): SaveReportResult {
    const { analyticsTrack } = useAnalytics();
    const [result, setResult] = useState<Result>(defaultResult);

    const runReport = useCallback(
        (reportConfigId: string, reportNotificationMethod: ReportNotificationMethod) => {
            setResult({
                isRunCompleted: false,
                isRunning: true,
                runError: null,
            });

            runReportRequest(reportConfigId, reportNotificationMethod)
                .then(() => {
                    setResult({
                        isRunCompleted: true,
                        isRunning: false,
                        runError: null,
                    });
                    onCompleted({ reportNotificationMethod });

                    if (reportNotificationMethod === 'EMAIL') {
                        analyticsTrack(VULNERABILITY_REPORT_SENT_MANUALLY);
                    } else if (reportNotificationMethod === 'DOWNLOAD') {
                        analyticsTrack(VULNERABILITY_REPORT_DOWNLOAD_GENERATED);
                    }
                })
                .catch((err) => {
                    setResult({
                        isRunCompleted: true,
                        isRunning: false,
                        runError: getAxiosErrorMessage(err),
                    });
                });
        },
        [analyticsTrack, onCompleted]
    );

    return {
        ...result,
        runReport,
    };
}

export default useRunReport;
