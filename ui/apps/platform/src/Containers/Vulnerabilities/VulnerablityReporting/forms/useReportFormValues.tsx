import { FormikProps, useFormik } from 'formik';
import * as yup from 'yup';

import { VulnerabilitySeverity, vulnerabilitySeverities } from 'types/cve.proto';
import { ImageType, IntervalType, imageTypes, intervalTypes } from 'services/ReportsService.types';
import {
    DayOfMonth,
    DayOfWeek,
    daysOfMonth,
    daysOfWeek,
} from 'Components/PatternFly/DayPickerDropdown';

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

export type SetReportFormValues = (newFormValues: ReportFormValues) => void;

export type FormFieldValue =
    | string
    | string[]
    | DeliveryDestination[]
    | Partial<Record<string, string | string[] | null>>;

export type SetReportFormFieldValue = (fieldName: string, value: FormFieldValue) => void;

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

export const cveStatuses = ['FIXABLE', 'NOT_FIXABLE'] as const;

export type CVEStatus = (typeof cveStatuses)[number];

export const cvesDiscoveredSince = ['ALL_VULN', 'SINCE_LAST_REPORT', 'START_DATE'] as const;

export type CVESDiscoveredSince = (typeof cvesDiscoveredSince)[number];

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
        cveSeverities: ['CRITICAL_VULNERABILITY_SEVERITY', 'IMPORTANT_VULNERABILITY_SEVERITY'],
        cveStatus: ['FIXABLE'],
        imageType: ['DEPLOYED', 'WATCHED'],
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

const validationSchema = yup.object().shape({
    reportId: yup.string(),
    reportParameters: yup.object().shape({
        reportName: yup.string().required('Report name is required'),
        description: yup.string(),
        cveSeverities: yup
            .array()
            .of(yup.string().oneOf(vulnerabilitySeverities))
            .min(1, 'Select at least 1 CVE severity'),
        cveStatus: yup
            .array()
            .of(yup.string().oneOf(cveStatuses))
            .min(1, 'Select at least 1 CVE status'),
        imageType: yup
            .array()
            .of(yup.string().oneOf(imageTypes))
            .min(1, 'Select at least 1 image type'),
        cvesDiscoveredSince: yup
            .string()
            .oneOf(cvesDiscoveredSince)
            .required('CVEs discovered since is required'),
        cvesDiscoveredStartDate: yup.string().when('cvesDiscoveredSince', {
            is: 'START_DATE',
            then: (schema) => schema.required('A custom start date is required'),
            otherwise: (schema) => schema,
        }),
        reportScope: yup.object().required('A report scope is required'),
    }),
    deliveryDestinations: yup
        .array()
        .of(
            yup.object().shape({
                notifier: yup
                    .object()
                    .nullable()
                    .strict()
                    .test('is-not-null', 'A notifier is required', (value) => {
                        return value !== null && value !== undefined;
                    }),
                mailingLists: yup
                    .array()
                    .of(yup.string())
                    .min(1, 'At least 1 delivery destination is required'),
            })
        )
        .strict()
        .when('reportParameters.cvesDiscoveredSince', {
            is: 'SINCE_LAST_REPORT',
            then: (schema) => {
                return schema.min(
                    1,
                    'Delivery destination & schedule are both required to be configured since the "Last successful scheduled run report" option has been selected in Step 1.'
                );
            },
            otherwise: (schema) => schema,
        })
        .notRequired(),
    schedule: yup.object().shape({
        intervalType: yup
            .string()
            .oneOf(intervalTypes)
            .strict()
            .test(
                'non-empty-delivery-destinations',
                'A schedule frequency is required',
                (value, context) => {
                    const deliveryDestinations = context?.from?.[1].value.deliveryDestinations;
                    if (deliveryDestinations.length !== 0) {
                        return value === 'MONTHLY' || value === 'WEEKLY';
                    }
                    return true;
                }
            )
            .notRequired(),
        daysOfWeek: yup
            .array()
            .of(yup.string().oneOf(daysOfWeek))
            .when('intervalType', {
                is: 'WEEKLY',
                then: (schema) => schema.min(1, 'At least 1 day is required'),
                otherwise: (schema) => schema,
            })
            .defined(),
        daysOfMonth: yup
            .array()
            .of(yup.string().oneOf(daysOfMonth))
            .when('intervalType', {
                is: 'MONTHLY',
                then: (schema) => schema.min(1, 'At least 1 day is required'),
                otherwise: (schema) => schema,
            })
            .defined(),
    }),
});

function useReportFormValues(): FormikProps<ReportFormValues> {
    const formik = useFormik({
        initialValues: defaultReportFormValues,
        validationSchema,
        onSubmit: () => {},
    });

    return formik;
}

export default useReportFormValues;
