import React from 'react';
import PropTypes from 'prop-types';
import { Trash2 } from 'react-feather';
import { Field, FieldArray } from 'redux-form';

import reduxFormPropTypes from 'constants/reduxFormPropTypes';
import Button from 'Components/Button';
import SectionHeaderInput from 'Components/SectionHeaderInput';
import AndOrOperator from 'Components/AndOrOperator';
import PolicyFieldCard from './PolicyFieldCard';
import { policyConfiguration } from './descriptors';
import { removeFieldArrayHandler } from './utils';
import PolicySectionDropTarget from './PolicySectionDropTarget';

function addPolicyFieldCardHandler(fields) {
    return (newPolicyFieldCard) => {
        fields.push(newPolicyFieldCard);
    };
}

function PolicySection({ fields, sectionName, removeSectionHandler, readOnly, isLast }) {
    return (
        <>
            <div
                className="bg-base-300 border-2 border-base-100 rounded"
                data-testid="policy-section"
            >
                <div className="flex justify-between items-center border-b-2 border-base-400">
                    <Field name={sectionName} component={SectionHeaderInput} readOnly={readOnly} />
                    {!readOnly && (
                        <Button
                            onClick={removeSectionHandler}
                            icon={<Trash2 className="w-5 h-5" />}
                            className="p-2 border-l-2 border-base-400 hover:bg-base-400"
                            dataTestId="remove-policy-section-btn"
                        />
                    )}
                </div>
                <div className="p-2">
                    {fields.map((name, i) => {
                        const field = fields.get(i);
                        let { fieldKey } = field;
                        if (!fieldKey) {
                            fieldKey = policyConfiguration.descriptor.find(
                                (fieldObj) =>
                                    fieldObj.name === field.fieldName ||
                                    fieldObj.label === field.fieldName
                            );
                        }
                        return (
                            <FieldArray
                                key={name}
                                name={`${name}.values`}
                                component={PolicyFieldCard}
                                booleanOperatorName={`${name}.booleanOperator`}
                                removeFieldHandler={removeFieldArrayHandler(fields, i)}
                                fieldKey={fieldKey}
                                toggleFieldName={`${name}.negate`}
                                readOnly={readOnly}
                                isLast={i === fields.length - 1}
                            />
                        );
                    })}
                    {!readOnly && (
                        <PolicySectionDropTarget
                            allFields={fields.getAll()}
                            addPolicyFieldCardHandler={addPolicyFieldCardHandler(fields)}
                        />
                    )}
                </div>
            </div>
            {(!isLast || !readOnly) && <AndOrOperator disabled />}
        </>
    );
}

PolicySection.propTypes = {
    ...reduxFormPropTypes,
    sectionName: PropTypes.string.isRequired,
    removeSectionHandler: PropTypes.func.isRequired,
    readOnly: PropTypes.bool,
    isLast: PropTypes.bool,
};

PolicySection.defaultProps = {
    readOnly: false,
    isLast: false,
};

export default PolicySection;
