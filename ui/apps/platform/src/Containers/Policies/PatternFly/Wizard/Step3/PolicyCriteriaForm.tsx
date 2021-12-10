import React from 'react';
import { Title, Flex, FlexItem, Divider, DragDrop, Button } from '@patternfly/react-core';
import { useFormikContext } from 'formik';

import {
    policyConfigurationDescriptor,
    networkDetectionDescriptor,
    auditLogDescriptor,
    Descriptor,
} from 'Containers/Policies/Wizard/Form/descriptors';
import { Policy } from 'types/policy.proto';
import PolicyCriteriaKeys from './PolicyCriteriaKeys';
import PolicySection from './PolicySection';

const MAX_POLICY_SECTIONS = 16;

function PolicyCriteriaForm() {
    const [descriptor, setDescriptor] = React.useState<Descriptor[]>([]);
    const { values, setFieldValue } = useFormikContext<Policy>();

    function addNewPolicySection() {
        if (values.policySections.length < MAX_POLICY_SECTIONS) {
            const newPolicySection = {
                sectionName: `Policy Section ${values.policySections.length + 1}`,
                policyGroups: [],
            };
            const newPolicySections = [...values.policySections, newPolicySection];
            setFieldValue('policySections', newPolicySections);
        }
    }

    React.useEffect(() => {
        if (values.eventSource === 'AUDIT_LOG_EVENT') {
            setDescriptor(auditLogDescriptor);
        } else {
            setDescriptor([...policyConfigurationDescriptor, ...networkDetectionDescriptor]);
        }
    }, [values.eventSource]);

    return (
        <DragDrop>
            <Flex>
                <FlexItem flex={{ default: 'flex_1' }}>
                    <Flex direction={{ default: 'row' }}>
                        <FlexItem flex={{ default: 'flex_1' }}>
                            <Title headingLevel="h2">Policy criteria</Title>
                            <div className="pf-u-mt-sm">
                                Construct policy rules by chaining criteria together with boolean
                                logic.
                            </div>
                        </FlexItem>
                        <FlexItem className="pf-u-pr-md" alignSelf={{ default: 'alignSelfCenter' }}>
                            <Button variant="secondary" onClick={addNewPolicySection}>
                                Add a new condition
                            </Button>
                        </FlexItem>
                    </Flex>
                    <Divider component="div" className="pf-u-mt-md" />
                    <Flex direction={{ default: 'row' }}>
                        {values.policySections.map((_, sectionIndex) => (
                            <PolicySection sectionIndex={sectionIndex} />
                        ))}
                    </Flex>
                </FlexItem>
                <Divider component="div" isVertical />
                <FlexItem>
                    <PolicyCriteriaKeys keys={descriptor} />
                </FlexItem>
            </Flex>
        </DragDrop>
    );
}

export default PolicyCriteriaForm;
