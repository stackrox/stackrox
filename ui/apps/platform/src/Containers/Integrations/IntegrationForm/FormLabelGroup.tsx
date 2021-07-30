import React, { ReactElement } from 'react';
import { FormGroup, FormGroupProps } from '@patternfly/react-core';
import { FormikErrors } from 'formik';
import get from 'lodash/get';

export interface FormLabelGroupProps<T> extends FormGroupProps {
    fieldId: string;
    errors: FormikErrors<T>;
    children: ReactElement | ReactElement[];
}

function FormLabelGroup<T>({
    fieldId,
    errors,
    children,
    ...rest
}: FormLabelGroupProps<T>): ReactElement {
    const error = get(errors, fieldId);
    return (
        <FormGroup
            fieldId={fieldId}
            {...rest}
            helperTextInvalid={error}
            validated={error ? 'error' : 'default'}
        >
            {children}
        </FormGroup>
    );
}

export default FormLabelGroup;
