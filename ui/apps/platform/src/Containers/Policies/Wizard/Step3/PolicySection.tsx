import React, { useState } from 'react';
import { useFormikContext } from 'formik';
import {
    Card,
    CardHeader,
    CardTitle,
    CardBody,
    Button,
    Divider,
    Flex,
    FlexItem,
    TextInput,
} from '@patternfly/react-core';
import { PencilAltIcon, TrashIcon, CheckIcon } from '@patternfly/react-icons';

import useFeatureFlags from 'hooks/useFeatureFlags';
import useModal from 'hooks/useModal';
import type { Policy } from 'types/policy.proto';
import type { Descriptor } from './policyCriteriaDescriptors';
import PolicyGroupCard from './PolicyGroupCard';
import PolicySectionDropTarget from './PolicySectionDropTarget';
import PolicyCriteriaModal from './PolicyCriteriaModal';

import './PolicySection.css';

type PolicySectionProps = {
    sectionIndex: number;
    descriptors: Descriptor[];
    readOnly?: boolean;
};

function PolicySection({ sectionIndex, descriptors, readOnly = false }: PolicySectionProps) {
    const [isEditingName, setIsEditingName] = useState(false);
    const { isModalOpen, openModal, closeModal } = useModal();
    const { values, setFieldValue, handleChange } = useFormikContext<Policy>();
    const { sectionName, policyGroups } = values.policySections[sectionIndex];

    const { isFeatureFlagEnabled } = useFeatureFlags();
    const showPolicyCriteriaModal = isFeatureFlagEnabled('ROX_POLICY_CRITERIA_MODAL');

    function onEditSectionName(_, e) {
        handleChange(e);
    }

    function onDeleteSection() {
        setFieldValue(
            'policySections',
            values.policySections.filter((_, i) => i !== sectionIndex)
        );
    }

    function addPolicyFieldCardHandler(fieldCard) {
        setFieldValue(`policySections[${sectionIndex.toString()}].policyGroups`, [
            ...policyGroups,
            fieldCard,
        ]);
    }

    return (
        <>
            <Card isFlat isCompact className={!readOnly ? 'policy-section-card' : ''}>
                <CardHeader
                    {...(!readOnly && {
                        actions: {
                            actions: (
                                <>
                                    <Button
                                        variant="plain"
                                        className="pf-v5-u-px-sm"
                                        onClick={() => setIsEditingName(!isEditingName)}
                                        title={
                                            isEditingName
                                                ? 'Save name of policy section'
                                                : 'Edit name of policy section'
                                        }
                                    >
                                        {isEditingName ? <CheckIcon /> : <PencilAltIcon />}
                                    </Button>
                                    <Divider
                                        component="div"
                                        orientation={{ default: 'vertical' }}
                                    />
                                    <Button
                                        variant="plain"
                                        className="pf-v5-u-mr-xs pf-v5-u-px-sm pf-v5-u-py-md"
                                        title="Delete policy section"
                                        onClick={onDeleteSection}
                                    >
                                        <TrashIcon />
                                    </Button>
                                </>
                            ),
                            hasNoOffset: true,
                            className: undefined,
                        },
                    })}
                    className="policy-section-card-header pf-v5-u-p-0"
                >
                    <CardTitle className="pf-v5-u-display-flex pf-v5-u-align-self-stretch">
                        <Flex
                            alignItems={{ default: 'alignItemsCenter' }}
                            flexWrap={{ default: 'nowrap' }}
                        >
                            <FlexItem className="pf-v5-u-pl-md">{sectionIndex + 1}</FlexItem>
                            <Divider component="div" orientation={{ default: 'vertical' }} />
                            <FlexItem>
                                {isEditingName ? (
                                    <TextInput
                                        id={`policySections[${sectionIndex}].sectionName`}
                                        name={`policySections[${sectionIndex}].sectionName`}
                                        value={values.policySections[sectionIndex].sectionName}
                                        onChange={(e, _) => onEditSectionName(_, e)}
                                    />
                                ) : (
                                    <div
                                        className="pf-v5-u-py-sm"
                                        data-testid="policy-section-name"
                                    >
                                        {sectionName}
                                    </div>
                                )}
                            </FlexItem>
                        </Flex>
                    </CardTitle>
                </CardHeader>
                <CardBody className="policy-section-card-body">
                    {policyGroups.map((group, groupIndex) => {
                        const descriptor = descriptors.find(
                            (descriptorField) =>
                                group.fieldName === descriptorField.name ||
                                group.fieldName === descriptorField.label
                        );
                        return (
                            descriptor && (
                                <PolicyGroupCard
                                    key={descriptor.name}
                                    descriptor={descriptor}
                                    groupIndex={groupIndex}
                                    sectionIndex={sectionIndex}
                                    readOnly={readOnly}
                                />
                            )
                        );
                    })}
                    {!showPolicyCriteriaModal && !readOnly && (
                        <PolicySectionDropTarget
                            sectionIndex={sectionIndex}
                            descriptors={descriptors}
                        />
                    )}
                    {showPolicyCriteriaModal && !readOnly && (
                        <Flex
                            className="pf-v5-u-mt-md"
                            justifyContent={{ default: 'justifyContentCenter' }}
                        >
                            <FlexItem>
                                <Button
                                    key={`policySections[${sectionIndex}].sectionName-add-policy-field`}
                                    variant="secondary"
                                    onClick={openModal}
                                >
                                    Add policy field
                                </Button>
                            </FlexItem>
                        </Flex>
                    )}
                </CardBody>
            </Card>
            {showPolicyCriteriaModal && (
                <PolicyCriteriaModal
                    descriptors={descriptors}
                    existingGroups={policyGroups}
                    isModalOpen={isModalOpen}
                    onClose={closeModal}
                    addPolicyFieldCardHandler={addPolicyFieldCardHandler}
                />
            )}
        </>
    );
}

export default PolicySection;
