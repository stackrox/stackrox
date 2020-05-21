import React from 'react';
import { connect } from 'react-redux';
import PropTypes from 'prop-types';
import { Trash2, PlusCircle } from 'react-feather';
import { createSelector, createStructuredSelector } from 'reselect';
import { formValueSelector } from 'redux-form';

import reduxFormPropTypes from 'constants/reduxFormPropTypes';
import BOOLEAN_LOGIC_VALUES from 'constants/booleanLogicValues';
import Button from 'Components/Button';
import ReduxToggleField from 'Components/forms/ReduxToggleField';
import AndOrOperator from 'Components/AndOrOperator';
import FieldValue from './FieldValue';
import { addFieldArrayHandler, removeFieldArrayHandler } from './utils';

const emptyFieldValue = {
    value: '',
};

function PolicyFieldCard({
    isNegated,
    removeFieldHandler,
    fields,
    booleanOperatorName,
    fieldKey,
    toggleFieldName,
    readOnly,
    isLast,
}) {
    const borderColorClass = isNegated ? 'border-accent-400' : 'border-base-400';
    const headerText = isNegated ? fieldKey.negatedName : fieldKey.longName || fieldKey.name;
    const lastFieldIndex = fields.length - 1;
    return (
        <>
            <div
                className={`bg-base-200 border-2 ${borderColorClass} rounded`}
                data-testid="policy-field-card"
            >
                <div className={`border-b-2 ${borderColorClass} flex`}>
                    <div className="flex flex-1 font-700 p-2 pl-3 text-base-600 text-sm uppercase items-center">
                        {headerText}:
                    </div>
                    {!readOnly && (
                        <>
                            {fieldKey.negatedName && (
                                <div
                                    className={`flex items-center p-2 border-l-2 ${borderColorClass}`}
                                >
                                    <label
                                        htmlFor={toggleFieldName}
                                        className="text-sm text-base-600 font-700 mr-2"
                                    >
                                        NOT
                                    </label>
                                    <ReduxToggleField
                                        name={toggleFieldName}
                                        className="self-center"
                                    />
                                </div>
                            )}
                            <Button
                                onClick={removeFieldHandler}
                                icon={<Trash2 className="w-5 h-5" />}
                                className={`p-2 border-l-2 ${borderColorClass}`}
                                dataTestId="remove-policy-field-card-btn"
                            />
                        </>
                    )}
                </div>
                <div className="p-2">
                    {fields.map((name, i) => (
                        <FieldValue
                            key={name}
                            name={name}
                            length={fields.length}
                            booleanOperatorName={booleanOperatorName}
                            fieldKey={fieldKey}
                            removeValueHandler={removeFieldArrayHandler(fields, i)}
                            isLast={i === lastFieldIndex}
                            readOnly={readOnly}
                        />
                    ))}
                    {/* this is because there can't be multiple boolean values */}
                    {!readOnly && fieldKey.type !== 'radioGroup' && (
                        <div className="flex flex-col pt-2">
                            <div className="flex justify-center">
                                <Button
                                    onClick={addFieldArrayHandler(fields, emptyFieldValue)}
                                    icon={<PlusCircle className="w-5 h-5" />}
                                    dataTestId="add-policy-field-value-btn"
                                />
                            </div>
                        </div>
                    )}
                </div>
            </div>
            {(!isLast || !readOnly) && <AndOrOperator value={BOOLEAN_LOGIC_VALUES.AND} disabled />}
        </>
    );
}

PolicyFieldCard.propTypes = {
    isNegated: PropTypes.bool.isRequired,
    removeFieldHandler: PropTypes.func.isRequired,
    booleanOperatorName: PropTypes.string.isRequired,
    toggleFieldName: PropTypes.string.isRequired,
    readOnly: PropTypes.bool,
    isLast: PropTypes.bool,
    ...reduxFormPropTypes,
};

PolicyFieldCard.defaultProps = {
    readOnly: false,
    isLast: false,
};

const isNegated = (state, ownProps) =>
    formValueSelector('policyCreationForm')(state, ownProps.toggleFieldName);

const getIsNegated = createSelector([isNegated], (negate) => negate);

const mapStateToProps = createStructuredSelector({
    isNegated: getIsNegated,
});

export default connect(mapStateToProps, null)(PolicyFieldCard);
