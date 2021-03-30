import React, { ReactElement } from 'react';
import { useFormik } from 'formik';
import * as yup from 'yup';

import FormFieldRequired from 'Components/forms/FormFieldRequired';

import { inputTextClassName, labelClassName } from '../AccessControlComponents';
import { AuthProvider } from '../accessControlTypes';

export type AuthProviderFormProps = {
    authProvider: AuthProvider;
    isEditing: boolean;
};

function AuthProviderForm({ authProvider, isEditing }: AuthProviderFormProps): ReactElement {
    const { handleChange, values } = useFormik({
        initialValues: authProvider,
        onSubmit: () => {},
        validationSchema: yup.object({
            name: yup.string().required(),
            // authProvider
            // minimumAccessRole
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
        </form>
    );
}

export default AuthProviderForm;
