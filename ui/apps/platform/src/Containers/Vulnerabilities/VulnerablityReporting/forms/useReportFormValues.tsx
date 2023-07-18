import { Dispatch, SetStateAction, useState } from 'react';
import { Collection } from 'services/CollectionsService';

import { VulnerabilitySeverity } from 'types/cve.proto';
import { ImageType } from 'types/reportConfigurationService.proto';

export type ReportFormValuesResult = {
    formValues: ReportFormValues;
    setFormValues: SetReportFormValues;
};

export type SetReportFormValues = Dispatch<SetStateAction<ReportFormValues>>;

export type ReportFormValues = {
    reportParameters: ReportParametersFormValues;
};

export type ReportParametersFormValues = {
    reportName: string;
    description: string;
    cveSeverities: VulnerabilitySeverity[];
    cveStatus: CVEStatus[];
    imageType: ImageType[];
    cvesDiscoveredSince: CVESDiscoveredSince;
    cvesDiscoveredStartDate: string | undefined;
    reportScope: Collection | null;
};

export type CVEStatus = 'FIXABLE' | 'NOT_FIXABLE';

export type CVESDiscoveredSince = 'ALL_VULN' | 'SINCE_LAST_REPORT' | 'START_DATE';

export const defaultReportFormValues: ReportFormValues = {
    reportParameters: {
        reportName: '',
        description: '',
        cveSeverities: [],
        cveStatus: [],
        imageType: [],
        cvesDiscoveredSince: 'ALL_VULN',
        cvesDiscoveredStartDate: undefined,
        reportScope: null,
    },
};

function useReportFormValues(): ReportFormValuesResult {
    const [formValues, setFormValues] = useState<ReportFormValues>(defaultReportFormValues);

    return {
        formValues,
        setFormValues,
    };
}

export default useReportFormValues;
