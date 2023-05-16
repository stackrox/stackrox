import React, { useState } from 'react';
import {
    Alert,
    Title,
    Form,
    FormGroup,
    Radio,
    Divider,
    Flex,
    Checkbox,
    Card,
    CardHeader,
    CardBody,
    CardTitle,
    CardActions,
    Switch,
    Grid,
    GridItem,
} from '@patternfly/react-core';
import { useFormikContext } from 'formik';
import cloneDeep from 'lodash/cloneDeep';

import ConfirmationModal from 'Components/PatternFly/ConfirmationModal';
import { ClientPolicy, LifecycleStage } from 'types/policy.proto';

import {
    appendEnforcementActionsForAddedLifecycleStage,
    filterEnforcementActionsForRemovedLifecycleStage,
    getLifeCyclesUpdates,
    hasEnforcementActionForLifecycleStage,
    initialPolicy,
} from '../../policies.utils';
import DownloadCLIDropdown from './DownloadCLIDropdown';

import './PolicyBehaviorForm.css';

type PolicyBehaviorFormProps = {
    hasActiveViolations: boolean;
};

function PolicyBehaviorForm({ hasActiveViolations }: PolicyBehaviorFormProps) {
    const { values, setFieldValue, setValues } = useFormikContext<ClientPolicy>();
    const [lifeCycleChange, setLifeCycleChange] = useState<{
        lifecycleStage: LifecycleStage;
        isChecked: boolean;
    } | null>(null);

    const hasEnforcementActions =
        values.enforcementActions?.length > 0 &&
        !values.enforcementActions?.includes('UNSET_ENFORCEMENT');
    const [showEnforcement, setShowEnforcement] = React.useState(hasEnforcementActions);

    function onChangeLifecycleStage(lifecycleStage: LifecycleStage, isChecked: boolean) {
        if (values.id) {
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

    function onChangeEnforcementActions(lifecycleStage: LifecycleStage, isChecked: boolean) {
        const { enforcementActions } = values;
        setFieldValue(
            'enforcementActions',
            isChecked
                ? appendEnforcementActionsForAddedLifecycleStage(lifecycleStage, enforcementActions)
                : filterEnforcementActionsForRemovedLifecycleStage(
                      lifecycleStage,
                      enforcementActions
                  ),
            false // do not validate, because code changes the value
        );
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
            const { label, ...rest } = scope || {};
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

    const responseMethodHelperText = showEnforcement
        ? 'Inform and enforce will execute enforcement behavior at the stages you select.'
        : 'Inform will always include violations for this policy in the violations list.';

    const hasBuild = values.lifecycleStages.includes('BUILD');
    const hasDeploy = values.lifecycleStages.includes('DEPLOY');
    const hasRuntime = values.lifecycleStages.includes('RUNTIME');

    return (
        <Flex
            direction={{ default: 'column' }}
            spaceItems={{ default: 'spaceItemsNone' }}
            flexWrap={{ default: 'nowrap' }}
        >
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
                Editing the lifecycle stage will reset and clear any saved criteria for this policy.
                You will be required to reselect policy criteria in the next step.
            </ConfirmationModal>
            <Flex
                direction={{ default: 'column' }}
                spaceItems={{ default: 'spaceItemsNone' }}
                className="pf-u-p-lg"
            >
                <Title headingLevel="h2">Policy behavior</Title>
                <div className="pf-u-mt-sm">
                    Select which stage of a container lifecycle this policy applies. Event sources
                    can only be chosen for policies that apply at runtime.
                </div>
                <Alert variant="info" isInline title="Info" className="pf-u-my-md">
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
                <Form>
                    <FormGroup
                        helperText="Choose lifecycle stage to which your policy is applicable. You can select more than one stage."
                        fieldId="policy-lifecycle-stage"
                        label="Lifecycle stages"
                        isRequired
                    >
                        <Flex direction={{ default: 'row' }} className="pf-u-pb-sm">
                            <Checkbox
                                label="Build"
                                isChecked={hasBuild}
                                id="policy-lifecycle-stage-build"
                                onChange={(isChecked) => {
                                    onChangeLifecycleStage('BUILD', isChecked);
                                }}
                                isDisabled={hasActiveViolations}
                            />
                            <Checkbox
                                label="Deploy"
                                isChecked={hasDeploy}
                                id="policy-lifecycle-stage-deploy"
                                onChange={(isChecked) => {
                                    onChangeLifecycleStage('DEPLOY', isChecked);
                                }}
                                isDisabled={hasActiveViolations}
                            />
                            <Checkbox
                                label="Runtime"
                                isChecked={hasRuntime}
                                id="policy-lifecycle-stage-runtime"
                                onChange={(isChecked) => {
                                    onChangeLifecycleStage('RUNTIME', isChecked);
                                }}
                                isDisabled={hasActiveViolations}
                            />
                        </Flex>
                    </FormGroup>
                    {hasActiveViolations && (
                        <Alert
                            isInline
                            variant="warning"
                            title="Policy has active violations, and the lifecycle stage cannot be changed. To update the lifecycle, clone and create a new policy."
                        />
                    )}
                    <FormGroup
                        fieldId="policy-event-source"
                        label="Event sources (Runtime lifecycle only)"
                        isRequired={hasRuntime}
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
                    </FormGroup>
                </Form>
            </Flex>
            <Divider component="div" />
            <Flex
                direction={{ default: 'column' }}
                spaceItems={{ default: 'spaceItemsNone' }}
                className="pf-u-p-lg"
            >
                <Title headingLevel="h2">Response method</Title>
                <div className="pf-u-mb-md pf-u-mt-sm">
                    Select a method to address violations of this policy.
                </div>
                <Form>
                    <FormGroup
                        fieldId="policy-response-method"
                        className="pf-u-mb-lg"
                        helperText={responseMethodHelperText}
                    >
                        <Flex direction={{ default: 'row' }}>
                            <Radio
                                label="Inform"
                                isChecked={!showEnforcement}
                                id="policy-response-inform"
                                name="inform"
                                onChange={() => {
                                    setShowEnforcement(false);
                                    setFieldValue('enforcementActions', [], false); // do not validate, because code changes the value
                                }}
                            />
                            <Radio
                                label="Inform and enforce"
                                isChecked={showEnforcement}
                                id="policy-response-inform-enforce"
                                name="enforce"
                                onChange={() => setShowEnforcement(true)}
                            />
                        </Flex>
                    </FormGroup>
                </Form>
                {showEnforcement && (
                    <>
                        <Title headingLevel="h2" className="pf-u-mt-md">
                            Configure enforcement behavior
                        </Title>
                        <div className="pf-u-mb-lg pf-u-mt-sm">
                            Based on the fields selected in your policy configuration, you may
                            choose to apply enforcement at the following stages.
                        </div>
                        <Grid hasGutter>
                            <GridItem span={4}>
                                <Card className="pf-u-h-100 policy-enforcement-card">
                                    <CardHeader>
                                        <CardTitle>Build</CardTitle>
                                        <CardActions>
                                            <Switch
                                                isChecked={hasEnforcementActionForLifecycleStage(
                                                    'BUILD',
                                                    values.enforcementActions
                                                )}
                                                isDisabled={!hasBuild}
                                                onChange={(isChecked) => {
                                                    onChangeEnforcementActions('BUILD', isChecked);
                                                }}
                                                label="Enforce on Build"
                                            />
                                        </CardActions>
                                    </CardHeader>
                                    <CardBody>
                                        If enabled, your CI builds will be failed when images
                                        violate this policy. Download the CLI to get started.
                                        <Flex
                                            justifyContent={{ default: 'justifyContentCenter' }}
                                            className="pf-u-pt-md"
                                        >
                                            <DownloadCLIDropdown hasBuild={hasBuild} />
                                        </Flex>
                                    </CardBody>
                                </Card>
                            </GridItem>
                            <GridItem span={4}>
                                <Card className="policy-enforcement-card">
                                    <CardHeader>
                                        <CardTitle>Deploy</CardTitle>
                                        <CardActions>
                                            <Switch
                                                isChecked={hasEnforcementActionForLifecycleStage(
                                                    'DEPLOY',
                                                    values.enforcementActions
                                                )}
                                                isDisabled={!hasDeploy}
                                                onChange={(isChecked) => {
                                                    onChangeEnforcementActions('DEPLOY', isChecked);
                                                }}
                                                label="Enforce on Deploy"
                                            />
                                        </CardActions>
                                    </CardHeader>
                                    <CardBody>
                                        If enabled, creation of deployments that violate this policy
                                        will be blocked. In clusters with the admission controller
                                        enabled, the Kubernetes API server will block deployments
                                        that violate this policy to prevent pods from being
                                        scheduled.
                                    </CardBody>
                                </Card>
                            </GridItem>
                            <GridItem span={4}>
                                <Card className="policy-enforcement-card">
                                    <CardHeader>
                                        <CardTitle>Runtime</CardTitle>
                                        <CardActions>
                                            <Switch
                                                isChecked={hasEnforcementActionForLifecycleStage(
                                                    'RUNTIME',
                                                    values.enforcementActions
                                                )}
                                                isDisabled={!hasRuntime}
                                                onChange={(isChecked) => {
                                                    onChangeEnforcementActions(
                                                        'RUNTIME',
                                                        isChecked
                                                    );
                                                }}
                                                label="Enforce on Runtime"
                                            />
                                        </CardActions>
                                    </CardHeader>
                                    <CardBody>
                                        If enabled, executions within a pod that violate this policy
                                        will result in the pod being killed. Actions taken through
                                        the API server that violate this policy will be blocked.
                                    </CardBody>
                                </Card>
                            </GridItem>
                        </Grid>
                    </>
                )}
            </Flex>
        </Flex>
    );
}

export default PolicyBehaviorForm;
