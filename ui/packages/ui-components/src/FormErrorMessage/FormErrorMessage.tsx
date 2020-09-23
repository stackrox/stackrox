import React, { ReactElement } from 'react';
import PropTypes, { InferProps } from 'prop-types';
import { ErrorMessage } from 'formik';

function FormErrorMessage({ name }: FormErrorMessageProps): ReactElement {
    return (
        <ErrorMessage name={name}>
            {(msg: string): ReactElement => (
                <div className="bg-alert-300 mt-2 p-2 text-alert-800 text-base rounded">{msg}</div>
            )}
        </ErrorMessage>
    );
}

FormErrorMessage.propTypes = {
    name: PropTypes.string.isRequired,
};

FormErrorMessage.defaultProps = {};

export type FormErrorMessageProps = InferProps<typeof FormErrorMessage.propTypes>;
export default FormErrorMessage;
