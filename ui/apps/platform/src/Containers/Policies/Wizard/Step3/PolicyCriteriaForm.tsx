import React from 'react';
import { Alert, Button, Divider, Flex, FlexItem, Title } from '@patternfly/react-core';
import { DndProvider } from 'react-dnd';
import { HTML5Backend } from 'react-dnd-html5-backend';
import { useFormikContext } from 'formik';

import { Policy } from 'types/policy.proto';
import useFeatureFlags from 'hooks/useFeatureFlags';
import { policyConfigurationDescriptor, auditLogDescriptor } from './policyCriteriaDescriptors';
import PolicyCriteriaKeys from './PolicyCriteriaKeys';
import BooleanPolicyLogicSection from './BooleanPolicyLogicSection';

import './PolicyCriteriaForm.css';

const MAX_POLICY_SECTIONS = 16;

function PolicyCriteriaForm() {
    const { values, setFieldValue } = useFormikContext<Policy>();
    const { criteriaLocked } = values;
    const { isFeatureFlagEnabled } = useFeatureFlags();

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

    const unfilteredDescriptors =
        values.eventSource === 'AUDIT_LOG_EVENT'
            ? auditLogDescriptor
            : policyConfigurationDescriptor;
    const descriptors = unfilteredDescriptors.filter((unfilteredDescriptor) => {
        if (typeof unfilteredDescriptor.featureFlagDependency === 'string') {
            return isFeatureFlagEnabled(unfilteredDescriptor.featureFlagDependency);
        }
        return true;
    });

    const headingElements = (
        <>
            <Title headingLevel="h2">Policy criteria</Title>
            <div className="pf-u-mt-sm">Chain criteria with boolean logic.</div>
        </>
    );

    if (criteriaLocked) {
        return (
            <Flex
                fullWidth={{ default: 'fullWidth' }}
                direction={{ default: 'column' }}
                spaceItems={{ default: 'spaceItemsNone' }}
                flexWrap={{ default: 'nowrap' }}
                className="pf-u-h-100 pf-u-p-lg"
                id="policy-sections-container"
            >
                {headingElements}
                <Alert
                    variant="info"
                    isInline
                    title="Editing policy criteria is disabled for system default policies"
                    className="pf-u-mt-sm pf-u-mb-md"
                    data-testid="default-policy-alert"
                >
                    If you need to edit policy criteria, clone this policy or create a new policy.
                </Alert>
                <BooleanPolicyLogicSection readOnly />
            </Flex>
        );
    }

    return (
        <DndProvider backend={HTML5Backend}>
            <Flex fullWidth={{ default: 'fullWidth' }} className="pf-u-h-100">
                <Flex
                    flex={{ default: 'flex_1' }}
                    direction={{ default: 'column' }}
                    className="pf-u-h-100"
                    spaceItems={{ default: 'spaceItemsNone' }}
                    fullWidth={{ default: 'fullWidth' }}
                    flexWrap={{ default: 'nowrap' }}
                    id="policy-sections-container"
                >
                    <Flex direction={{ default: 'row' }} className="pf-u-p-lg">
                        <FlexItem flex={{ default: 'flex_1' }}>{headingElements}</FlexItem>
                        <FlexItem alignSelf={{ default: 'alignSelfCenter' }}>
                            <Button
                                variant="secondary"
                                onClick={addNewPolicySection}
                                data-testid="add-section-btn"
                            >
                                Add condition
                            </Button>
                        </FlexItem>
                    </Flex>
                    <Divider component="div" />
                    <Flex
                        direction={{ default: 'column', lg: 'row' }}
                        flexWrap={{ default: 'nowrap' }}
                        id="policy-sections"
                        className="pf-u-p-lg pf-u-h-100"
                    >
                        <BooleanPolicyLogicSection />
                    </Flex>
                </Flex>
                <Divider component="div" isVertical />
                <Flex className="pf-u-h-100 pf-u-pt-lg" id="policy-criteria-keys-container">
                    <PolicyCriteriaKeys keys={descriptors} />
                </Flex>
            </Flex>
        </DndProvider>
    );
}

export default PolicyCriteriaForm;
