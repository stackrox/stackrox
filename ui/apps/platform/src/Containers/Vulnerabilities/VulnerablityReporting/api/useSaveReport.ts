import { useCallback, useState } from 'react';

import { updateReportConfiguration } from 'services/ReportsService';
import { ReportConfiguration } from 'services/ReportsService.types';
import { ReportFormValues } from '../forms/useReportFormValues';
import { getReportConfigurationFromFormValues } from '../utils';

export type UseSaveReportProps = {
    onCompleted: (response: ReportConfiguration) => void;
};

type Result = {
    data: ReportConfiguration | null;
    isSaving: boolean;
    saveError: string | null;
};

type SaveReportResult = {
    saveReport: (reportId: string, formValues: ReportFormValues) => void;
} & Result;

const defaultResult = {
    data: null,
    isSaving: false,
    saveError: null,
};

function useSaveReport({ onCompleted }: UseSaveReportProps): SaveReportResult {
    const [result, setResult] = useState<Result>(defaultResult);

    const saveReport = useCallback((reportId: string, formValues: ReportFormValues) => {
        setResult({
            data: null,
            isSaving: true,
            saveError: null,
        });

        const reportConfiguration = getReportConfigurationFromFormValues({
            ...formValues,
            reportId,
        });

        // send API call
        updateReportConfiguration(reportId, reportConfiguration)
            .then((response) => {
                setResult({
                    data: response,
                    isSaving: false,
                    saveError: null,
                });
                onCompleted(response);
            })
            .catch((err) => {
                setResult({
                    data: null,
                    isSaving: false,
                    saveError: err.response.data.message,
                });
            });
    }, []);

    return {
        ...result,
        saveReport,
    };
}

export default useSaveReport;
