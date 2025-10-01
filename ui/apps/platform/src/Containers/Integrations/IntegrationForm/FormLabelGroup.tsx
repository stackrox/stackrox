import React from 'react';
import type { ReactElement } from 'react';
import {
    FormGroup,
    FormHelperText,
    HelperText,
    HelperTextItem,
    ValidatedOptions,
} from '@patternfly/react-core';
import type { FormGroupProps } from '@patternfly/react-core';
import type { FormikTouched, FormikErrors } from 'formik';
import get from 'lodash/get';

export interface FormLabelGroupProps<T> extends FormGroupProps {
    fieldId: string;
    touched?: FormikTouched<T>;
    errors: FormikErrors<T>;
    children: ReactElement | ReactElement[];
    helperText?: string | ReactElement;
}

function FormLabelGroup<T>({
    fieldId,
    touched,
    errors,
    children,
    helperText,
    ...rest
}: FormLabelGroupProps<T>): ReactElement {
    const error = get(errors, fieldId);
    const isTouched = touched && get(touched, fieldId);

    return (
        <FormGroup fieldId={fieldId} {...rest}>
            {children}
            <FormHelperText>
                <HelperText id={`${fieldId}-helper`}>
                    {isTouched && error ? (
                        <HelperTextItem variant={ValidatedOptions.error}>{error}</HelperTextItem>
                    ) : (
                        <HelperTextItem variant={ValidatedOptions.default}>
                            {helperText}
                        </HelperTextItem>
                    )}
                </HelperText>
            </FormHelperText>
        </FormGroup>
    );
}

export default FormLabelGroup;
