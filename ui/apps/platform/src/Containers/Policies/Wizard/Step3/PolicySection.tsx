import React from 'react';
import { useFormikContext } from 'formik';
import {
    Card,
    CardHeader,
    CardTitle,
    CardBody,
    Button,
    Divider,
    EmptyState,
    EmptyStateBody,
    EmptyStateHeader,
    EmptyStateIcon,
    Flex,
    FlexItem,
    TextInput,
} from '@patternfly/react-core';
import { PencilAltIcon, TrashIcon, CheckIcon, PlusCircleIcon } from '@patternfly/react-icons';

import useFeatureFlags from 'hooks/useFeatureFlags';
import useModal from 'hooks/useModal';
import { Policy } from 'types/policy.proto';
import { Descriptor } from './policyCriteriaDescriptors';
import PolicyGroupCard from './PolicyGroupCard';
import PolicySectionDropTarget from './PolicySectionDropTarget';
import PolicyCriteriaModal from './PolicyCriteriaModal';

import './PolicySection.css';

type PolicySectionProps = {
    sectionIndex: number;
    onChangeSelected: (sectionIndex: number) => void;
    descriptors: Descriptor[];
    readOnly?: boolean;
    selectedSectionIndex?: number;
};

function PolicySection({
    sectionIndex,
    onChangeSelected,
    descriptors,
    readOnly = false,
    selectedSectionIndex = -1,
}: PolicySectionProps) {
    const [isEditingName, setIsEditingName] = React.useState(false);
    const { isModalOpen, openModal, closeModal } = useModal();
    const { values, setFieldValue, handleChange } = useFormikContext<Policy>();
    const { sectionName, policyGroups } = values.policySections[sectionIndex];

    const { isFeatureFlagEnabled } = useFeatureFlags();
    const showPolicyCriteriaModal = isFeatureFlagEnabled('ROX_POLICY_CRITERIA_MODAL');
    const showAccessiblePolicyCriteria = isFeatureFlagEnabled(
        'ROX_ACCESSIBLE_POLICY_CRITERIA_EDITING'
    );

    const isSelected = sectionIndex === selectedSectionIndex;

    function onEditSectionName(_, e) {
        handleChange(e);
    }

    function onDeleteSection() {
        const newSelectedSection = selectedSectionIndex - 1;
        onChangeSelected(newSelectedSection);

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
            <Card
                isFlat
                isCompact
                isSelected={isSelected}
                isSelectable={showAccessiblePolicyCriteria}
                className={!readOnly ? 'policy-section-card' : ''}
            >
                <CardHeader
                    {...(!readOnly && {
                        actions: {
                            actions: (
                                <>
                                    <Button
                                        variant="plain"
                                        className="pf-v5-u-px-sm"
                                        onClick={() => setIsEditingName(!isEditingName)}
                                    >
                                        {isEditingName ? (
                                            <CheckIcon data-testid="save-section-name-btn" />
                                        ) : (
                                            <PencilAltIcon data-testid="edit-section-name-btn" />
                                        )}
                                    </Button>
                                    <Divider
                                        component="div"
                                        orientation={{ default: 'vertical' }}
                                    />
                                    <Button
                                        variant="plain"
                                        className="pf-v5-u-mr-xs pf-v5-u-px-sm pf-v5-u-py-md"
                                        data-testid="delete-section-btn"
                                        onClick={onDeleteSection}
                                    >
                                        <TrashIcon />
                                    </Button>
                                </>
                            ),
                            hasNoOffset: true,
                            className: undefined,
                        },
                        selectableActions: {
                            selectableActionId: `policy-section-${sectionIndex}`,
                            selectableActionAriaLabel: `Policy section ${sectionIndex + 1}: ${sectionName}`,
                            variant: 'single',
                            onChange: () => {
                                onChangeSelected(sectionIndex);
                            },
                            isChecked: isSelected,
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
                    {showAccessiblePolicyCriteria && !readOnly && (
                        <EmptyState>
                            <EmptyStateHeader
                                titleText=""
                                icon={<EmptyStateIcon icon={PlusCircleIcon} />}
                            />
                            <EmptyStateBody>
                                Add a policy criterion from the panel to the right
                            </EmptyStateBody>
                        </EmptyState>
                    )}
                    {!showPolicyCriteriaModal && !showAccessiblePolicyCriteria && !readOnly && (
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
