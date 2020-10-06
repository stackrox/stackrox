import React from 'react';
import { Field } from 'formik';

import FormFieldRemoveButton from 'Components/FormFieldRemoveButton';

const CIDRFormRow = ({ idx, onRemoveRow }) => {
    const isFirstRow = idx === 0;
    return (
        <div className="flex mb-2">
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
                    name={`entities.${idx}.entity.name`}
                    type="text"
                    id="cidr-block-name"
                    className="border border-base-300 rounded-l h-10 px-2 w-full"
                    placeholder="CIDR block"
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
                    name={`entities.${idx}.entity.cidr`}
                    type="text"
                    id="cidr-block-address"
                    className="border border-base-300 h-10 px-2 w-full"
                    placeholder="192.0.0.2/24"
                />
            </div>
            <FormFieldRemoveButton
                onClick={onRemoveRow}
                className="p-1 rounded-l-none text-base-100 text-alert-700 hover:text-alert-800 bg-alert-200 hover:bg-alert-300 border-alert-300 rounded"
            />
        </div>
    );
};

export default CIDRFormRow;
