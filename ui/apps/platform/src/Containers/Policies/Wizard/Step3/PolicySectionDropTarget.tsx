import React from 'react';
import keyBy from 'lodash/keyBy';
import { Flex } from '@patternfly/react-core';
import { useDrop } from 'react-dnd';
import { useFormikContext } from 'formik';

import { Policy } from 'types/policy.proto';
import { Descriptor } from './policyCriteriaDescriptors';
import { getEmptyPolicyFieldCard } from '../../policies.utils';

import './PolicySectionDropTarget.css';

function getPolicyCriteriaFieldKeys(policyGroups, descriptors) {
    const fieldNameMap = keyBy(policyGroups, (field) => field.fieldName as string);
    const availableFieldKeys: string[] = [];
    descriptors.forEach((field) => {
        if (!fieldNameMap[field.name]) {
            availableFieldKeys.push(field.name);
        }
    });
    return availableFieldKeys;
}

interface DragItem {
    index: number;
    id: string;
    fieldKey: Descriptor;
}

function PolicySectionDropTarget({ sectionIndex, descriptors }) {
    const { values, setFieldValue } = useFormikContext<Policy>();
    const { policyGroups } = values.policySections[sectionIndex];
    const acceptedFields = getPolicyCriteriaFieldKeys(policyGroups, descriptors);

    function addPolicyFieldCardHandler(fieldCard) {
        setFieldValue(`policySections[${sectionIndex as string}].policyGroups`, [
            ...policyGroups,
            fieldCard,
        ]);
    }

    const [{ isOver, canDrop, getItemType }, drop] = useDrop({
        accept: acceptedFields,
        drop: (item: DragItem) => {
            const newPolicyFieldCard = getEmptyPolicyFieldCard(item.fieldKey);
            addPolicyFieldCardHandler(newPolicyFieldCard);
        },
        collect: (monitor) => ({
            isOver: monitor.isOver(),
            canDrop: monitor.canDrop(),
            getItemType: monitor.getItemType(),
        }),
    });

    let dropStyle = 'pf-u-background-color-200';
    // getItemType returns the item type if an item is currently being dragged
    if (!canDrop && !!getItemType) {
        dropStyle = 'pf-u-background-color-disabled-color-200';
    } else if (canDrop && isOver) {
        dropStyle = 'pf-u-background-color-success';
    } else if (canDrop) {
        dropStyle = 'pf-u-background-color-default';
    }

    return (
        <div ref={drop}>
            <Flex
                data-testid="policy-section-drop-target"
                justifyContent={{ default: 'justifyContentCenter' }}
                className={`pf-u-p-sm dropzone ${dropStyle}`}
            >
                Drop a policy field inside
            </Flex>
        </div>
    );
}

export default PolicySectionDropTarget;
