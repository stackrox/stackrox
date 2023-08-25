import { useState } from 'react';
import { FormikProps, FormikValues, useFormik } from 'formik';

import { FormResponseMessage } from 'Components/PatternFly/FormMessage';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

type UseFormModalProps<T> = {
    initialValues: T;
    validationSchema;
    onSendRequest: (values: T) => Promise<FormResponseMessage>;
    onCompleteRequest: (any) => void;
    onCancel: () => void;
};

type UseFormModalResults<T> = {
    formik: FormikProps<T>;
    message: FormResponseMessage;
    onHandleSubmit: () => void;
    onHandleCancel: () => void;
};

function useFormModal<T extends FormikValues>({
    initialValues,
    validationSchema,
    onSendRequest,
    onCompleteRequest,
    onCancel,
}: UseFormModalProps<T>): UseFormModalResults<T> {
    const [message, setMessage] = useState<FormResponseMessage>(null);
    const formik = useFormik<T>({
        initialValues,
        validationSchema,
        onSubmit: (values: T) => {
            const response = onSendRequest(values);
            return response;
        },
    });

    function onHandleSubmit() {
        setMessage(null);
        formik
            .submitForm()
            .then((response) => {
                formik.resetForm();
                onCompleteRequest(response);
            })
            .catch((response) => {
                const extractedMessage = response?.response?.data?.message || response?.message;
                const error = new Error(extractedMessage);
                setMessage({
                    message: getAxiosErrorMessage(error),
                    isError: true,
                });

                // TODO: factor out and increase robustness of the following
                //       scroll to error behavior
                const container = document.querySelector('.pf-c-modal-box__body'); // PF modal body element
                const alertEl = document.getElementById('form-message-alert'); // PF alert message element
                if (container && alertEl) {
                    container.scrollTop = alertEl.offsetTop - container.scrollTop;
                }
            });
    }

    function onHandleCancel() {
        setMessage(null);
        formik.resetForm();
        onCancel();
    }

    return { formik, message, onHandleSubmit, onHandleCancel };
}

export default useFormModal;
