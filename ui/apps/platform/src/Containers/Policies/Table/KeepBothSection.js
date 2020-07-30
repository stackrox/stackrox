import React from 'react';
import PropTypes from 'prop-types';
import { useFormikContext, Field } from 'formik';

import { POLICY_DUPE_ACTIONS } from './PolicyImport.utils';

const KeepBothSection = ({ changeRadio }) => {
    const { values } = useFormikContext();

    const highlightColor =
        values.resolution === POLICY_DUPE_ACTIONS.KEEP_BOTH ? 'bg-tertiary-200' : '';

    return (
        <label
            htmlFor="keep-both"
            className={`flex items-center py-2 px-2 py-4 rounded text-base-600 font-700 ${highlightColor}`}
        >
            <Field name="resolution">
                {({ field }) => (
                    <input
                        name={field.name}
                        id="keep-both"
                        type="radio"
                        className="form-radio border-base-600 text-base-600"
                        value="keepBoth"
                        checked={field.value === POLICY_DUPE_ACTIONS.KEEP_BOTH}
                        onChange={changeRadio(
                            field.onChange,
                            field.name,
                            POLICY_DUPE_ACTIONS.KEEP_BOTH
                        )}
                    />
                )}
            </Field>
            <span className="ml-1">
                Keep both policies (imported policy will be assigned a new ID)
            </span>
        </label>
    );
};

KeepBothSection.propTypes = {
    changeRadio: PropTypes.func.isRequired,
};

export default KeepBothSection;
