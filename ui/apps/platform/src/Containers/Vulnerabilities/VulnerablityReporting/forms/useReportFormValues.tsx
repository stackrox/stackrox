import { Dispatch, SetStateAction, useState } from 'react';
import cloneDeep from 'lodash/cloneDeep';
import set from 'lodash/set';

import { VulnerabilitySeverity } from 'types/cve.proto';
import { ImageType, IntervalType } from 'services/ReportsService.types';
import { DayOfMonth, DayOfWeek } from 'Components/PatternFly/DayPickerDropdown';

export type ReportFormValuesResult = {
    formValues: ReportFormValues;
    setFormValues: SetReportFormValues;
    clearFormValues: () => void;
    setFormFieldValue: SetReportFormFieldValue;
};

export type ReportFormValues = {
    reportId: string;
    reportParameters: ReportParametersFormValues;
    deliveryDestinations: DeliveryDestination[];
    schedule: {
        intervalType: IntervalType | null;
        daysOfWeek: DayOfWeek[];
        daysOfMonth: DayOfMonth[];
    };
};

export type SetReportFormValues = Dispatch<SetStateAction<ReportFormValues>>;

export type SetReportFormFieldValue = (
    fieldName: string,
    value: string | string[] | DeliveryDestination[]
) => void;

export type ReportParametersFormValues = {
    reportName: string;
    description: string;
    cveSeverities: VulnerabilitySeverity[];
    cveStatus: CVEStatus[];
    imageType: ImageType[];
    cvesDiscoveredSince: CVESDiscoveredSince;
    cvesDiscoveredStartDate: CVESDiscoveredStartDate;
    reportScope: ReportScope | null;
};

export type CVEStatus = 'FIXABLE' | 'NOT_FIXABLE';

export type CVESDiscoveredSince = 'ALL_VULN' | 'SINCE_LAST_REPORT' | 'START_DATE';

export type CVESDiscoveredStartDate = string | undefined;

export type ReportScope = {
    id: string;
    name: string;
};

export type DeliveryDestination = {
    notifier: ReportNotifier | null;
    mailingLists: string[];
};

export type ReportNotifier = {
    id: string;
    name: string;
};

export const defaultReportFormValues: ReportFormValues = {
    reportId: '',
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
    deliveryDestinations: [],
    schedule: {
        intervalType: null,
        daysOfWeek: [],
        daysOfMonth: [],
    },
};

function useReportFormValues(): ReportFormValuesResult {
    const [formValues, setFormValues] = useState<ReportFormValues>(defaultReportFormValues);

    function setFormFieldValue(
        fieldName: string,
        value: string | string[] | DeliveryDestination[]
    ) {
        setFormValues((prevValues) => {
            const newValues = cloneDeep(prevValues);
            set(newValues, fieldName, value);
            return newValues;
        });
    }

    function clearFormValues() {
        setFormValues(defaultReportFormValues);
    }

    return {
        formValues,
        setFormValues,
        clearFormValues,
        setFormFieldValue,
    };
}

export default useReportFormValues;
