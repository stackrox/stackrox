import React from 'react';
import {
    Card,
    CardHeader,
    CardTitle,
    CardActions,
    CardBody,
    Divider,
    Flex,
    FlexItem,
    Button,
    Checkbox,
} from '@patternfly/react-core';
import { TrashIcon, PlusIcon } from '@patternfly/react-icons';
import { useFormikContext } from 'formik';

import { Descriptor } from 'Containers/Policies/Wizard/Form/descriptors';
import { Policy } from 'types/policy.proto';
import FieldValue from './FieldValue';
import AndOrOperatorField from './AndOrOperatorField';

type PolicyGroupCardProps = {
    descriptor: Descriptor;
    groupIndex: number;
    sectionIndex: number;
    readOnly?: boolean;
};

function PolicyGroupCard({
    descriptor,
    groupIndex,
    sectionIndex,
    readOnly = false,
}: PolicyGroupCardProps) {
    const { values, setFieldValue } = useFormikContext<Policy>();
    const { policyGroups } = values.policySections[sectionIndex];
    const group = policyGroups[groupIndex];

    function onDeleteGroup() {
        setFieldValue(
            `policySections[${sectionIndex}].policyGroups`,
            policyGroups.filter((_, idx) => idx !== groupIndex)
        );
    }

    function handleNegate() {
        setFieldValue(`policySections[${sectionIndex}].policyGroups[${groupIndex}]`, {
            ...group,
            negate: !group.negate,
        });
    }

    function handleRemoveValue(valueIndex) {
        return () => {
            setFieldValue(
                `policySections[${sectionIndex}].policyGroups[${groupIndex}].values`,
                group.values.filter((_, idx) => idx !== valueIndex)
            );
        };
    }

    function handleAddValue() {
        setFieldValue(`policySections[${sectionIndex}].policyGroups[${groupIndex}].values`, [
            ...group.values,
            { value: '' },
        ]);
    }

    const headerText = group.negate
        ? descriptor.negatedName
        : descriptor?.longName || descriptor?.shortName || descriptor?.name;

    console.log('PolicyGroupCard', group);
    return (
        <>
            <Card isFlat isCompact>
                <CardHeader className="pf-u-p-0">
                    <CardTitle className="pf-u-pl-md">
                        <Flex alignItems={{ default: 'alignItemsCenter' }}>{headerText}:</Flex>
                    </CardTitle>
                    <CardActions hasNoOffset>
                        {descriptor?.negatedName && (
                            <>
                                <Divider component="div" isVertical />
                                <Checkbox
                                    label="Not"
                                    isChecked={group.negate}
                                    onChange={handleNegate}
                                    id={`${group.fieldName}-negate`}
                                />
                            </>
                        )}
                        <Divider isVertical component="div" />
                        <Button
                            variant="plain"
                            className="pf-u-mr-xs pf-u-px-sm pf-u-py-md"
                            onClick={onDeleteGroup}
                        >
                            <TrashIcon />
                        </Button>
                    </CardActions>
                </CardHeader>
                <Divider component="div" />
                <CardBody>
                    {group.values.map((_, valueIndex) => {
                        const name = `policySections[${sectionIndex}].policyGroups[${groupIndex}].values[${valueIndex}]`;
                        const groupName = `policySections[${sectionIndex}].policyGroups[${groupIndex}]`;
                        return (
                            // eslint-disable-next-line react/no-array-index-key
                            <Flex
                                direction={{ default: 'column' }}
                                spaceItems={{ default: 'spaceItemsNone' }}
                            >
                                <FieldValue
                                    name={name}
                                    length={group.values.length}
                                    descriptor={descriptor}
                                    handleRemoveValue={handleRemoveValue(valueIndex)}
                                />
                                {/* only show and/or operator if not at end of array */}
                                {valueIndex !== group.values.length - 1 && (
                                    <FlexItem alignSelf={{ default: 'alignSelfCenter' }}>
                                        <AndOrOperatorField
                                            name={groupName}
                                            readOnly={readOnly || !descriptor.canBooleanLogic}
                                        />
                                    </FlexItem>
                                )}
                            </Flex>
                        );
                    })}
                    {/* this is because there can't be multiple boolean values */}
                    {!readOnly && descriptor?.type !== 'radioGroup' && (
                        <Flex
                            direction={{ default: 'column' }}
                            alignItems={{ default: 'alignItemsCenter' }}
                            className="pf-u-pt-sm"
                        >
                            <Button
                                onClick={handleAddValue}
                                variant="plain"
                                // dataTestId="add-policy-field-value-btn"
                            >
                                <PlusIcon />
                            </Button>
                        </Flex>
                    )}
                </CardBody>
            </Card>
            <Flex
                direction={{ default: 'row' }}
                className="pf-u-my-sm"
                justifyContent={{ default: 'justifyContentCenter' }}
            >
                — and —
            </Flex>
        </>
    );
}

export default PolicyGroupCard;
