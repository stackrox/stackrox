import keyBy from 'lodash/keyBy';
import { Flex } from '@patternfly/react-core';
import { useDrop } from 'react-dnd';
import { useFormikContext } from 'formik';

import type { Policy } from 'types/policy.proto';
import type { Descriptor } from './policyCriteriaDescriptors';
import { getEmptyPolicyFieldCard } from '../../policies.utils';

import './PolicySectionDropTarget.css';
import type { CSSProperties } from 'react';

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

    const dropStyle: CSSProperties = {
        backgroundColor: 'var(--pf-t--global--background--color--secondary--default)',
    };
    // getItemType returns the item type if an item is currently being dragged
    if (!canDrop && !!getItemType) {
        dropStyle.backgroundColor = 'var(--pf-t--global--background--color--disabled--default)';
    } else if (canDrop && isOver) {
        dropStyle.borderColor = 'var(--pf-t--global--border--color--brand--default)';
    }

    return (
        <div ref={drop}>
            <Flex
                data-testid="policy-section-drop-target"
                justifyContent={{ default: 'justifyContentCenter' }}
                className={`pf-v6-u-p-sm dropzone`}
                style={dropStyle}
            >
                Drop a policy field inside
            </Flex>
        </div>
    );
}

export default PolicySectionDropTarget;
