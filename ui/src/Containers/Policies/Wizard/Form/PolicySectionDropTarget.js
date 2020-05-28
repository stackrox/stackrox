import React from 'react';
import PropTypes from 'prop-types';
import { useDrop } from 'react-dnd';

import { getPolicyCriteriaFieldKeys } from './utils';

const getEmptyPolicyFieldCard = (fieldKey) => {
    const defaultValue = fieldKey.defaultValue !== undefined ? fieldKey.defaultValue : '';
    return {
        fieldName: fieldKey.name,
        booleanOperator: 'OR',
        values: [
            {
                value: defaultValue,
            },
        ],
        negate: false,
        fieldKey,
    };
};

function PolicySectionDropTarget({ allFields, addPolicyFieldCardHandler }) {
    const acceptedFields = getPolicyCriteriaFieldKeys(allFields);

    const [{ isOver, canDrop }, drop] = useDrop({
        accept: acceptedFields,
        drop: ({ fieldKey }) => {
            const newPolicyFieldCard = getEmptyPolicyFieldCard(fieldKey);
            addPolicyFieldCardHandler(newPolicyFieldCard);
        },
        canDrop: ({ fieldKey }) => {
            return !allFields.find((field) => field.fieldName === fieldKey.name);
        },
        collect: (monitor) => ({
            isOver: monitor.isOver(),
            canDrop: monitor.canDrop(),
        }),
    });

    const disabledDrop = !canDrop && isOver;

    const disabledDropStyle = disabledDrop
        ? 'bg-base-300 border-base-400'
        : 'bg-base-200 border-base-300';
    const canDropStyle = !disabledDrop && canDrop ? 'border-accent-700' : '';
    const isOverStyle = !disabledDrop && isOver ? 'bg-accent-300' : '';

    return (
        <div
            ref={drop}
            data-testid="policy-section-drop-target"
            className={`${disabledDropStyle} ${canDropStyle} ${isOverStyle} rounded border-2 border-dashed flex font-700 justify-center p-3 text-base-500 text-sm uppercase`}
        >
            Drop a policy field inside
        </div>
    );
}

PolicySectionDropTarget.propTypes = {
    allFields: PropTypes.arrayOf(PropTypes.shape({})).isRequired,
    addPolicyFieldCardHandler: PropTypes.func.isRequired,
};

export default PolicySectionDropTarget;
