import React from 'react';
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
import intersection from 'lodash/intersection';

import { Policy, EnforcementAction, LifecycleStage } from 'types/policy.proto';
import DownloadCLIDropdown from './DownloadCLIDropdown';

const lifecycleToEnforcementsMap: Record<LifecycleStage, EnforcementAction[]> = {
    BUILD: ['FAIL_BUILD_ENFORCEMENT'],
    DEPLOY: ['SCALE_TO_ZERO_ENFORCEMENT', 'UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT'],
    RUNTIME: ['KILL_POD_ENFORCEMENT', 'FAIL_KUBE_REQUEST_ENFORCEMENT'],
};

function PolicyBehaviorForm() {
    const { values, setFieldValue } = useFormikContext<Policy>();
    const hasEnforcementActions =
        values.enforcementActions?.length > 0 &&
        !values.enforcementActions?.includes('UNSET_ENFORCEMENT');
    const [showEnforcement, setShowEnforcement] = React.useState(hasEnforcementActions);

    function onLifecycleStageChangeHandler(lifecycleStage: LifecycleStage) {
        return (isChecked) => {
            if (isChecked) {
                setFieldValue('lifecycleStages', [...values.lifecycleStages, lifecycleStage]);
            } else {
                setFieldValue(
                    'lifecycleStages',
                    values.lifecycleStages.filter((stage) => stage !== lifecycleStage)
                );
                if (lifecycleStage === 'RUNTIME') {
                    setFieldValue('eventSource', 'NOT_APPLICABLE');
                }
                if (lifecycleStage === 'BUILD') {
                    setFieldValue('excludedImageNames', []);
                }
                onEnforcementActionChangeHandler(lifecycleStage)(false);
            }
        };
    }

    function onEnforcementActionChangeHandler(lifecycleStage: LifecycleStage) {
        return (isChecked) => {
            if (isChecked) {
                setFieldValue('enforcementActions', [
                    ...values.enforcementActions,
                    ...lifecycleToEnforcementsMap[lifecycleStage],
                ]);
            } else {
                setFieldValue(
                    'enforcementActions',
                    values.enforcementActions.filter(
                        (action) => !lifecycleToEnforcementsMap[lifecycleStage].includes(action)
                    )
                );
            }
        };
    }

    function auditLogEventSourceChangeHandler() {
        setFieldValue('eventSource', 'AUDIT_LOG_EVENT');
        setFieldValue('excludedImageNames', []);
        values.scope.forEach(({ label, ...rest }, idx) => {
            if (label) {
                setFieldValue(`scope[${idx}]`, { ...rest });
            }
        });
        values.excludedDeploymentScopes.forEach(({ scope }, idx) => {
            const { label, ...rest } = scope || {};
            setFieldValue(`excludedDeploymentScopes[${idx}]`, {
                scope: {
                    ...rest,
                },
            });
        });
    }

    function hasEnforcementForLifecycle(lifecycleStage: LifecycleStage) {
        return (
            intersection(values.enforcementActions, lifecycleToEnforcementsMap[lifecycleStage])
                .length > 0
        );
    }

    const responseMethodHelperText = showEnforcement
        ? 'Inform and enforce will execute enforcement behavior at the stages you select.'
        : 'Inform will always include violations for this policy in the violations list.';

    const hasBuild = values.lifecycleStages.includes('BUILD');
    const hasDeploy = values.lifecycleStages.includes('DEPLOY');
    const hasRuntime = values.lifecycleStages.includes('RUNTIME');

    return (
        <div>
            <Title headingLevel="h2">Policy behavior</Title>
            <div className="pf-u-mt-sm">
                Select which stage of a container lifecycle this policy applies. Event sources can
                only be chosen for policies that apply at runtime.
            </div>
            <Alert variant="info" isInline title="Info" className="pf-u-mt-md pf-u-mb-md">
                <div className="pf-u-mb-sm">
                    Build-time policies apply to image fields such as CVEs and Dockerfile
                    instructions.
                </div>
                <div className="pf-u-mb-sm">
                    Deploy-time policies can include all build-time policy criteria but they can
                    also include data form your cluster configurations, such as running in
                    privileged mode or mounting the Docker socket.
                </div>
                <div>
                    Runtime policies can include all build-time and deploy-time policy criteria but
                    they can also include data about process executions during runtime.
                </div>
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
                            onChange={onLifecycleStageChangeHandler('BUILD')}
                        />
                        <Checkbox
                            label="Deploy"
                            isChecked={hasDeploy}
                            id="policy-lifecycle-stage-deploy"
                            onChange={onLifecycleStageChangeHandler('DEPLOY')}
                        />
                        <Checkbox
                            label="Runtime"
                            isChecked={hasRuntime}
                            id="policy-lifecycle-stage-runtime"
                            onChange={onLifecycleStageChangeHandler('RUNTIME')}
                        />
                    </Flex>
                </FormGroup>
                <FormGroup
                    fieldId="policy-event-source"
                    label="Event sources (Runtime lifecycle only)"
                    className="pf-u-mb-lg"
                >
                    <Flex direction={{ default: 'row' }}>
                        <Radio
                            label="Deployment"
                            isChecked={values.eventSource === 'DEPLOYMENT_EVENT'}
                            id="policy-event-source-deployment"
                            name="eventSource"
                            onChange={() => setFieldValue('eventSource', 'DEPLOYMENT_EVENT')}
                            isDisabled={!values.lifecycleStages.includes('RUNTIME')}
                        />
                        <Radio
                            label="Audit logs"
                            isChecked={values.eventSource === 'AUDIT_LOG_EVENT'}
                            id="policy-event-source-audit-logs"
                            name="eventSource"
                            onChange={auditLogEventSourceChangeHandler}
                            isDisabled={!values.lifecycleStages.includes('RUNTIME')}
                        />
                    </Flex>
                </FormGroup>
            </Form>
            <Divider component="div" />
            <Title headingLevel="h2" className="pf-u-mt-md">
                Response method
            </Title>
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
                                setFieldValue('enforcementActions', []);
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
                        Based on the fields selected in your policy configuration, you may choose to
                        apply enforcement at the following stages.
                    </div>
                    <Grid hasGutter>
                        <GridItem span={4}>
                            <Card className="pf-u-h-100">
                                <CardHeader>
                                    <CardTitle>Build</CardTitle>
                                    <CardActions>
                                        <Switch
                                            isChecked={hasEnforcementForLifecycle('BUILD')}
                                            isDisabled={!hasBuild}
                                            onChange={onEnforcementActionChangeHandler('BUILD')}
                                        />
                                    </CardActions>
                                </CardHeader>
                                <CardBody>
                                    If enabled, ACS will fail your CI builds when images match the
                                    conditions of this policy. Download the CLI to get started.
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
                            <Card>
                                <CardHeader>
                                    <CardTitle>Deploy</CardTitle>
                                    <CardActions>
                                        <Switch
                                            isChecked={hasEnforcementForLifecycle('DEPLOY')}
                                            isDisabled={!hasDeploy}
                                            onChange={onEnforcementActionChangeHandler('DEPLOY')}
                                        />
                                    </CardActions>
                                </CardHeader>
                                <CardBody>
                                    If enabled, ACS will automatically block creation of deployments
                                    that match the conditions of this policy. In clusters with the
                                    ACS admissions controller enabled, the Kubernetes API server
                                    will block noncompliant deployments to prevent pods from being
                                    scheduled.
                                </CardBody>
                            </Card>
                        </GridItem>
                        <GridItem span={4}>
                            <Card>
                                <CardHeader>
                                    <CardTitle>Runtime</CardTitle>
                                    <CardActions>
                                        <Switch
                                            isChecked={hasEnforcementForLifecycle('RUNTIME')}
                                            isDisabled={!hasRuntime}
                                            onChange={onEnforcementActionChangeHandler('RUNTIME')}
                                        />
                                    </CardActions>
                                </CardHeader>
                                <CardBody>
                                    If enabled, ACS will either kill the offending pod or block the
                                    action taken on the pod. Executions within a pod that match the
                                    conditions of the policy will result in the pod being killed.
                                    Actions taken through the API server that math policy criteria
                                    will be blocked.
                                </CardBody>
                            </Card>
                        </GridItem>
                    </Grid>
                </>
            )}
        </div>
    );
}

export default PolicyBehaviorForm;
