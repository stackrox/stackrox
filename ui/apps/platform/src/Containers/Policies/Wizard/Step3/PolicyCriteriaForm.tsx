import React, { useState, CSSProperties } from 'react';
import { Alert, Button, Divider, Flex, FlexItem, Title } from '@patternfly/react-core';
import { DndProvider } from 'react-dnd';
import { HTML5Backend } from 'react-dnd-html5-backend';
import { useFormikContext } from 'formik';

import { Policy } from 'types/policy.proto';
import useFeatureFlags from 'hooks/useFeatureFlags';
import { getPolicyDescriptors } from 'Containers/Policies/policies.utils';
import PolicyCriteriaKeys from './PolicyCriteriaKeys';
import PolicyCriteriaOptions from './PolicyCriteriaOptions';
import BooleanPolicyLogicSection from './BooleanPolicyLogicSection';

import './PolicyCriteriaForm.css';

const policyCriteriaOptionsStyleConstant = {
    '--pf-v5-u-min-width--MinWidth': '35ch',
    '--pf-v5-u-max-width--MaxWidth': '35ch',
} as CSSProperties;

const MAX_POLICY_SECTIONS = 16;

type PolicyBehaviorFormProps = {
    hasActiveViolations: boolean;
};

function PolicyCriteriaForm({ hasActiveViolations }: PolicyBehaviorFormProps) {
    const [selectedSection, setSelectedSection] = useState(0);
    const { values, setFieldValue } = useFormikContext<Policy>();
    const { criteriaLocked } = values;
    const { isFeatureFlagEnabled } = useFeatureFlags();

    const showAccessiblePolicyCriteria = isFeatureFlagEnabled(
        'ROX_ACCESSIBLE_POLICY_CRITERIA_EDITING'
    );

    function addNewPolicySection() {
        if (values.policySections.length < MAX_POLICY_SECTIONS) {
            const newPolicySection = {
                sectionName: `Policy Section ${values.policySections.length + 1}`,
                policyGroups: [],
            };
            const newPolicySections = [...values.policySections, newPolicySection];
            setFieldValue('policySections', newPolicySections);
            setSelectedSection(newPolicySections.length - 1);
            // document.getElementById('policy-sections').scrollLeft += 20;
        }
    }

    const filteredDescriptors = getPolicyDescriptors(
        isFeatureFlagEnabled,
        values.eventSource,
        values.lifecycleStages
    );

    const headingElements = (
        <>
            <Title headingLevel="h2">Policy criteria</Title>
            <div className="pf-v5-u-mt-sm">Chain criteria with boolean logic.</div>
        </>
    );

    if (criteriaLocked || hasActiveViolations) {
        return (
            <Flex
                fullWidth={{ default: 'fullWidth' }}
                direction={{ default: 'column' }}
                spaceItems={{ default: 'spaceItemsNone' }}
                flexWrap={{ default: 'nowrap' }}
                className="pf-v5-u-h-100 pf-v5-u-p-lg"
                id="policy-sections-container"
            >
                {headingElements}
                {criteriaLocked ? (
                    <Alert
                        variant="info"
                        isInline
                        title="Editing policy criteria is disabled for system default policies"
                        component="p"
                        className="pf-v5-u-mt-sm pf-v5-u-mb-md"
                        data-testid="default-policy-alert"
                    >
                        If you need to edit policy criteria, clone this policy or create a new
                        policy.
                    </Alert>
                ) : (
                    <Alert
                        variant="warning"
                        isInline
                        title="This policy has active violations, and the policy criteria cannot be changed. To update criteria, disable the policy first."
                        component="p"
                        className="pf-v5-u-mt-sm pf-v5-u-mb-md"
                        data-testid="active-violations-policy-alert"
                    />
                )}
                <BooleanPolicyLogicSection readOnly />
            </Flex>
        );
    }

    return showAccessiblePolicyCriteria ? (
        <Flex
            direction={{ default: 'column' }}
            className="pf-v5-u-h-100"
            spaceItems={{ default: 'spaceItemsNone' }}
            fullWidth={{ default: 'fullWidth' }}
            flexWrap={{ default: 'nowrap' }}
        >
            <FlexItem span={12}>
                <Flex direction={{ default: 'row' }} className="pf-v5-u-p-lg">
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
            </FlexItem>
            <FlexItem>
                <Divider component="div" />
            </FlexItem>
            <FlexItem span={12}>
                <Flex flexWrap={{ default: 'nowrap' }} direction={{ default: 'row' }} className="">
                    <FlexItem
                        flex={{ default: 'flex_1' }}
                        className="pf-v5-u-pr-0 pf-v5-u-mr-0"
                        style={{ overflow: 'auto' }}
                    >
                        <Flex
                            direction={{ default: 'column', lg: 'row' }}
                            flexWrap={{ default: 'nowrap' }}
                            id="policy-sections"
                            className="pf-v5-u-p-lg pf-v5-u-h-100 pf-v5-u-align-items-stretch"
                        >
                            <BooleanPolicyLogicSection
                                selectedSection={selectedSection}
                                onChangeSelected={setSelectedSection}
                            />
                        </Flex>
                    </FlexItem>
                    <Divider component="div" orientation={{ default: 'vertical' }} />
                    <FlexItem
                        alignSelf={{ default: 'alignSelfCenter' }}
                        className="pf-v5-u-h-100 pf-v5-u-pt-lg pf-v5-u-min-width pf-v5-u-max-width"
                        style={policyCriteriaOptionsStyleConstant}
                    >
                        <PolicyCriteriaOptions
                            descriptors={filteredDescriptors}
                            selectedSectionIndex={selectedSection}
                        />
                    </FlexItem>
                </Flex>
            </FlexItem>
        </Flex>
    ) : (
        /*
        (dv 2024-05-01) Upgrading to React types 18 causes a type error below

        @ts-expect-error DndProvider types do not expect children as props */
        <DndProvider backend={HTML5Backend}>
            <Flex fullWidth={{ default: 'fullWidth' }} className="pf-v5-u-h-100">
                <Flex
                    flex={{ default: 'flex_2' }}
                    direction={{ default: 'column' }}
                    className="pf-v5-u-h-100"
                    spaceItems={{ default: 'spaceItemsNone' }}
                    fullWidth={{ default: 'fullWidth' }}
                    flexWrap={{ default: 'nowrap' }}
                    id="policy-sections-container"
                >
                    <Flex direction={{ default: 'row' }} className="pf-v5-u-p-lg">
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
                        className="pf-v5-u-p-lg pf-v5-u-h-100"
                    >
                        <BooleanPolicyLogicSection />
                    </Flex>
                </Flex>
                <Divider component="div" orientation={{ default: 'vertical' }} />
                <Flex className="pf-v5-u-h-100 pf-v5-u-pt-lg" id="policy-criteria-keys-container">
                    <PolicyCriteriaKeys keys={filteredDescriptors} />
                </Flex>
            </Flex>
        </DndProvider>
    );
}

export default PolicyCriteriaForm;
