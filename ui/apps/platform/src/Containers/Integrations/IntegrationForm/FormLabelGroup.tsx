import React, { ReactElement } from 'react';
import {
    FormGroup,
    FormGroupProps,
    FormHelperText,
    HelperText,
    HelperTextItem,
    ValidatedOptions,
} from '@patternfly/react-core';
import { FormikTouched, FormikErrors } from 'formik';
import get from 'lodash/get';

export interface FormLabelGroupProps<T> extends FormGroupProps {
    fieldId: string;
    touched?: FormikTouched<T>;
    errors: FormikErrors<T>;
    children: ReactElement | ReactElement[];
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

    return (
        <FormGroup fieldId={fieldId} {...rest}>
            <FormHelperText>
                <HelperText>
                    <HelperTextItem
                        variant={
                            isTouched && error ? ValidatedOptions.error : ValidatedOptions.default
                        }
                    >
                        {error}
                    </HelperTextItem>
                </HelperText>
            </FormHelperText>
            {children}
        </FormGroup>
    );
}

export default FormLabelGroup;
