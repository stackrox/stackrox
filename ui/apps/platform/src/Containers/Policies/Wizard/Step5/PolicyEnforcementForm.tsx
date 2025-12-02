import { useState } from 'react';
import {
    Card,
    CardBody,
    CardHeader,
    CardTitle,
    Flex,
    Form,
    FormGroup,
    FormHelperText,
    Grid,
    GridItem,
    HelperText,
    HelperTextItem,
    Radio,
    Switch,
    Title,
} from '@patternfly/react-core';
import { useFormikContext } from 'formik';

import type { ClientPolicy, LifecycleStage } from 'types/policy.proto';

import DownloadCLIDropdown from './DownloadCLIDropdown';
import {
    appendEnforcementActionsForAddedLifecycleStage,
    filterEnforcementActionsForRemovedLifecycleStage,
    hasEnforcementActionForLifecycleStage,
} from '../../policies.utils';
import './PolicyEnforcementForm.css';

function PolicyEnforcementForm() {
    const { setFieldValue, values } = useFormikContext<ClientPolicy>();

    const hasEnforcementActions =
        values.enforcementActions?.length > 0 &&
        !values.enforcementActions?.includes('UNSET_ENFORCEMENT');
    const [showEnforcement, setShowEnforcement] = useState(hasEnforcementActions);

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

    const hasBuild = values.lifecycleStages.includes('BUILD');
    const hasDeploy = values.lifecycleStages.includes('DEPLOY');
    const hasRuntime = values.lifecycleStages.includes('RUNTIME');
    const hasAuditLog = values.eventSource === 'AUDIT_LOG_EVENT';
    const hasNodeEvent = values.eventSource === 'NODE_EVENT';

    let responseMethodHelperText = showEnforcement
        ? 'Inform and enforce will execute enforcement behavior at the stages you select.'
        : 'Inform will always include violations for this policy in the violations list.';

    if (hasAuditLog) {
        responseMethodHelperText = 'Enforcement is not available for audit log event sources.';
    }
    if (hasNodeEvent) {
        responseMethodHelperText = 'Enforcement is not available for node event sources.';
    }

    const isEnforcementDisabled = hasAuditLog || hasNodeEvent;

    return (
        <Form>
            <FormGroup fieldId="policy-response-method">
                <Flex direction={{ default: 'row' }}>
                    <Radio
                        label="Inform"
                        isChecked={!showEnforcement}
                        id="policy-response-inform"
                        name="inform"
                        isDisabled={isEnforcementDisabled}
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
                        isDisabled={isEnforcementDisabled}
                        onChange={() => setShowEnforcement(true)}
                    />
                </Flex>
                <FormHelperText>
                    <HelperText>
                        <HelperTextItem>{responseMethodHelperText}</HelperTextItem>
                    </HelperText>
                </FormHelperText>
            </FormGroup>
            {showEnforcement && (
                <div>
                    <Title headingLevel="h2" className="pf-v6-u-mt-md">
                        Configure enforcement behavior
                    </Title>
                    <div className="pf-v6-u-mb-lg pf-v6-u-mt-sm">
                        Based on the fields selected in your policy configuration, you may choose to
                        apply enforcement at the following stages.
                    </div>
                    <Grid hasGutter>
                        <GridItem span={4}>
                            <Card className="pf-v6-u-h-100 policy-enforcement-card">
                                <CardHeader>
                                    <CardTitle component="h3">Build</CardTitle>
                                </CardHeader>
                                <CardBody>
                                    <Switch
                                        isChecked={hasEnforcementActionForLifecycleStage(
                                            'BUILD',
                                            values.enforcementActions
                                        )}
                                        isDisabled={!hasBuild}
                                        onChange={(_event, isChecked) => {
                                            onChangeEnforcementActions('BUILD', isChecked);
                                        }}
                                        label="Enforce on Build"
                                    />
                                    <p className="pf-v6-u-pt-md pf-v6-u-pb-md">
                                        If enabled, your CI builds will be failed when images
                                        violate this policy. Download the CLI to get started.
                                    </p>
                                    <DownloadCLIDropdown hasBuild={hasBuild} />
                                </CardBody>
                            </Card>
                        </GridItem>
                        <GridItem span={4}>
                            <Card className="policy-enforcement-card">
                                <CardHeader>
                                    <CardTitle component="h3">Deploy</CardTitle>
                                </CardHeader>
                                <CardBody>
                                    <Switch
                                        isChecked={hasEnforcementActionForLifecycleStage(
                                            'DEPLOY',
                                            values.enforcementActions
                                        )}
                                        isDisabled={!hasDeploy}
                                        onChange={(_event, isChecked) => {
                                            onChangeEnforcementActions('DEPLOY', isChecked);
                                        }}
                                        label="Enforce on Deploy"
                                    />
                                    <p className="pf-v6-u-pt-md">
                                        If enabled, creation of deployments that violate this policy
                                        will be blocked. In clusters with the admission controller
                                        enabled, the Kubernetes API server will block deployments
                                        that violate this policy to prevent pods from being
                                        scheduled.
                                    </p>
                                </CardBody>
                            </Card>
                        </GridItem>
                        <GridItem span={4}>
                            <Card className="policy-enforcement-card">
                                <CardHeader>
                                    <CardTitle component="h3">Runtime</CardTitle>
                                </CardHeader>
                                <CardBody>
                                    <Switch
                                        isChecked={hasEnforcementActionForLifecycleStage(
                                            'RUNTIME',
                                            values.enforcementActions
                                        )}
                                        isDisabled={!hasRuntime || isEnforcementDisabled}
                                        onChange={(_event, isChecked) => {
                                            onChangeEnforcementActions('RUNTIME', isChecked);
                                        }}
                                        label="Enforce on Runtime"
                                    />
                                    <p className="pf-v6-u-pt-md">
                                        If enabled, executions within a pod that violate this policy
                                        will result in the pod being deleted. Actions taken through
                                        the API server that violate this policy will be blocked.
                                    </p>
                                </CardBody>
                            </Card>
                        </GridItem>
                    </Grid>
                </div>
            )}
        </Form>
    );
}

export default PolicyEnforcementForm;
