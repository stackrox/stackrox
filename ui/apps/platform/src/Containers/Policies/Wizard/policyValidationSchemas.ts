import * as yup from 'yup';

import type { ClientPolicy } from 'types/policy.proto';

import {
    POLICY_BEHAVIOR_SCOPE_ID,
    POLICY_DEFINITION_DETAILS_ID,
    POLICY_DEFINITION_LIFECYCLE_ID,
    POLICY_DEFINITION_RULES_ID,
} from '../policies.constants';
import type { WizardPolicyStep4, WizardScope } from '../policies.utils';
import {
    imageSigningCriteriaName,
    mountPropagationCriteriaName,
} from './Step3/policyCriteriaDescriptors';
import { policySectionValidators } from './Step3/policyCriteriaValidators';

type PolicyStep1 = Pick<ClientPolicy, 'name' | 'severity' | 'categories'>;

const policyNameLengthMessage = 'Policy name must be between 5 and 128 characters';
// Backend policy validation reference at central/policy/service/validator.go
const validationSchemaStep1: yup.ObjectSchema<PolicyStep1> = yup.object().shape({
    name: yup
        .string()
        .trim()
        .min(5, policyNameLengthMessage)
        .max(128, policyNameLengthMessage)
        .matches(/^[^\n$]*$/, 'Policy name must not contain newlines or dollar signs')
        .required('Policy name is required'),
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

type PolicyStep2 = Pick<ClientPolicy, 'eventSource' | 'lifecycleStages'>;

const validationSchemaStep2: yup.ObjectSchema<PolicyStep2> = yup.object().shape({
    eventSource: yup
        .string()
        .oneOf(['NOT_APPLICABLE', 'DEPLOYMENT_EVENT', 'AUDIT_LOG_EVENT', 'NODE_EVENT'])
        .when('lifecycleStages', {
            is: (lifecycleStages: string[]) => lifecycleStages.includes('RUNTIME'),
            // Remove values of eventSource that are not relevant for lifecycle stage.
            then: (eventSourceSchema) => eventSourceSchema.notOneOf(['NOT_APPLICABLE']),
            otherwise: (eventSourceSchema) =>
                eventSourceSchema.notOneOf(['DEPLOYMENT_EVENT', 'AUDIT_LOG_EVENT', 'NODE_EVENT']),
        })
        .required(),
    lifecycleStages: yup
        .array()
        .of(yup.string().oneOf(['BUILD', 'DEPLOY', 'RUNTIME']).required())
        .min(1, 'At least one lifecycle stage is required')
        .required(),
    // Schema omits enforcementActions, because code (not user) changes the value.
});

type PolicyStep3 = Pick<ClientPolicy, 'policySections'>;

const validationSchemaStep3: yup.ObjectSchema<PolicyStep3> = yup.object().shape(
    {
        policySections: yup
            .array()
            .of(
                yup
                    .object()
                    .shape({
                        sectionName: yup.string().trim().optional(),
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
                                                                context.from[1]?.value
                                                                    ?.fieldName ===
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
                                                                context.from[1]?.value
                                                                    ?.fieldName ===
                                                                    mountPropagationCriteriaName
                                                            ) {
                                                                const currentValue =
                                                                    context.from[0]?.value?.value;
                                                                return (
                                                                    typeof currentValue ===
                                                                        'string' &&
                                                                    currentValue.trim().length > 0
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
                    .test(
                        'policy-section',
                        'Invalid policy section',
                        (value, context: yup.TestContext) => {
                            // @ts-expect-error: `yup` hard codes the `context.from.value` type here as `any`, so this is not a safe cast.
                            // We will assert as `unknown` and then cast to `PolicyStep2` with a comment so this unsafe cast is visible.
                            const topLevelContext: PolicyStep2 = context.from?.[
                                context.from.length - 1
                            ].value as unknown;

                            // Run all applicable validators, stopping at the first error
                            let validationError: string | undefined;
                            policySectionValidators.forEach((validator) => {
                                if (!validationError && validator.appliesTo(topLevelContext)) {
                                    const error = validator.validate(value, topLevelContext);
                                    if (error) {
                                        validationError = error;
                                    }
                                }
                            });

                            return validationError
                                ? context.createError({ message: validationError })
                                : true;
                        }
                    )
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

export function getValidationSchema(stepId: number | string): yup.Schema {
    switch (stepId) {
        case POLICY_DEFINITION_DETAILS_ID:
            return validationSchemaStep1;
        case POLICY_DEFINITION_LIFECYCLE_ID:
            return validationSchemaStep2;
        case POLICY_DEFINITION_RULES_ID:
            return validationSchemaStep3;
        case POLICY_BEHAVIOR_SCOPE_ID:
            return validationSchemaStep4;
        default:
            return validationSchemaStep5;
    }
}
