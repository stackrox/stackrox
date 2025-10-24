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
    children: ReactElement;
    helperText?: string;
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
    const showError = touched === undefined ? error : isTouched && error;
    const validated = showError ? ValidatedOptions.error : ValidatedOptions.default;

    return (
        <FormGroup fieldId={fieldId} {...rest}>
            {children}
            <FormHelperText>
                <HelperText id={`${fieldId}-helper`}>
                    <HelperTextItem variant={validated}>
                        {showError ? error : helperText}
                    </HelperTextItem>
                </HelperText>
            </FormHelperText>
        </FormGroup>
    );
}

export default FormLabelGroup;
