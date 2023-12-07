import { FormikProps, useFormik } from 'formik';
import * as yup from 'yup';

import { DayOfMonth, DayOfWeek } from 'Components/PatternFly/DayPickerDropdown';

export type ScanConfigParameters = {
    name: string;
    description: string;
    intervalType: 'DAILY' | 'WEEKLY' | 'MONTHLY' | null;
    time: string;
    daysOfWeek: DayOfWeek[];
    daysOfMonth: DayOfMonth[];
};

export type ScanConfigFormValues = {
    parameters: ScanConfigParameters;
    clusters: string[];
    profiles: string[];
};

export const defaultScanConfigFormValues: ScanConfigFormValues = {
    parameters: {
        name: '',
        description: '',
        intervalType: null,
        time: '',
        daysOfWeek: [],
        daysOfMonth: [],
    },
    clusters: [],
    profiles: [],
};

const validationSchema = yup.object().shape({
    parameters: yup.object().shape({
        name: yup.string().required('Scan name is required'),
        description: yup.string(),
        intervalType: yup.string().required('Frequency is required'),
        time: yup.string().required('Time is required'),
        daysOfWeek: yup
            .array()
            .when('intervalType', (intervalType, schema) =>
                intervalType[0] === 'WEEKLY'
                    ? schema.of(yup.string()).min(1, 'Selection is required')
                    : schema.notRequired()
            ),
        daysOfMonth: yup
            .array()
            .when('intervalType', (intervalType, schema) =>
                intervalType[0] === 'MONTHLY'
                    ? schema.of(yup.string()).min(1, 'Selection is required')
                    : schema.notRequired()
            ),
    }),
});

function useFormikScanConfig(): FormikProps<ScanConfigFormValues> {
    const formik = useFormik<ScanConfigFormValues>({
        initialValues: defaultScanConfigFormValues,
        validationSchema,
        onSubmit: () => {},
        validateOnMount: true,
    });

    return formik;
}

export default useFormikScanConfig;
