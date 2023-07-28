import { useCallback, useState } from 'react';

import { runReportRequest } from 'services/ReportsService';
import { ReportNotificationMethod } from 'services/ReportsService.types';

export type UseSaveReportProps = {
    onCompleted: () => void;
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
                    onCompleted();
                })
                .catch((err) => {
                    setResult({
                        isRunCompleted: true,
                        isRunning: false,
                        runError: err.response.data.message,
                    });
                });
        },
        [onCompleted]
    );

    return {
        ...result,
        runReport,
    };
}

export default useRunReport;
