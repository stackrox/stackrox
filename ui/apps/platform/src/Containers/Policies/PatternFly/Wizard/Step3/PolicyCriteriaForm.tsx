import React from 'react';
import { Alert, Button, Divider, Flex, FlexItem, Title } from '@patternfly/react-core';
import { DndProvider } from 'react-dnd';
import { HTML5Backend } from 'react-dnd-html5-backend';
import { useFormikContext } from 'formik';

import {
    policyConfigurationDescriptor,
    networkDetectionDescriptor,
    auditLogDescriptor,
    Descriptor,
} from 'Containers/Policies/Wizard/Form/descriptors';
import { Policy } from 'types/policy.proto';
import PolicyCriteriaKeys from './PolicyCriteriaKeys';
import BooleanPolicyLogicSection from './BooleanPolicyLogicSection';

import './PolicyCriteriaForm.css';

const MAX_POLICY_SECTIONS = 16;

function PolicyCriteriaForm() {
    const [descriptor, setDescriptor] = React.useState<Descriptor[]>([]);
    const { values, setFieldValue } = useFormikContext<Policy>();
    const { criteriaLocked } = values;

    function addNewPolicySection() {
        if (values.policySections.length < MAX_POLICY_SECTIONS) {
            const newPolicySection = {
                sectionName: `Policy Section ${values.policySections.length + 1}`,
                policyGroups: [],
            };
            const newPolicySections = [...values.policySections, newPolicySection];
            setFieldValue('policySections', newPolicySections);
            // document.getElementById('policy-sections').scrollLeft += 20;
        }
    }

    React.useEffect(() => {
        if (values.eventSource === 'AUDIT_LOG_EVENT') {
            setDescriptor(auditLogDescriptor);
        } else {
            setDescriptor([...policyConfigurationDescriptor, ...networkDetectionDescriptor]);
        }
    }, [values.eventSource]);

    const headingElements = (
        <>
            <Title headingLevel="h2">Policy criteria</Title>
            <div className="pf-u-mt-sm">
                Construct policy rules by chaining criteria together with boolean logic.
            </div>
        </>
    );

    if (criteriaLocked) {
        return (
            <>
                {headingElements}
                <Alert
                    variant="info"
                    isInline
                    title="Editing policy criteria is disabled for system default policies"
                    className="pf-u-mt-sm pf-u-mb-md"
                >
                    If you need to edit policy criteria, clone this policy or create a new policy.
                </Alert>
                <BooleanPolicyLogicSection readOnly />
            </>
        );
    }

    return (
        <DndProvider backend={HTML5Backend}>
            <Flex>
                <FlexItem flex={{ default: 'flex_1' }} className="pf-u-w-66">
                    <Flex direction={{ default: 'row' }}>
                        <FlexItem flex={{ default: 'flex_1' }}>{headingElements}</FlexItem>
                        <FlexItem className="pf-u-pr-md" alignSelf={{ default: 'alignSelfCenter' }}>
                            <Button variant="secondary" onClick={addNewPolicySection}>
                                Add a new condition
                            </Button>
                        </FlexItem>
                    </Flex>
                    <Divider component="div" className="pf-u-mt-md pf-u-mb-lg" />
                    <Flex
                        direction={{ default: 'row' }}
                        flexWrap={{ default: 'nowrap' }}
                        id="policy-sections"
                    >
                        <BooleanPolicyLogicSection />
                    </Flex>
                </FlexItem>
                <Divider component="div" isVertical />
                <FlexItem>
                    <PolicyCriteriaKeys keys={descriptor} />
                </FlexItem>
            </Flex>
        </DndProvider>
    );
}

export default PolicyCriteriaForm;
