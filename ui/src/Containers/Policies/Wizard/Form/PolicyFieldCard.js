import React from 'react';
import PropTypes from 'prop-types';
import { Trash2, PlusCircle } from 'react-feather';

import reduxFormPropTypes from 'constants/reduxFormPropTypes';
import Button from 'Components/Button';
import ToggleSwitch from 'Components/ToggleSwitch';
import AndOrOperator from 'Components/AndOrOperator';
// import FieldValue from './FieldValue';

function PolicyFieldCard({
    toggleHandler,
    isNegated,
    removeFieldHandler,
    fields,
    header,
    addValueHander,
    booleanOperator
}) {
    const borderColorClass = isNegated ? 'border-accent-400' : 'border-base-400';
    return (
        <>
            <div className={`bg-base-200 border-2 ${borderColorClass} rounded`}>
                <div className={`border-b-2 ${borderColorClass} flex`}>
                    <div className="flex flex-1 font-700 p-2 pl-3 text-base-600 text-sm uppercase items-center">
                        {header}:
                    </div>
                    {toggleHandler && (
                        <div className={`flex items-center p-2 border-l-2 ${borderColorClass}`}>
                            <ToggleSwitch
                                small
                                id="policy-field-card-negation"
                                toggleHandler={toggleHandler}
                                enabled={isNegated}
                                label="NOT"
                                labelClassName="text-sm text-base-600 font-700"
                            />
                        </div>
                    )}
                    <Button
                        onClick={removeFieldHandler}
                        icon={<Trash2 className="w-5 h-5" />}
                        className={`p-2 border-l-2 ${borderColorClass}`}
                    />
                </div>
                {fields.map((name, index) => {
                    const fieldValue = fields.get(index);
                    return <div key={name}>{fieldValue.value}</div>;
                })}
                <div className="flex flex-col p-2">
                    {addValueHander && (
                        <div className="flex justify-center">
                            <Button
                                onClick={addValueHander}
                                icon={<PlusCircle className="w-5 h-5" />}
                            />
                        </div>
                    )}
                </div>
            </div>
            <AndOrOperator value={booleanOperator} />
        </>
    );
}

PolicyFieldCard.propTypes = {
    toggleHandler: PropTypes.func,
    isNegated: PropTypes.bool,
    removeFieldHandler: PropTypes.func.isRequired,
    header: PropTypes.string.isRequired,
    addValueHander: PropTypes.func,
    booleanOperator: PropTypes.string.isRequired,
    ...reduxFormPropTypes
};

PolicyFieldCard.defaultProps = {
    toggleHandler: null,
    isNegated: false,
    addValueHander: null
};

export default PolicyFieldCard;
