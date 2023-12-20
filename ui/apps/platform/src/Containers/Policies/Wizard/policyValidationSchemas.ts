import * as yup from 'yup';

import { Policy } from 'types/policy.proto';

import { WizardPolicyStep4, WizardScope } from '../policies.utils';
import {
    imageSigningCriteriaName,
    mountPropagationCriteriaName,
} from './Step3/policyCriteriaDescriptors';

type PolicyStep1 = Pick<Policy, 'name' | 'severity' | 'categories'>;

const validationSchemaStep1: yup.ObjectSchema<PolicyStep1> = yup.object().shape({
    name: yup.string().trim().required('Policy name is required'),
    severity: yup
        .string()
        .oneOf(['LOW_SEVERITY', 'MEDIUM_SEVERITY', 'HIGH_SEVERITY', 'CRITICAL_SEVERITY'])
        .required(),
    categories: yup
        .array()
        .of(yup.string().required())
        .min(1, 'At least one category is required')
        .required(),
});

type PolicyStep2 = Pick<Policy, 'eventSource' | 'lifecycleStages'>;

const validationSchemaStep2: yup.ObjectSchema<PolicyStep2> = yup.object().shape({
    eventSource: yup
        .string()
        .oneOf(['NOT_APPLICABLE', 'DEPLOYMENT_EVENT', 'AUDIT_LOG_EVENT'])
        .when('lifecycleStages', {
            is: (lifecycleStages: string[]) => lifecycleStages.includes('RUNTIME'),
            // Remove values of eventSource that are not relevant for lifecycle stage.
            then: (eventSourceSchema) => eventSourceSchema.notOneOf(['NOT_APPLICABLE']),
            otherwise: (eventSourceSchema) =>
                eventSourceSchema.notOneOf(['DEPLOYMENT_EVENT', 'AUDIT_LOG_EVENT']),
        })
        .required(),
    lifecycleStages: yup
        .array()
        .of(yup.string().oneOf(['BUILD', 'DEPLOY', 'RUNTIME']).required())
        .min(1, 'At least one lifecycle stage is required')
        .required(),
    // Schema omits enforcementActions, because code (not user) changes the value.
});

// TODO validation apparently fails when sectionName is empty string, but why?
/*
type PolicyStep3 = Pick<Policy, 'policySections'>;

const validationSchemaStep3: yup.ObjectSchema<PolicyStep3> = yup.object().shape({
*/
const validationSchemaStep3 = yup.object().shape(
    {
        policySections: yup
            .array()
            .of(
                yup.object().shape({
                    /*
                sectionName: yup.string().defined(),
                */
                    policyGroups: yup
                        .array()
                        .of(
                            yup.object().shape({
                                fieldName: yup.string().trim().required(),
                                booleanOperator: yup.string().oneOf(['OR', 'AND']).required(),
                                negate: yup.boolean().required(),
                                values: yup
                                    .array()
                                    .of(
                                        yup.object().shape({
                                            // value: yup.string(), // dryrun validates whether value is required
                                            arrayValue: yup
                                                .array(yup.string().required())
                                                .test(
                                                    'policy-criteria',
                                                    'Please enter a valid value',
                                                    (value, context: yup.TestContext) => {
                                                        if (
                                                            // from[1] means one level up in the object
                                                            context.from &&
                                                            context.from[1]?.value?.fieldName ===
                                                                imageSigningCriteriaName
                                                        ) {
                                                            return (
                                                                Array.isArray(value) &&
                                                                value.length !== 0
                                                            );
                                                        }
                                                        if (
                                                            // from[1] means one level up in the object
                                                            context.from &&
                                                            context.from[1]?.value?.fieldName ===
                                                                mountPropagationCriteriaName
                                                        ) {
                                                            return (
                                                                Array.isArray(
                                                                    context.from[0]?.value?.value
                                                                ) &&
                                                                context.from[0]?.value?.value
                                                                    .length !== 0
                                                            );
                                                        }

                                                        return true;
                                                    }
                                                ),
                                        })
                                    )
                                    .min(1)
                                    .required(),
                            })
                        )
                        .min(1)
                        .required(),
                })
            )
            .min(1)
            .required(),
    },
    [['value', 'arrayValue']]
);

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

const validationSchemaStep5 = yup.object().shape({});

export function getValidationSchema(stepId: number | string | undefined): yup.Schema {
    switch (stepId) {
        case 1:
            return validationSchemaStep1;
        case 2:
            return validationSchemaStep2;
        case 3:
            return validationSchemaStep3;
        case 4:
            return validationSchemaStep4;
        default:
            return validationSchemaStep5;
    }
}
