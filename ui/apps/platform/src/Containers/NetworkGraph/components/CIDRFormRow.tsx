import React from 'react';
import { Field, ErrorMessage } from 'formik';

import FormFieldRemoveButton from 'Components/FormFieldRemoveButton';

const CIDRFormRowErrorMessage = ({ errors, touched, idx }) => {
    const { name: nameError, cidr: cidrError } = errors?.entities?.[idx]?.entity || {};
    const { name: nameTouched, cidr: cidrTouched } = touched?.entities?.[idx]?.entity || {};
    const showNameError = nameError && nameTouched;
    const showCidrError = cidrError && cidrTouched;
    if (!showNameError && !showCidrError) {
        return null;
    }
    return (
        <div className="bg-alert-300 p-1 text-alert-800 text-xs rounded-b border-alert-400 border-b border-l border-r flex flex-col">
            {showNameError && (
                <div>
                    <ErrorMessage name={`entities.${idx as string}.entity.name`} />
                </div>
            )}
            {showCidrError && (
                <div>
                    <ErrorMessage name={`entities.${idx as string}.entity.cidr`} />
                </div>
            )}
        </div>
    );
};

const CIDRFormRow = ({ idx, onRemoveRow, errors, touched }) => {
    const isFirstRow = idx === 0;
    return (
        <div className="flex flex-col mb-2">
            <div className="flex">
                <div className="flex flex-1 flex-col">
                    {isFirstRow && (
                        <label
                            htmlFor="cidr-block-name"
                            className="pb-1"
                            aria-labelledby="cidr-block-name-label"
                        >
                            CIDR Block Name
                            <span data-testid="required" className="text-alert-500 ml-1">
                                *
                            </span>
                        </label>
                    )}
                    <Field
                        name={`entities.${idx as string}.entity.name`}
                        type="text"
                        id="cidr-block-name"
                        placeholder="CIDR block"
                        className="border border-base-300 rounded-l h-10 px-2 w-full"
                    />
                </div>
                <div className="flex flex-1 flex-col">
                    {isFirstRow && (
                        <label
                            htmlFor="cidr-block-address"
                            className="pb-1"
                            aria-labelledby="cidr-block-address-label"
                        >
                            CIDR Address
                            <span data-testid="required" className="text-alert-500 ml-1">
                                *
                            </span>
                        </label>
                    )}
                    <Field
                        name={`entities.${idx as string}.entity.cidr`}
                        type="text"
                        className="border border-base-300 h-10 px-2 w-full"
                        id="cidr-block-address"
                        placeholder="192.0.0.2/24"
                    />
                </div>
                <FormFieldRemoveButton
                    onClick={onRemoveRow}
                    className="p-1 rounded-l-none text-base-100 text-alert-700 hover:text-alert-800 bg-alert-200 hover:bg-alert-300 border-alert-300 rounded"
                />
            </div>
            <CIDRFormRowErrorMessage errors={errors} touched={touched} idx={idx} />
        </div>
    );
};

export default CIDRFormRow;
