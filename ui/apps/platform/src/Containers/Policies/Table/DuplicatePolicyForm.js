import React, { useCallback } from 'react';
import PropTypes from 'prop-types';
import { useFormikContext, Field } from 'formik';

import RenamePolicySection from './RenamePolicySection';
import KeepBothSection from './KeepBothSection';
import { POLICY_DUPE_ACTIONS } from './PolicyImport.utils';

const DuplicatePolicyForm = ({ updateResolution, showKeepBothPolicies }) => {
    const { values } = useFormikContext();

    // this creates a partially applied function to update the radio button value,
    //   and then notified the parent
    const changeRadio = useCallback(
        (handler, name, value) => () => {
            handler(name)(value);
            updateResolution(name, value);
        },
        [updateResolution]
    );

    // this creates a partially applied function to update a text value,
    //   and then notified the parent
    const changeText = useCallback(
        (handler, name) => (evt) => {
            handler(evt);
            updateResolution(name, evt.target.value);
        },
        [updateResolution]
    );

    const highlightColor =
        values.resolution === POLICY_DUPE_ACTIONS.OVERWRITE ? 'bg-tertiary-200' : '';

    return (
        <form className="flex flex-col" data-testid="dupe-policy-form">
            {!showKeepBothPolicies && (
                <RenamePolicySection changeRadio={changeRadio} changeText={changeText} />
            )}
            {showKeepBothPolicies && <KeepBothSection changeRadio={changeRadio} />}
            <label
                htmlFor="overwrite-radio"
                className={`flex items-center py-2 px-2 py-4 rounded text-base-600 font-700 ${highlightColor}`}
            >
                <Field name="resolution">
                    {({ field }) => (
                        <input
                            name={field.name}
                            id="overwrite-radio"
                            type="radio"
                            className="form-radio border-base-600 text-base-600"
                            value="overwrite"
                            checked={field.value === POLICY_DUPE_ACTIONS.OVERWRITE}
                            onChange={changeRadio(
                                field.onChange,
                                field.name,
                                POLICY_DUPE_ACTIONS.OVERWRITE
                            )}
                        />
                    )}
                </Field>
                <span className="ml-1">Overwrite existing policy</span>
            </label>
        </form>
    );
};

DuplicatePolicyForm.propTypes = {
    updateResolution: PropTypes.func.isRequired,
    showKeepBothPolicies: PropTypes.bool.isRequired,
};

export default DuplicatePolicyForm;
