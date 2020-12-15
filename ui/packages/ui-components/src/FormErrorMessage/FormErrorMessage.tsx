import React, { ReactElement } from 'react';
import { ErrorMessage } from 'formik';

export type FormErrorMessageProps = {
    name: string;
};

function FormErrorMessage({ name }: FormErrorMessageProps): ReactElement {
    return (
        <ErrorMessage name={name}>
            {(msg: string): ReactElement => (
                <div className="bg-alert-300 mt-2 p-2 text-alert-800 text-base rounded">{msg}</div>
            )}
        </ErrorMessage>
    );
}

export default FormErrorMessage;
