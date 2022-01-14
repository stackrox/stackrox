import React from 'react';
import { Flex, FlexItem } from '@patternfly/react-core';
import { useFormikContext } from 'formik';

import {
    policyConfigurationDescriptor,
    networkDetectionDescriptor,
    auditLogDescriptor,
    Descriptor,
} from 'Containers/Policies/Wizard/Form/descriptors';
import { Policy } from 'types/policy.proto';
import PolicySection from './PolicySection';

import './PolicyCriteriaForm.css';

type BooleanPolicyLogicSectionProps = {
    readOnly?: boolean;
};

function BooleanPolicyLogicSection({ readOnly = false }: BooleanPolicyLogicSectionProps) {
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
        <>
            {values.policySections.map((_, sectionIndex) => (
                // eslint-disable-next-line react/no-array-index-key
                <React.Fragment key={sectionIndex}>
                    <PolicySection
                        sectionIndex={sectionIndex}
                        descriptors={descriptor}
                        readOnly={readOnly}
                    />
                    {sectionIndex !== values.policySections.length - 1 && (
                        <Flex alignSelf={{ default: 'alignSelfCenter' }}>
                            <FlexItem>or</FlexItem>
                        </Flex>
                    )}
                </React.Fragment>
            ))}
        </>
    );
}

export default BooleanPolicyLogicSection;
