import { useFormik } from 'formik';
import type { FormikProps } from 'formik';
import * as yup from 'yup';

import {
    customBodyValidation,
    customSubjectValidation,
} from 'Components/EmailTemplate/EmailTemplate.utils';

import {
    areNodeRolesValid,
    defaultNodeRoles,
    nodeRoleValidationMessage,
} from '../compliance.scanConfigs.utils';
import type { ScanConfigFormValues } from '../compliance.scanConfigs.utils';

export const defaultScanConfigFormValues: ScanConfigFormValues = {
    parameters: {
        name: '',
        description: '',
        intervalType: 'DAILY',
        time: '',
        daysOfWeek: [],
        daysOfMonth: [],
        nodeRoles: [...defaultNodeRoles],
    },
    clusters: [],
    profiles: [],
    report: {
        notifierConfigurations: [],
    },
};

export const helperTextForName =
    "Name can contain only lowercase alphanumeric characters, hyphen '-' or period '.', and start and end with an alphanumeric character.";
export const helperTextForNameEdit =
    "Scan config name cannot be changed because it's linked to existing scan results.";
export const helperTextForTime = 'Select or enter scan time between 00:00 and 23:59 UTC';
export const helperTextForNodeRoles =
    'Determines which nodes are scanned for node-type profiles. If left empty, defaults to master and worker. Common roles: master, worker, infra, control-plane. Use @all to scan all nodes.';

const timeRegExp = /\d\d:\d\d/;

function timeValidator(value: string) {
    if (!timeRegExp.test(value)) {
        return false;
    }

    const [hourString, minuteString] = value.split(':');
    const hour = parseInt(hourString);
    const minute = parseInt(minuteString);
    return hour >= 0 && hour < 24 && minute >= 0 && minute < 60;
}

const validationSchema = yup.object().shape({
    parameters: yup.object().shape({
        name: yup
            .string()
            // omit trim because confusion when RegExp matches trimmed string
            .required('Name is required')
            .matches(/^[a-z0-9][a-z0-9.-]{0,251}[a-z0-9]$/, helperTextForName),
        description: yup.string(),
        intervalType: yup.string().required('Frequency is required'),
        time: yup
            .string()
            .required('Time is required')
            .test('time', helperTextForTime, timeValidator),
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
        nodeRoles: yup
            .array()
            .of(yup.string().required())
            .test('valid-node-roles', nodeRoleValidationMessage, (roles) =>
                areNodeRolesValid(roles ?? [])
            ),
    }),
    clusters: yup.array().min(1),
    profiles: yup.array().min(1),
    report: yup.object().shape({
        notifierConfigurations: yup
            .array()
            .of(
                yup.object().shape({
                    emailConfig: yup.object().shape({
                        notifierId: yup.string().required('A notifier is required'),
                        mailingLists: yup
                            .array()
                            .of(yup.string())
                            .min(1, 'At least 1 delivery destination is required'),
                        customSubject: customSubjectValidation,
                        customBody: customBodyValidation,
                    }),
                    notifierName: yup.string(),
                })
            )
            .strict(),
    }),
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
