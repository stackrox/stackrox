import { useState } from 'react';
import { useFormikContext } from 'formik';
import {
    Button,
    Card,
    CardBody,
    CardHeader,
    CardTitle,
    Divider,
    Flex,
    FlexItem,
    TextInput,
} from '@patternfly/react-core';
import { CheckIcon, PencilAltIcon, TrashIcon } from '@patternfly/react-icons';

import type { Policy } from 'types/policy.proto';
import type { Descriptor } from './policyCriteriaDescriptors';
import PolicyGroupCard from './PolicyGroupCard';
import PolicySectionDropTarget from './PolicySectionDropTarget';

import './PolicySection.css';
import { PolicySectionValidationError } from './PolicySectionValidationError';

type PolicySectionProps = {
    sectionIndex: number;
    descriptors: Descriptor[];
    readOnly?: boolean;
};

function PolicySection({ sectionIndex, descriptors, readOnly = false }: PolicySectionProps) {
    const [isEditingName, setIsEditingName] = useState(false);
    const { values, errors, setFieldValue, handleChange } = useFormikContext<Policy>();
    const { sectionName, policyGroups } = values.policySections[sectionIndex];

    function onEditSectionName(_, e) {
        handleChange(e);
    }

    function onDeleteSection() {
        setFieldValue(
            'policySections',
            values.policySections.filter((_, i) => i !== sectionIndex)
        );
    }

    return (
        <>
            <Card isCompact className={!readOnly ? 'policy-section-card' : ''}>
                <CardHeader
                    {...(!readOnly && {
                        actions: {
                            actions: (
                                <>
                                    <Button
                                        icon={isEditingName ? <CheckIcon /> : <PencilAltIcon />}
                                        variant="plain"
                                        onClick={() => setIsEditingName((prev) => !prev)}
                                        title={
                                            isEditingName
                                                ? 'Save name of policy section'
                                                : 'Edit name of policy section'
                                        }
                                    />
                                    <Divider
                                        component="div"
                                        orientation={{ default: 'vertical' }}
                                    />
                                    <Button
                                        icon={<TrashIcon />}
                                        variant="plain"
                                        className="pf-v6-u-mr-md"
                                        title="Delete policy section"
                                        onClick={onDeleteSection}
                                    />
                                </>
                            ),
                            hasNoOffset: true,
                            className: 'pf-v6-u-py-sm',
                        },
                    })}
                    className="policy-section-card-header pf-v6-u-p-0"
                >
                    <CardTitle className="pf-v6-u-display-flex pf-v6-u-align-self-stretch">
                        <Flex
                            alignItems={{ default: 'alignItemsCenter' }}
                            flexWrap={{ default: 'nowrap' }}
                        >
                            <FlexItem className="pf-v6-u-pl-md">{sectionIndex + 1}</FlexItem>
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
                                        className="pf-v6-u-py-sm"
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
                    {errors.policySections && (
                        <PolicySectionValidationError
                            sectionIndex={sectionIndex}
                            errors={errors.policySections}
                            className="pf-v6-u-mb-md"
                        />
                    )}
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
                    {!readOnly && (
                        <PolicySectionDropTarget
                            sectionIndex={sectionIndex}
                            descriptors={descriptors}
                        />
                    )}
                </CardBody>
            </Card>
        </>
    );
}

export default PolicySection;
