import { FormikProps, useFormik } from 'formik';
import * as yup from 'yup';

import { ScanConfigFormValues } from '../compliance.scanConfigs.utils';

export const defaultScanConfigFormValues: ScanConfigFormValues = {
    parameters: {
        name: '',
        description: '',
        intervalType: 'DAILY',
        time: '',
        daysOfWeek: [],
        daysOfMonth: [],
    },
    clusters: [],
    profiles: [],
};

const validationSchema = yup.object().shape({
    parameters: yup.object().shape({
        name: yup
            .string()
            .trim()
            .required('Scan name is required')
            .matches(
                /^[a-z0-9][a-z0-9.-]{0,251}[a-z0-9]$/,
                "Name can contain only lowercase alphanumeric characters, '-' or '.', and start and end with an alphanumeric character."
            ),
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
    clusters: yup.array().min(1),
    profiles: yup.array().min(1),
});

function useFormikScanConfig(initialFormValues): FormikProps<ScanConfigFormValues> {
    const formik = useFormik<ScanConfigFormValues>({
        initialValues: initialFormValues || defaultScanConfigFormValues,
        validationSchema,
        onSubmit: () => {},
        validateOnMount: true,
    });

    return formik;
}

export default useFormikScanConfig;
