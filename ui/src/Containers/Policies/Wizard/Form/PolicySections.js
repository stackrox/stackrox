import React from 'react';
import { PlusCircle } from 'react-feather';
import { FieldArray } from 'redux-form';

import reduxFormPropTypes from 'constants/reduxFormPropTypes';
import PolicySection from './PolicySection';

const emptyPolicySection = {
    section_name: '',
    policy_groups: []
};

function PolicySections({ fields }) {
    function addPolicySectionHandler() {
        const newPolicySection = {
            ...emptyPolicySection,
            section_name: `policy section ${fields.length}`
        };
        fields.push(newPolicySection);
    }

    function removeSectionHandler(index) {
        return () => fields.remove(index);
    }

    return (
        <div className="w-2/3 p-3">
            <h2 className="text-2xl pb-2">Policy Criteria</h2>
            <div className="text-sm italic pb-5 text-base-500">
                Construct policy rules by chaining criteria together with boolean logic.
            </div>
            {fields.map((name, i) => {
                // we get name and index when iterating through fields in a FieldArray in redux-form
                // https://redux-form.com/8.2.2/docs/api/fieldarray.md/#iteration
                const { section_name: sectionName } = fields.get(i);

                return (
                    <FieldArray
                        key={name}
                        name={`${name}.policy_groups`}
                        component={PolicySection}
                        header={sectionName}
                        removeSectionHandler={removeSectionHandler(i)}
                    />
                );
            })}
            <button
                type="button"
                onClick={addPolicySectionHandler}
                className="p-2 w-full border-2 border-base-100 bg-base-300 flex justify-center items-center"
            >
                <PlusCircle className="w-4 h-4 text-base-600" />
                <div className="pl-2 py-1 text-sm text-base-600 font-700">Add a new condition</div>
            </button>
        </div>
    );
}

PolicySections.propTypes = {
    fields: reduxFormPropTypes.isRequired
};

export default PolicySections;
