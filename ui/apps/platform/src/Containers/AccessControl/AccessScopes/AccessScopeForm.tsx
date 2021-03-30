import React, { ReactElement } from 'react';
import { useFormik } from 'formik';
import * as yup from 'yup';

import FormFieldRequired from 'Components/forms/FormFieldRequired';

import { inputTextClassName, labelClassName } from '../AccessControlComponents';
import { AccessScope } from '../accessControlTypes';

export type AccessScopeFormProps = {
    accessScope: AccessScope;
    isEditing: boolean;
};

function AccessScopeForm({ accessScope, isEditing }: AccessScopeFormProps): ReactElement {
    const { handleChange, values } = useFormik({
        initialValues: accessScope,
        onSubmit: () => {},
        validationSchema: yup.object({
            name: yup.string().required(),
            description: yup.string(),
        }),
    });

    const disabled = !isEditing;

    return (
        <form className="pt-4 px-4 text-base-600">
            <div className="pb-4">
                <label htmlFor="name" className={labelClassName}>
                    Name <FormFieldRequired empty={values.name.length === 0} />
                </label>
                <input
                    id="name"
                    name="name"
                    value={values.name}
                    onChange={handleChange}
                    disabled={disabled}
                    className={inputTextClassName}
                />
            </div>
            <div className="pb-4">
                <label htmlFor="description" className={labelClassName}>
                    Description
                </label>
                <input
                    id="description"
                    name="description"
                    value={values.description}
                    onChange={handleChange}
                    disabled={disabled}
                    className={inputTextClassName}
                />
            </div>
        </form>
    );
}

export default AccessScopeForm;
