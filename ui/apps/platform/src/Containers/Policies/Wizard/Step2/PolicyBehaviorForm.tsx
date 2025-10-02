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
import ExternalLink from 'Components/PatternFly/IconText/ExternalLink';
import useMetadata from 'hooks/useMetadata';
import { ClientPolicy } from 'types/policy.proto';
import type { PolicyEventSource } from 'types/policy.proto';
import { getVersionedDocs } from 'utils/versioning';

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

function getEventSourceHelperText(eventSource: PolicyEventSource) {
    if (eventSource === 'DEPLOYMENT_EVENT') {
        return 'Monitor deployments for process activity, baseline deviation, and user issued container commands.';
    }

    if (eventSource === 'AUDIT_LOG_EVENT') {
        return 'Inspect the Kubernetes audit log for access to sensitive Kubernetes resources.';
    }

    return '';
}

function PolicyBehaviorForm({ hasActiveViolations }: PolicyBehaviorFormProps) {
    const { errors, setFieldTouched, setFieldValue, setValues, touched, values } =
        useFormikContext<ClientPolicy>();
    const [lifeCycleChanges, setLifeCycleChanges] = useState<ValidPolicyLifeCycle | null>(null);
    const { version } = useMetadata();

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
                    isExpandable
                    isInline
                    title="How policies work in each lifecycle stage"
                    component="p"
                    className="pf-v5-u-mb-md"
                >
                    <Flex
                        direction={{ default: 'column' }}
                        spaceItems={{ default: 'spaceItemsSm' }}
                        className="pf-v5-u-pt-sm"
                    >
                        <p>
                            <strong>Build stage</strong> policies can inspect only images built in
                            the build pipeline, using criteria related to the image registry,
                            content, vulnerability data and scanning process. When enforced, policy
                            violations may be used to fail the build.
                        </p>
                        <p>
                            <strong>Deploy stage</strong> policies can inspect workload
                            configurations and/or their images. They are evaluated while creating or
                            updating a workload resource and re-evaluated periodically or on demand.
                            When enforced, policy violations result in rejection of workload
                            admission or update, or if admitted, workload replicas being scaled down
                            to zero.
                        </p>
                        <p>
                            <strong>Build+Deploy stage</strong> policies are a convenient option to
                            inspect images in both the build pipeline and during workload admission
                            and apply enforcement to either or both stages in a single policy.
                        </p>
                        <p>
                            <strong>Runtime</strong> policies inspect either workload activity or
                            Kubernetes resource operations, depending on the event source selected.
                            When enforced, runtime policies that inspect workload activity terminate
                            the offending pod. Enforcement is not available for the policies
                            inspecting sensitive operations via the Kubernetes audit log.
                        </p>
                        <div className="pf-v5-u-pt-md">
                            Learn more about policy{' '}
                            <ExternalLink>
                                <a
                                    href={getVersionedDocs(
                                        version,
                                        'operating/managing-security-policies#con-policy-lifecycle_about-security-policies'
                                    )}
                                    target="_blank"
                                    rel="noopener noreferrer"
                                >
                                    lifecycle stages
                                </a>
                            </ExternalLink>
                            .
                        </div>
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
                            'Choose the lifecycle stage to which your policy is applicable.'
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
                                onChange={() => setFieldValue('eventSource', 'DEPLOYMENT_EVENT')}
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
                </div>
            </Form>
        </Flex>
    );
}

export default PolicyBehaviorForm;
