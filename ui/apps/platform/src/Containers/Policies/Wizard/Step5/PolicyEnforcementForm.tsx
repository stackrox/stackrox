import React, { useState } from 'react';
import {
    Card,
    CardBody,
    CardHeader,
    CardTitle,
    Flex,
    Form,
    FormGroup,
    Grid,
    GridItem,
    Radio,
    Switch,
    Title,
    FormHelperText,
    HelperText,
    HelperTextItem,
} from '@patternfly/react-core';
import { useFormikContext } from 'formik';

import { ClientPolicy, LifecycleStage } from 'types/policy.proto';

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

    const responseMethodHelperText = showEnforcement
        ? 'Inform and enforce will execute enforcement behavior at the stages you select.'
        : 'Inform will always include violations for this policy in the violations list.';

    const hasBuild = values.lifecycleStages.includes('BUILD');
    const hasDeploy = values.lifecycleStages.includes('DEPLOY');
    const hasRuntime = values.lifecycleStages.includes('RUNTIME');

    return (
        <Form>
            <FormGroup fieldId="policy-response-method">
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
                <FormHelperText>
                    <HelperText>
                        <HelperTextItem>{responseMethodHelperText}</HelperTextItem>
                    </HelperText>
                </FormHelperText>
            </FormGroup>
            {showEnforcement && (
                <div>
                    <Title headingLevel="h2" className="pf-v5-u-mt-md">
                        Configure enforcement behavior
                    </Title>
                    <div className="pf-v5-u-mb-lg pf-v5-u-mt-sm">
                        Based on the fields selected in your policy configuration, you may choose to
                        apply enforcement at the following stages.
                    </div>
                    <Grid hasGutter>
                        <GridItem span={4}>
                            <Card className="pf-v5-u-h-100 policy-enforcement-card">
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
                                    <p className="pf-v5-u-pt-md pf-v5-u-pb-md">
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
                                    <p className="pf-v5-u-pt-md">
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
                                        isDisabled={!hasRuntime}
                                        onChange={(_event, isChecked) => {
                                            onChangeEnforcementActions('RUNTIME', isChecked);
                                        }}
                                        label="Enforce on Runtime"
                                    />
                                    <p className="pf-v5-u-pt-md">
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
