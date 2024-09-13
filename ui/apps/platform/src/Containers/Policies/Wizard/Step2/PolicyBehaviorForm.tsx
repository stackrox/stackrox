import React, { useState } from 'react';
import {
    Alert,
    Checkbox,
    Divider,
    Flex,
    Form,
    FormGroup,
    Radio,
    Title,
    FormHelperText,
    HelperText,
    HelperTextItem,
} from '@patternfly/react-core';
import { useFormikContext } from 'formik';
import cloneDeep from 'lodash/cloneDeep';

import ConfirmationModal from 'Components/PatternFly/ConfirmationModal';
import FormLabelGroup from 'Components/PatternFly/FormLabelGroup';
import { ClientPolicy, LifecycleStage } from 'types/policy.proto';

import { getLifeCyclesUpdates, initialPolicy } from '../../policies.utils';

type PolicyBehaviorFormProps = {
    hasActiveViolations: boolean;
};

function getEventSourceHelperText(eventSource) {
    if (eventSource === 'DEPLOYMENT_EVENT') {
        return 'Event sources that include process and network activity, pod exec and pod port forwarding.';
    }

    if (eventSource === 'AUDIT_LOG_EVENT') {
        return 'Event sources that match Kubernetes audit log records.';
    }

    return '';
}

function PolicyBehaviorForm({ hasActiveViolations }: PolicyBehaviorFormProps) {
    const { errors, setFieldTouched, setFieldValue, setValues, touched, values } =
        useFormikContext<ClientPolicy>();
    const [lifeCycleChange, setLifeCycleChange] = useState<{
        lifecycleStage: LifecycleStage;
        isChecked: boolean;
    } | null>(null);

    function onChangeLifecycleStage(lifecycleStage: LifecycleStage, isChecked: boolean) {
        const hasNonEmptyPolicyGroup = values.policySections.some(
            (section) => section.policyGroups.length > 0
        );
        if (hasNonEmptyPolicyGroup) {
            // for existing policies, warn that changing lifecycles will clear all policy criteria
            setLifeCycleChange({ lifecycleStage, isChecked });
        } else {
            // for new policies, just update lifecycle stages
            const newValues = getLifeCyclesUpdates(values, lifecycleStage, isChecked);
            setValues(newValues);
        }
    }

    function onConfirmChangeLifecycle(
        lifecycleStage: LifecycleStage | undefined,
        isChecked: boolean | undefined
    ) {
        // type guard, because TS is a cruel master
        if (lifecycleStage) {
            // first, update the lifecycles
            const newValues = getLifeCyclesUpdates(values, lifecycleStage, !!isChecked);

            // second, clear the policy criteria
            const clearedCriteria = cloneDeep(initialPolicy.policySections);
            newValues.policySections = clearedCriteria;
            setValues(newValues);
        }
        setLifeCycleChange(null);
    }

    function onCancelChangeLifecycle() {
        setLifeCycleChange(null);
    }

    function onChangeAuditLogEventSource() {
        setFieldValue('eventSource', 'AUDIT_LOG_EVENT');

        // Do not validate the following, because changed values are on other steps.
        setFieldValue('excludedImageNames', [], false);
        values.scope.forEach(({ label, ...rest }, idx) => {
            if (label) {
                setFieldValue(`scope[${idx}]`, { ...rest }, false);
            }
        });
        values.excludedDeploymentScopes.forEach(({ scope }, idx) => {
            // disable because unused label might be specified for rest spread idiom.
            /* eslint-disable @typescript-eslint/no-unused-vars */
            const { label, ...rest } = scope || {};
            /* eslint-enable @typescript-eslint/no-unused-vars */
            setFieldValue(
                `excludedDeploymentScopes[${idx}]`,
                {
                    scope: {
                        ...rest,
                    },
                },
                false
            );
        });

        // clear policy sections to prevent non-runtime criteria from being sent to BE
        const clearedCriteria = cloneDeep(initialPolicy.policySections);
        setFieldValue('policySections', clearedCriteria, false);
    }

    const eventSourceHelperText = getEventSourceHelperText(values.eventSource);

    const hasBuild = values.lifecycleStages.includes('BUILD');
    const hasDeploy = values.lifecycleStages.includes('DEPLOY');
    const hasRuntime = values.lifecycleStages.includes('RUNTIME');

    return (
        <Flex
            direction={{ default: 'column' }}
            spaceItems={{ default: 'spaceItemsNone' }}
            flexWrap={{ default: 'nowrap' }}
        >
            <Flex
                direction={{ default: 'column' }}
                spaceItems={{ default: 'spaceItemsSm' }}
                className="pf-v5-u-p-lg"
            >
                <Title headingLevel="h2">Lifecycle</Title>
                <div>
                    Select which stage of a container lifecycle this policy applies. Event sources
                    can only be chosen for policies that apply at runtime.
                </div>
            </Flex>
            <Divider component="div" />
            <ConfirmationModal
                ariaLabel="Reset policy criteria"
                confirmText="Reset policy criteria"
                isOpen={!!lifeCycleChange}
                onConfirm={() =>
                    onConfirmChangeLifecycle(
                        lifeCycleChange?.lifecycleStage,
                        lifeCycleChange?.isChecked
                    )
                }
                onCancel={onCancelChangeLifecycle}
                title="Reset policy criteria?"
            >
                Editing the lifecycle stage will reset and clear any saved rules for this policy.
                You will be required to reselect policy rules in the next step.
            </ConfirmationModal>
            <Flex
                direction={{ default: 'column' }}
                spaceItems={{ default: 'spaceItemsNone' }}
                className="pf-v5-u-px-lg pf-v5-u-pt-lg"
            >
                <Alert
                    variant="info"
                    isInline
                    title="Lifecycle stages"
                    component="p"
                    className="pf-v5-u-mb-md"
                >
                    <Flex
                        direction={{ default: 'column' }}
                        spaceItems={{ default: 'spaceItemsSm' }}
                    >
                        <p>
                            Build-time policies apply to image fields such as CVEs and Dockerfile
                            instructions.
                        </p>
                        <p>
                            Deploy-time policies can include all build-time policy criteria but they
                            can also include data from your cluster configurations, such as running
                            in privileged mode or mounting the Docker socket.
                        </p>
                        <p>
                            Runtime policies can include all build-time and deploy-time policy
                            criteria but they <strong>must</strong> include at least one policy
                            criterion from process, network flow, audit log events, or Kubernetes
                            events criteria categories.
                        </p>
                    </Flex>
                </Alert>
            </Flex>
            <Form>
                <div className="pf-v5-u-px-lg">
                    <FormLabelGroup
                        label="Lifecycle stages"
                        fieldId="lifecycleStages"
                        errors={errors}
                        isRequired
                        touched={touched}
                        helperText={
                            'Choose lifecycle stage to which your policy is applicable. You can select more than one stage.'
                        }
                    >
                        <Flex direction={{ default: 'row' }} className="pf-v5-u-pb-sm">
                            <Checkbox
                                label="Build"
                                isChecked={hasBuild}
                                id="policy-lifecycle-stage-build"
                                onChange={(_event, isChecked) => {
                                    setFieldTouched('lifecycleStages', true, true);
                                    onChangeLifecycleStage('BUILD', isChecked);
                                }}
                                isDisabled={hasActiveViolations}
                            />
                            <Checkbox
                                label="Deploy"
                                isChecked={hasDeploy}
                                id="policy-lifecycle-stage-deploy"
                                onChange={(_event, isChecked) => {
                                    setFieldTouched('lifecycleStages', true, true);
                                    onChangeLifecycleStage('DEPLOY', isChecked);
                                }}
                                isDisabled={hasActiveViolations}
                            />
                            <Checkbox
                                label="Runtime"
                                isChecked={hasRuntime}
                                id="policy-lifecycle-stage-runtime"
                                onChange={(_event, isChecked) => {
                                    setFieldTouched('lifecycleStages', true, true);
                                    onChangeLifecycleStage('RUNTIME', isChecked);
                                }}
                                isDisabled={hasActiveViolations}
                            />
                        </Flex>
                    </FormLabelGroup>
                    {hasActiveViolations && (
                        <Alert
                            isInline
                            variant="warning"
                            title="Policy has active violations, and the lifecycle stage cannot be changed. To update the lifecycle, clone and create a new policy."
                            component="p"
                        />
                    )}
                    <FormGroup
                        fieldId="policy-event-source"
                        label="Event sources (Runtime lifecycle only)"
                        isRequired={hasRuntime}
                        className="pf-v5-u-pt-lg"
                    >
                        <Flex direction={{ default: 'row' }}>
                            <Radio
                                label="Deployment"
                                isChecked={values.eventSource === 'DEPLOYMENT_EVENT'}
                                id="policy-event-source-deployment"
                                name="eventSource"
                                onChange={() => setFieldValue('eventSource', 'DEPLOYMENT_EVENT')}
                                isDisabled={!hasRuntime || hasActiveViolations}
                            />
                            <Radio
                                label="Audit logs"
                                isChecked={values.eventSource === 'AUDIT_LOG_EVENT'}
                                id="policy-event-source-audit-logs"
                                name="eventSource"
                                onChange={onChangeAuditLogEventSource}
                                isDisabled={!hasRuntime || hasActiveViolations}
                            />
                        </Flex>
                        <FormHelperText>
                            <HelperText>
                                <HelperTextItem>{eventSourceHelperText}</HelperTextItem>
                            </HelperText>
                        </FormHelperText>
                    </FormGroup>
                </div>
            </Form>
        </Flex>
    );
}

export default PolicyBehaviorForm;
