import React from 'react';
import PropTypes from 'prop-types';
import { PlusCircle } from 'react-feather';
import { FieldArray } from 'redux-form';

import reduxFormPropTypes from 'constants/reduxFormPropTypes';
import { addFieldArrayHandler, removeFieldArrayHandler } from './utils';
import PolicySection from './PolicySection';

const MAX_POLICY_SECTIONS = 16;

const emptyPolicySection = {
    sectionName: '',
    policyGroups: [],
};

function getNewPolicySection(length) {
    return {
        ...emptyPolicySection,
        sectionName: `Policy Section ${length}`,
    };
}

function PolicySections({ fields, readOnly, className, hasHeader }) {
    const newPolicySection = getNewPolicySection(fields.length);
    return (
        <div className={`p-3 ${className}`}>
            {hasHeader && <h2 className="text-2xl pb-2">Policy Criteria</h2>}
            {!readOnly && hasHeader && (
                <div className="text-sm italic pb-5 text-base-500">
                    Construct policy rules by chaining criteria together with boolean logic.
                </div>
            )}
            {fields.map((name, i) => {
                // we get name and index when iterating through fields in a FieldArray in redux-form
                // https://redux-form.com/8.2.2/docs/api/fieldarray.md/#iteration
                return (
                    <FieldArray
                        key={name}
                        name={`${name}.policyGroups`}
                        component={PolicySection}
                        sectionName={`${name}.sectionName`}
                        removeSectionHandler={removeFieldArrayHandler(fields, i)}
                        readOnly={readOnly}
                        isLast={i === fields.length - 1}
                    />
                );
            })}
            {!readOnly && fields.length < MAX_POLICY_SECTIONS && (
                <button
                    type="button"
                    onClick={addFieldArrayHandler(fields, newPolicySection)}
                    className="p-2 w-full border-2 border-base-100 bg-base-300 flex justify-center items-center"
                    data-testid="add-policy-section-btn"
                >
                    <PlusCircle className="w-4 h-4 text-base-600" />
                    <div className="pl-2 py-1 text-sm text-base-600 font-700">
                        Add a new condition
                    </div>
                </button>
            )}
        </div>
    );
}

PolicySections.propTypes = {
    ...reduxFormPropTypes,
    className: PropTypes.string,
    readOnly: PropTypes.bool,
    hasHeader: PropTypes.bool,
};
PolicySections.defaultProps = { className: 'w-2/3', readOnly: false, hasHeader: true };

export default PolicySections;
