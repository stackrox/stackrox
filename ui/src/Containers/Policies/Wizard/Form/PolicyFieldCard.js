import React from 'react';
import PropTypes from 'prop-types';
import { Trash2, PlusCircle } from 'react-feather';

import reduxFormPropTypes from 'constants/reduxFormPropTypes';
import Button from 'Components/Button';
// import ToggleSwitch from 'Components/ToggleSwitch';
import ReduxToggleField from 'Components/forms/ReduxToggleField';
import AndOrOperator from 'Components/AndOrOperator';
import FieldValue from './FieldValue';

const emptyFieldValue = {
    value: ''
};

function PolicyFieldCard({
    isNegated,
    removeFieldHandler,
    fields,
    header,
    booleanOperator,
    fieldKey,
    toggleFieldName
}) {
    function addValueHander() {
        fields.push(emptyFieldValue);
    }

    function removeValueHandler(index) {
        return () => fields.remove(index);
    }

    const borderColorClass = isNegated ? 'border-accent-400' : 'border-base-400';
    return (
        <>
            <div className={`bg-base-200 border-2 ${borderColorClass} rounded`}>
                <div className={`border-b-2 ${borderColorClass} flex`}>
                    <div className="flex flex-1 font-700 p-2 pl-3 text-base-600 text-sm uppercase items-center">
                        {header}:
                    </div>
                    {fieldKey.canNegate && (
                        <div className={`flex items-center p-2 border-l-2 ${borderColorClass}`}>
                            <label
                                htmlFor={toggleFieldName}
                                className="text-sm text-base-600 font-700 mr-2"
                            >
                                NOT
                            </label>
                            <ReduxToggleField name={toggleFieldName} className="self-center" />
                        </div>
                    )}
                    <Button
                        onClick={removeFieldHandler}
                        icon={<Trash2 className="w-5 h-5" />}
                        className={`p-2 border-l-2 ${borderColorClass}`}
                    />
                </div>
                <div className="p-2">
                    {fields.map((name, i) => (
                        <FieldValue
                            key={name}
                            name={`${name}.value`}
                            length={fields.length}
                            booleanOperator={booleanOperator}
                            fieldKey={fieldKey}
                            removeValueHandler={removeValueHandler(i)}
                            index={i}
                        />
                    ))}
                    <div className="flex flex-col pt-2">
                        <div className="flex justify-center">
                            <Button
                                onClick={addValueHander}
                                icon={<PlusCircle className="w-5 h-5" />}
                            />
                        </div>
                    </div>
                </div>
            </div>
            <AndOrOperator value={booleanOperator} />
        </>
    );
}

PolicyFieldCard.propTypes = {
    isNegated: PropTypes.bool,
    removeFieldHandler: PropTypes.func.isRequired,
    header: PropTypes.string.isRequired,
    booleanOperator: PropTypes.string.isRequired,
    toggleFieldName: PropTypes.string.isRequired,
    ...reduxFormPropTypes
};

PolicyFieldCard.defaultProps = {
    isNegated: false
};

export default PolicyFieldCard;
