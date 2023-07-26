import { useCallback, useState } from 'react';

import { createReportConfiguration } from 'services/ReportsService';
import { ReportConfiguration } from 'services/ReportsService.types';
import { ReportFormValues } from '../forms/useReportFormValues';
import { getReportConfigurationFromFormValues } from '../utils';

export type UseCreateReportProps = {
    onCompleted: (response: ReportConfiguration) => void;
};

export type Result = {
    data: ReportConfiguration | null;
    isLoading: boolean;
    error: string | null;
};

export type CreateReportResult = {
    createReport: (formValues: ReportFormValues) => void;
} & Result;

const defaultResult = {
    data: null,
    isLoading: false,
    error: null,
};

function useCreateReport({ onCompleted }: UseCreateReportProps): CreateReportResult {
    const [result, setResult] = useState<Result>(defaultResult);

    const createReport = useCallback((formValues: ReportFormValues) => {
        setResult({
            data: null,
            isLoading: true,
            error: null,
        });

        const reportConfiguration = getReportConfigurationFromFormValues(formValues);

        // send API call
        createReportConfiguration(reportConfiguration)
            .then((response) => {
                setResult({
                    data: response,
                    isLoading: false,
                    error: null,
                });
                onCompleted(response);
            })
            .catch((err) => {
                setResult({
                    data: null,
                    isLoading: false,
                    error: err.response.data.message,
                });
            });
    }, []);

    return {
        ...result,
        createReport,
    };
}

export default useCreateReport;
