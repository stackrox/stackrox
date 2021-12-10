import React from 'react';
import { Title, Flex, FlexItem, Divider, DragDrop } from '@patternfly/react-core';
import { useFormikContext } from 'formik';

import {
    policyConfigurationDescriptor,
    networkDetectionDescriptor,
    auditLogDescriptor,
    Descriptor,
} from 'Containers/Policies/Wizard/Form/descriptors';
import { Policy } from 'types/policy.proto';
import PolicyCriteriaKeys from './PolicyCriteriaKeys';

function PolicyCriteriaForm() {
    const [descriptor, setDescriptor] = React.useState<Descriptor[]>([]);
    const { values } = useFormikContext<Policy>();

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
                    <Title headingLevel="h2">Policy criteria</Title>
                    <div className="pf-u-mt-sm">
                        Construct policy rules by chaining criteria together with boolean logic.
                    </div>
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
