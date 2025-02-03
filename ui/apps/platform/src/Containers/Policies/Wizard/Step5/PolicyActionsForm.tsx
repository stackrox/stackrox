import React from 'react';
import { Flex, FlexItem, Title, Divider, Form, FormGroup, Radio } from '@patternfly/react-core';
import { FormikContextType, useFormikContext } from 'formik';

import { ClientPolicy } from 'types/policy.proto';

import PolicyEnforcementForm from './PolicyEnforcementForm';
import NotifiersForm from './NotifiersForm';

function PolicyActionsForm() {
    const { setFieldValue, values }: FormikContextType<ClientPolicy> = useFormikContext();
    return (
        <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsNone' }}>
            <FlexItem flex={{ default: 'flex_1' }} className="pf-v5-u-p-lg">
                <Title headingLevel="h2">Actions</Title>
                <div className="pf-v5-u-mt-sm">
                    Configure activation state, enforcement, and notifiers of this policy.
                </div>
            </FlexItem>
            <Divider component="div" />
            <Flex direction={{ default: 'column' }} className="pf-v5-u-p-lg">
                <Flex>
                    <FlexItem flex={{ default: 'flex_1' }}>
                        <Title headingLevel="h3">Activation state</Title>
                        <div className="pf-v5-u-mt-sm">
                            Select whether to enable or disable the policy.
                        </div>
                    </FlexItem>
                </Flex>
                <FlexItem>
                    <Form>
                        <FormGroup fieldId="policy-enablement">
                            <Flex direction={{ default: 'row' }}>
                                <Radio
                                    label="Enable"
                                    isChecked={!values.disabled}
                                    id="policy-enabled"
                                    name="enable"
                                    onChange={() => {
                                        setFieldValue('disabled', false);
                                    }}
                                />
                                <Radio
                                    label="Disable"
                                    isChecked={values.disabled}
                                    id="policy-disabled"
                                    name="disable"
                                    onChange={() => {
                                        setFieldValue('disabled', true);
                                    }}
                                />
                            </Flex>
                        </FormGroup>
                    </Form>
                </FlexItem>
            </Flex>
            <Divider component="div" />
            <Flex direction={{ default: 'column' }} className="pf-v5-u-p-lg">
                <Flex>
                    <FlexItem flex={{ default: 'flex_1' }}>
                        <Title headingLevel="h3">Enforcement</Title>
                        <div className="pf-v5-u-mt-sm">
                            Select a method to address violations of this policy
                        </div>
                    </FlexItem>
                </Flex>
                <FlexItem>
                    <PolicyEnforcementForm />
                </FlexItem>
            </Flex>
            <Divider component="div" />
            <Flex direction={{ default: 'column' }} className="pf-v5-u-p-lg">
                <Flex>
                    <FlexItem flex={{ default: 'flex_1' }}>
                        <Title headingLevel="h3">Notifiers</Title>
                        <div className="pf-v5-u-mt-sm">
                            Forward policy violations to external tooling by selecting one or more
                            notifiers from existing integrations.
                        </div>
                    </FlexItem>
                </Flex>
                <FlexItem>
                    <NotifiersForm />
                </FlexItem>
            </Flex>
        </Flex>
    );
}

export default PolicyActionsForm;
