import React, { useState } from 'react';
import {
    Alert,
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
import omit from 'lodash/omit';

import ConfirmationModal from 'Components/PatternFly/ConfirmationModal';
import FormLabelGroup from 'Components/PatternFly/FormLabelGroup';
import { ClientPolicy } from 'types/policy.proto';

import {
    getLifeCyclesUpdates,
    initialPolicy,
    isRuntimePolicy,
    isBuildPolicy,
    isBuildAndDeployPolicy,
    isDeployPolicy,
} from '../../policies.utils';
import type { ValidPolicyLifeCycle } from '../../policies.utils';

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
    const [lifeCycleChanges, setLifeCycleChanges] = useState<ValidPolicyLifeCycle | null>(null);

    function onChangeLifecycleStages(lifecycleStages: ValidPolicyLifeCycle) {
        const hasNonEmptyPolicyGroup = values.policySections.some(
            (section) => section.policyGroups.length > 0
        );

        if (hasNonEmptyPolicyGroup) {
            // for existing policies, warn that changing lifecycles will clear all policy criteria
            setLifeCycleChanges(lifecycleStages);
        } else {
            // for new policies, just update lifecycle stages
            const newValues = getLifeCyclesUpdates(values, lifecycleStages);
            setValues(newValues);
        }
    }

    function onConfirmChangeLifecycle(lifecycleStages: ValidPolicyLifeCycle | null) {
        if (lifecycleStages) {
            // first, update the lifecycles
            const newValues = getLifeCyclesUpdates(values, lifecycleStages);

            // second, clear the policy criteria
            const clearedCriteria = cloneDeep(initialPolicy.policySections);
            newValues.policySections = clearedCriteria;
            setValues(newValues);
        }
        setLifeCycleChanges(null);
    }

    function onCancelChangeLifecycle() {
        setLifeCycleChanges(null);
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
            const { ...rest } = omit(scope || {}, 'label');

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

    const isBuild = isBuildPolicy(values.lifecycleStages);
    const isDeploy = isDeployPolicy(values.lifecycleStages);
    const isBuildAndDeploy = isBuildAndDeployPolicy(values.lifecycleStages);
    const isRuntime = isRuntimePolicy(values.lifecycleStages);

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
                isOpen={!!lifeCycleChanges && lifeCycleChanges.length > 0}
                onConfirm={() => onConfirmChangeLifecycle(lifeCycleChanges)}
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
                            <Radio
                                label="Build"
                                isChecked={isBuild}
                                id="policy-lifecycle-stage-build"
                                name="lifecycleStages"
                                onChange={() => {
                                    setFieldTouched('lifecycleStages', true, true);
                                    onChangeLifecycleStages(['BUILD']);
                                }}
                                isDisabled={hasActiveViolations}
                            />
                            <Radio
                                label="Deploy"
                                isChecked={isDeploy}
                                id="policy-lifecycle-stage-deploy"
                                name="lifecycleStages"
                                onChange={() => {
                                    setFieldTouched('lifecycleStages', true, true);
                                    onChangeLifecycleStages(['DEPLOY']);
                                }}
                                isDisabled={hasActiveViolations}
                            />
                            <Radio
                                label="Build and Deploy"
                                isChecked={isBuildAndDeploy}
                                id="policy-lifecycle-stage-build-and-deploy"
                                name="lifecycleStages"
                                onChange={() => {
                                    setFieldTouched('lifecycleStages', true, true);
                                    onChangeLifecycleStages(['BUILD', 'DEPLOY']);
                                }}
                                isDisabled={hasActiveViolations}
                            />
                            <Radio
                                label="Runtime"
                                isChecked={isRuntime}
                                id="policy-lifecycle-stage-runtime"
                                name="lifecycleStages"
                                onChange={() => {
                                    setFieldTouched('lifecycleStages', true, true);
                                    onChangeLifecycleStages(['RUNTIME']);
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
                    {isRuntime && (
                        <FormGroup
                            fieldId="policy-event-source"
                            label="Event sources (Runtime lifecycle only)"
                            isRequired={isRuntime}
                            className="pf-v5-u-pt-lg"
                        >
                            <Flex direction={{ default: 'row' }}>
                                <Radio
                                    label="Deployment"
                                    isChecked={values.eventSource === 'DEPLOYMENT_EVENT'}
                                    id="policy-event-source-deployment"
                                    name="eventSource"
                                    onChange={() =>
                                        setFieldValue('eventSource', 'DEPLOYMENT_EVENT')
                                    }
                                    isDisabled={!isRuntime || hasActiveViolations}
                                />
                                <Radio
                                    label="Audit logs"
                                    isChecked={values.eventSource === 'AUDIT_LOG_EVENT'}
                                    id="policy-event-source-audit-logs"
                                    name="eventSource"
                                    onChange={onChangeAuditLogEventSource}
                                    isDisabled={!isRuntime || hasActiveViolations}
                                />
                            </Flex>
                            <FormHelperText>
                                <HelperText>
                                    <HelperTextItem>{eventSourceHelperText}</HelperTextItem>
                                </HelperText>
                            </FormHelperText>
                        </FormGroup>
                    )}
                </div>
            </Form>
        </Flex>
    );
}

export default PolicyBehaviorForm;
