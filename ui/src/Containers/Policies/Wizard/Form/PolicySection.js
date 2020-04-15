import React from 'react';
import PropTypes from 'prop-types';
import { useDrop } from 'react-dnd';
import { Trash2 } from 'react-feather';
import { FieldArray } from 'redux-form';

import reduxFormPropTypes from 'constants/reduxFormPropTypes';
import DRAG_DROP_TYPES from 'constants/dragDropTypes';
import Button from 'Components/Button';
import SectionHeaderInput from 'Components/SectionHeaderInput';
import AndOrOperator from 'Components/AndOrOperator';
import PolicyFieldCard from './PolicyFieldCard';
import { policyConfiguration } from './descriptors';

const getEmptyPolicyFieldCard = fieldKey => ({
    field_name: fieldKey.name,
    boolean_operator: 'OR',
    values: [
        {
            value: 'hi'
        }
    ],
    negate: fieldKey.negate,
    fieldKey
});

function PolicySection({ fields, header, removeSectionHandler }) {
    const [, drop] = useDrop({
        accept: DRAG_DROP_TYPES.KEY,
        drop: ({ fieldKey }) => {
            const newPolicyFieldCard = getEmptyPolicyFieldCard(fieldKey);
            fields.push(newPolicyFieldCard);
        }
    });

    function removeFieldHandler(index) {
        return () => fields.remove(index);
    }

    return (
        <>
            <div className="bg-base-300 border-2 border-base-100 rounded">
                <div className="flex justify-between items-center border-b-2 border-base-400">
                    <SectionHeaderInput header={header} />
                    <Button
                        onClick={removeSectionHandler}
                        icon={<Trash2 className="w-5 h-5" />}
                        className="p-2 border-l-2 border-base-400 hover:bg-base-400"
                    />
                </div>
                <div className="p-2">
                    {fields.map((name, i) => {
                        const field = fields.get(i);
                        const {
                            negate,
                            field_name: fieldName,
                            boolean_operator: booleanOperator
                        } = field;
                        let { fieldKey } = field;
                        if (!fieldKey)
                            fieldKey = policyConfiguration.descriptor.find(
                                fieldObj => fieldObj.name === fieldName
                            );
                        return (
                            <FieldArray
                                key={name}
                                name={`${name}.values`}
                                component={PolicyFieldCard}
                                isNegated={negate}
                                header={fieldName}
                                booleanOperator={booleanOperator}
                                removeFieldHandler={removeFieldHandler(i)}
                                fieldKey={fieldKey}
                                toggleFieldName={`${name}.negate`}
                            />
                        );
                    })}
                    <div
                        ref={drop}
                        className="bg-base-200 rounded border-2 border-base-300 border-dashed flex font-700 justify-center p-3 text-base-500 text-sm uppercase"
                    >
                        Drop a policy field inside
                    </div>
                </div>
            </div>
            <AndOrOperator />
        </>
    );
}

PolicySection.propTypes = {
    ...reduxFormPropTypes,
    header: PropTypes.string.isRequired,
    removeSectionHandler: PropTypes.func.isRequired
};

export default PolicySection;
