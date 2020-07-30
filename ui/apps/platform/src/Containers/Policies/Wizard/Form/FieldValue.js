import React from 'react';
import PropTypes from 'prop-types';

import ReduxAndOrOperatorField from 'Components/forms/ReduxAndOrOperatorField';
import FormFieldRemoveButton from 'Components/FormFieldRemoveButton';
import Field from './Field';

function FieldValue({
    name,
    length,
    booleanOperatorName,
    fieldKey,
    removeValueHandler,
    isLast,
    readOnly,
}) {
    return (
        <>
            <div className="flex" data-testid="policy-field-value">
                <Field key={name} field={fieldKey} name={name} readOnly={readOnly} BPLenabled />
                {/* only show remove button if there is more than one value */}
                {!readOnly && length > 1 && (
                    <FormFieldRemoveButton
                        field={name}
                        onClick={removeValueHandler}
                        dataTestId="remove-policy-field-value-btn"
                        className="border-base-300 hover:border-base-400 hover:text-base-600 rounded-r text-base-100 text-base-500"
                    />
                )}
            </div>
            {/* only show and/or operator if not at end of array */}
            {!isLast && (
                <ReduxAndOrOperatorField
                    name={booleanOperatorName}
                    disabled={readOnly || !fieldKey.canBooleanLogic}
                    isCircular
                />
            )}
        </>
    );
}

FieldValue.propTypes = {
    name: PropTypes.string.isRequired,
    length: PropTypes.number.isRequired,
    fieldKey: PropTypes.shape({
        canBooleanLogic: PropTypes.bool,
    }).isRequired,
    booleanOperatorName: PropTypes.string.isRequired,
    removeValueHandler: PropTypes.func.isRequired,
    isLast: PropTypes.bool.isRequired,
    readOnly: PropTypes.bool,
};

FieldValue.defaultProps = {
    readOnly: false,
};

export default FieldValue;
