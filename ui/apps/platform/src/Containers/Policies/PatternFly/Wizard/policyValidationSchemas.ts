/* eslint-disable import/prefer-default-export */
import * as yup from 'yup';

import { WizardPolicyStep4, WizardScope } from '../policies.utils';

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

const scopeSchema: yup.ObjectSchema<WizardScope> = yup.object().shape({
    cluster: yup.string(),
    namespace: yup.string(),
    label: yup
        .object()
        .shape({
            key: yup.string(),
            value: yup.string(),
        })
        .nullable(),
});

export const validationSchemaStep4: yup.ObjectSchema<WizardPolicyStep4> = yup.object().shape({
    scope: yup
        .array()
        .of(
            scopeSchema.test(
                'scope-has-at-least-one-property',
                () => 'scope must have at least one property',
                ({ cluster, namespace, label }) => {
                    // Optional chaining in case unexpected temporary states while editing.
                    return Boolean(
                        cluster?.trim() ||
                            namespace?.trim() ||
                            label?.key?.trim() ||
                            label?.value?.trim()
                    );
                }
            )
        )
        .required(),
    excludedDeploymentScopes: yup
        .array()
        .of(
            yup
                .object()
                .shape({
                    name: yup.string(),
                    scope: scopeSchema,
                })
                .test(
                    'excluded-deployment-has-at-least-one-property',
                    () => 'excluded deployment must have at least one property',
                    ({ name, scope }) => {
                        // Optional chaining in case unexpected temporary states while editing.
                        return Boolean(
                            name?.trim() ||
                                scope?.cluster?.trim() ||
                                scope?.namespace?.trim() ||
                                scope?.label?.key?.trim() ||
                                scope?.label?.value?.trim()
                        );
                    }
                )
        )
        .required(),
    excludedImageNames: yup.array().of(yup.string().trim().required()).required(),
});

const validationSchemaDefault = yup.object().shape({});

export function getValidationSchema(stepId: number | string | undefined): yup.Schema {
    switch (stepId) {
        case 1:
            return validationSchema1;
        case 2:
            return validationSchema2;
        case 3:
            return validationSchema3;
        case 4:
            return validationSchemaStep4;
        default:
            return validationSchemaDefault;
    }
}
