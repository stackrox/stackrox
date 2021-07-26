import { useState } from 'react';
import { FormikErrors, useFormik } from 'formik';

import useIntegrationActions from '../hooks/useIntegrationActions';

export type FormResponseMessage = {
    message: string;
    isError: boolean;
} | null;

export type UseIntegrationForm<T, V> = {
    initialValues: T;
    validationSchema: V;
};

export type UseIntegrationFormResult<T> = {
    values: T;
    errors: FormikErrors<T>;
    setFieldValue: (
        field: string,
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        value: any,
        shouldValidate?: boolean | undefined
    ) => Promise<void> | Promise<FormikErrors<T>>;
    isSubmitting: boolean;
    isTesting: boolean;
    onSave: () => void;
    onTest: () => void;
    onCancel: () => void;
    message: FormResponseMessage;
};

function useIntegrationForm<T, V>({
    initialValues,
    validationSchema,
}: UseIntegrationForm<T, V>): UseIntegrationFormResult<T> {
    const { onSave, onTest, onCancel } = useIntegrationActions();
    // we will submit the form when clicking "Test" or "Create" so this value will distinguish
    // between the two
    const [isTesting, setIsTesting] = useState(false);
    // This message will be displayed in a banner using the response we get from either creating
    // or testing an integration
    const [message, setMessage] = useState<FormResponseMessage>(null);
    const { values, errors, setFieldValue, submitForm, isSubmitting } = useFormik<T>({
        initialValues,
        onSubmit: (formValues) => {
            if (isTesting) {
                const response = onTest(formValues);
                return response;
            }
            const response = onSave(formValues);
            return response;
        },
        validationSchema,
    });

    async function onTestHandler() {
        setIsTesting(true);
        const response = await submitForm();
        setMessage(response);
    }

    async function onSaveHandler() {
        setIsTesting(false);
        const response = await submitForm();
        setMessage(response);
    }

    return {
        values,
        errors,
        setFieldValue,
        isSubmitting,
        isTesting,
        onSave: onSaveHandler,
        onTest: onTestHandler,
        onCancel,
        message,
    };
}

export default useIntegrationForm;
