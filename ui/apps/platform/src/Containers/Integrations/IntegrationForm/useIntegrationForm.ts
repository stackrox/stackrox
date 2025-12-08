import { useState } from 'react';
import { useFormik } from 'formik';
import type { FormikProps, FormikValues } from 'formik';
import type { Schema } from 'yup';

import type { IntegrationOptions } from 'services/IntegrationsService';

import type { FormResponseMessage } from 'Components/PatternFly/FormMessage';
import useIntegrationActions from '../hooks/useIntegrationActions';

export type UseIntegrationForm<T> = {
    initialValues: T;
    validationSchema: Schema | (() => Schema);
};

export type UseIntegrationFormResult<T> = FormikProps<T> & {
    isTesting: boolean;
    onSave: (options?: IntegrationOptions) => void;
    onTest: (options?: IntegrationOptions) => void;
    onCancel: () => void;
    message: FormResponseMessage;
};

function useIntegrationForm<T extends FormikValues>({
    initialValues,
    validationSchema,
}: UseIntegrationForm<T>): UseIntegrationFormResult<T> {
    const { onSave, onTest, onCancel } = useIntegrationActions();
    // we will submit the form when clicking "Test" or "Create" so this value will distinguish
    // between the two
    const [isTesting, setIsTesting] = useState(false);
    const [options, setOptions] = useState<IntegrationOptions>({});
    // This message will be displayed in a banner using the response we get from either creating
    // or testing an integration
    const [message, setMessage] = useState<FormResponseMessage>(null);
    const formik = useFormik<T>({
        initialValues,
        onSubmit: (formValues) => {
            if (isTesting) {
                const response = onTest(formValues, options);
                return response;
            }
            const response = onSave(formValues, options);
            return response;
        },
        validationSchema,
        validateOnMount: true,
    });

    const { submitForm } = formik;

    function scrollToFormAlert() {
        const alertEl = document.getElementById('form-message-alert');

        if (alertEl) {
            alertEl.scrollIntoView({ behavior: 'smooth' });
        }
    }

    async function onTestHandler(optionsArg = {}) {
        setMessage(null);
        setIsTesting(true);
        setOptions(optionsArg);
        const response = await submitForm();
        setMessage(response);
        scrollToFormAlert();
    }

    async function onSaveHandler(optionsArg = {}) {
        setMessage(null);
        setIsTesting(false);
        setOptions(optionsArg);
        const response = await submitForm();
        setMessage(response);
        scrollToFormAlert();
    }

    return {
        ...formik,
        isTesting,
        onSave: onSaveHandler,
        onTest: onTestHandler,
        onCancel,
        message,
    };
}

export default useIntegrationForm;
