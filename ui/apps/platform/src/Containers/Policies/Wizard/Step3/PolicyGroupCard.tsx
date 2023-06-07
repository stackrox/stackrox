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
    Stack,
    StackItem,
} from '@patternfly/react-core';
import { TrashIcon, PlusIcon } from '@patternfly/react-icons';
import { useFormikContext } from 'formik';

import { Policy } from 'types/policy.proto';
import { Descriptor } from './policyCriteriaDescriptors';
import PolicyCriteriaFieldValue from './PolicyCriteriaFieldValue';
import AndOrOperatorField from './AndOrOperatorField';
import './PolicyGroupCard.css';

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

    const headerLongText = group.negate ? descriptor.negatedName : descriptor.longName;

    return (
        <>
            <Card isFlat isCompact data-testid="policy-criteria-group-card">
                <CardHeader className="pf-u-p-0">
                    <CardTitle className="pf-u-pl-md">
                        <Flex
                            alignItems={{ default: 'alignItemsCenter' }}
                            className="pf-u-py-sm pf-u-text-wrap-on-sm"
                        >
                            <Stack>
                                <StackItem>{descriptor.shortName}</StackItem>
                                {headerLongText && headerLongText !== descriptor.shortName && (
                                    <StackItem className="pf-u-font-weight-normal">
                                        {headerLongText}:
                                    </StackItem>
                                )}
                            </Stack>
                        </Flex>
                    </CardTitle>
                    <CardActions hasNoOffset className="policy-group-card">
                        {descriptor.negatedName && !readOnly && (
                            <>
                                <Divider component="div" isVertical />
                                <Checkbox
                                    label="Not"
                                    isChecked={group.negate}
                                    onChange={handleNegate}
                                    id={`${group.fieldName}-negate`}
                                    isDisabled={readOnly}
                                    data-testid="policy-criteria-value-negate-checkbox"
                                />
                            </>
                        )}
                        {!readOnly && (
                            <>
                                <Divider isVertical component="div" />
                                <Button
                                    variant="plain"
                                    className="pf-u-mr-xs pf-u-px-sm pf-u-py-md"
                                    onClick={onDeleteGroup}
                                    data-testid="delete-policy-criteria-btn"
                                >
                                    <TrashIcon />
                                </Button>
                            </>
                        )}
                    </CardActions>
                </CardHeader>
                <Divider component="div" />
                <CardBody>
                    {group.values.map((_, valueIndex) => {
                        const name = `policySections[${sectionIndex}].policyGroups[${groupIndex}].values[${valueIndex}]`;
                        const groupName = `policySections[${sectionIndex}].policyGroups[${groupIndex}]`;
                        return (
                            <React.Fragment key={name}>
                                <Flex
                                    direction={{ default: 'column' }}
                                    spaceItems={{ default: 'spaceItemsNone' }}
                                >
                                    <PolicyCriteriaFieldValue
                                        name={name}
                                        length={group.values.length}
                                        descriptor={descriptor}
                                        handleRemoveValue={handleRemoveValue(valueIndex)}
                                        readOnly={readOnly}
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
                            </React.Fragment>
                        );
                    })}
                    {/* this is because there can't be multiple boolean values */}
                    {!readOnly &&
                        descriptor.type !== 'tableModal' &&
                        descriptor.type !== 'radioGroup' &&
                        descriptor.type !== 'radioGroupString' && (
                            <Flex
                                direction={{ default: 'column' }}
                                alignItems={{ default: 'alignItemsCenter' }}
                                className="pf-u-pt-sm"
                            >
                                <Button
                                    onClick={handleAddValue}
                                    variant="plain"
                                    data-testid="add-policy-criteria-value-btn"
                                >
                                    <PlusIcon />
                                </Button>
                            </Flex>
                        )}
                </CardBody>
            </Card>
            {(policyGroups.length - 1 !== groupIndex || !readOnly) && (
                <Flex
                    direction={{ default: 'row' }}
                    className="pf-u-my-sm"
                    justifyContent={{ default: 'justifyContentCenter' }}
                >
                    — and —
                </Flex>
            )}
        </>
    );
}

export default PolicyGroupCard;
