import React from 'react';
import { Flex } from '@patternfly/react-core';
import { useFormikContext } from 'formik';

import {
    policyConfigurationDescriptor,
    networkDetectionDescriptor,
    auditLogDescriptor,
    Descriptor,
} from 'Containers/Policies/Wizard/Form/descriptors';
import { Policy } from 'types/policy.proto';
import PolicySection from './PolicySection';

import './BooleanPolicyLogicSection.css';

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
            {values.policySections?.map((_, sectionIndex) => (
                // eslint-disable-next-line react/no-array-index-key
                <React.Fragment key={sectionIndex}>
                    <PolicySection
                        sectionIndex={sectionIndex}
                        descriptors={descriptor}
                        readOnly={readOnly}
                    />
                    {sectionIndex !== values.policySections.length - 1 && (
                        <Flex
                            alignSelf={{ default: 'alignSelfCenter' }}
                            alignItems={{ default: 'alignItemsCenter' }}
                            direction={{ default: 'column' }}
                            flexWrap={{ default: 'nowrap' }}
                            spaceItems={{ default: 'spaceItemsNone' }}
                            className="pf-u-h-100"
                        >
                            <div className="or-divider" />
                            OR
                            <div className="or-divider" />
                        </Flex>
                    )}
                </React.Fragment>
            ))}
        </>
    );
}

export default BooleanPolicyLogicSection;
