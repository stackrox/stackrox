import React from 'react';
import PropTypes from 'prop-types';
import { useFormikContext, Field } from 'formik';

import { POLICY_DUPE_ACTIONS } from './PolicyImport.utils';

const RenamePolicySection = ({ changeRadio, changeText }) => {
    const { values } = useFormikContext();

    const highlightColor =
        values.resolution === POLICY_DUPE_ACTIONS.RENAME ? 'bg-tertiary-200' : '';

    return (
        <fieldset className={`flex items-center mb-4 p-2 rounded ${highlightColor}`}>
            <label htmlFor="rename-radio" className="flex items-center py-2 text-base-600 font-700">
                <Field name="resolution">
                    {({ field }) => (
                        <input
                            name={field.name}
                            id="rename-radio"
                            type="radio"
                            className="form-radio border-base-600 text-base-600"
                            value="rename"
                            checked={field.value === POLICY_DUPE_ACTIONS.RENAME}
                            onChange={changeRadio(
                                field.onChange,
                                field.name,
                                POLICY_DUPE_ACTIONS.RENAME
                            )}
                        />
                    )}
                </Field>
                <span className="ml-1">Rename incoming policy</span>
            </label>
            <label
                htmlFor="new-policy-name"
                className="flex flex-col ml-4 block py-2 text-base-600 font-700"
            >
                <span className="mb-2">New name:</span>
                <Field name="newName">
                    {({ field, form, meta }) => {
                        const isDisabled = form.values.resolution !== POLICY_DUPE_ACTIONS.RENAME;
                        return (
                            <div>
                                <input
                                    className={`bg-base-100 ${
                                        isDisabled ? 'bg-base-200' : 'hover:border-base-400'
                                    } border-2 rounded p-2 border-base-300 w-full font-600 text-base-600 leading-normal min-h-10`}
                                    name={field.name}
                                    id="new-policy-name"
                                    type="text"
                                    value={field.value}
                                    disabled={isDisabled}
                                    onChange={changeText(field.onChange, field.name)}
                                    onBlur={field.onBlur}
                                />
                                {meta.touched && meta.error && (
                                    <div
                                        className="text-alert-700 mt-1"
                                        data-testid="new-name-error"
                                    >
                                        {meta.error}
                                    </div>
                                )}
                            </div>
                        );
                    }}
                </Field>
            </label>
        </fieldset>
    );
};

RenamePolicySection.propTypes = {
    changeRadio: PropTypes.func.isRequired,
    changeText: PropTypes.func.isRequired,
};

export default RenamePolicySection;
