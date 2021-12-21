import React from 'react';
import {
    Card,
    CardHeader,
    CardTitle,
    CardActions,
    CardBody,
    Divider,
    Flex,
    Button,
    Checkbox,
} from '@patternfly/react-core';
import { TrashIcon } from '@patternfly/react-icons';
import { useFormikContext } from 'formik';

import { Descriptor } from 'Containers/Policies/Wizard/Form/descriptors';
import { Policy } from 'types/policy.proto';

type PolicyGroupCardProps = {
    field: Descriptor;
    groupIndex: number;
    sectionIndex: number;
};

function PolicyGroupCard({ field, groupIndex, sectionIndex }: PolicyGroupCardProps) {
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

    // function handleBooleanOperator() {
    //     const newBooleanValue = group.booleanOperator === 'AND' ? 'OR' : 'AND';
    //     setFieldValue(
    //         `policySections[${sectionIndex as string}].policyGroups[${groupIndex as string}]`,
    //         { ...group, booleanOperator: newBooleanValue }
    //     );
    // }

    const headerText = group.negate
        ? field.negatedName
        : field?.longName || field?.shortName || field?.name;

    return (
        <>
            <Card isFlat isCompact>
                <CardHeader className="pf-u-p-0">
                    <CardTitle className="pf-u-pl-md">
                        <Flex alignItems={{ default: 'alignItemsCenter' }}>{headerText}:</Flex>
                    </CardTitle>
                    <CardActions hasNoOffset>
                        {field?.negatedName && (
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
                <CardBody>sigh</CardBody>
            </Card>
            <Flex
                direction={{ default: 'row' }}
                className="pf-u-my-sm"
                justifyContent={{ default: 'justifyContentCenter' }}
            >
                {/* <Button
                        variant="plain"
                        onClick={handleBooleanOperator}
                        isDisabled={field?.canBooleanLogic}
                    > */}
                — and —{/* </Button> */}
            </Flex>
        </>
    );
}

export default PolicyGroupCard;
