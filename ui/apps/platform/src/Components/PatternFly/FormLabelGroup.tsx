import React, { ReactElement } from 'react';
import { FormGroup, FormGroupProps, ValidatedOptions } from '@patternfly/react-core';
import { FormikTouched, FormikErrors } from 'formik';
import get from 'lodash/get';

export interface FormLabelGroupProps<T> extends FormGroupProps {
    fieldId: string;
    touched?: FormikTouched<T>;
    errors: FormikErrors<T>;
    children: ReactElement;
}

function FormLabelGroup<T>({
    fieldId,
    touched,
    errors,
    children,
    ...rest
}: FormLabelGroupProps<T>): ReactElement {
    const error = get(errors, fieldId);
    const isTouched = touched && get(touched, fieldId);
    const showError = touched === undefined ? error : isTouched && error;

    return (
        <FormGroup
            fieldId={fieldId}
            helperTextInvalid={error}
            validated={showError ? ValidatedOptions.error : ValidatedOptions.default}
            {...rest}
        >
            {children}
        </FormGroup>
    );
}

export default FormLabelGroup;
