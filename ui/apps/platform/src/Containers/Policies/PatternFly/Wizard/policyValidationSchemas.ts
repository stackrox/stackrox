/* eslint-disable import/prefer-default-export */
import * as yup from 'yup';

const validationSchema1 = yup.object().shape({
    name: yup.string().trim().required('Policy name is required'),
    severity: yup
        .string()
        .trim()
        .oneOf(['LOW_SEVERITY', 'MEDIUM_SEVERITY', 'HIGH_SEVERITY', 'CRITICAL_SEVERITY']),
    categories: yup.array().of(yup.string().trim()).min(1, 'At least one category is required'), // TODO redundant? .required('Category is required'),
});

const validationSchema2 = yup.object().shape({
    lifecycleStages: yup
        .array()
        .of(yup.string().trim().oneOf(['BUILD', 'DEPLOY', 'RUNTIME']))
        .min(1, 'At least one lifecycle state is required'), // TODO redundant? .required('Lifecycle stage is required'),
});

const validationSchema3 = yup.object().shape({}); // TODO

const validationSchema4 = yup.object().shape({}); // TODO

const validationSchemaDefault = yup.object().shape({});

export function getValidationSchema(stepId: number | string | undefined): yup.BaseSchema {
    switch (stepId) {
        case 1:
            return validationSchema1;
        case 2:
            return validationSchema2;
        case 3:
            return validationSchema3;
        case 4:
            return validationSchema4;
        default:
            return validationSchemaDefault;
    }
}
