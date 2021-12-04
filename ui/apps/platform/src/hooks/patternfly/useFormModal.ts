import { useState } from 'react';
import { FormikProps, useFormik } from 'formik';

import { FormResponseMessage } from 'Components/PatternFly/FormMessage';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

type UseFormModalProps<T> = {
    initialValues: T;
    validationSchema;
    onSendRequest: (values: T) => Promise<FormResponseMessage>;
    onCompleteRequest: () => void;
    onCancel: () => void;
};

type UseFormModalResults<T> = {
    formik: FormikProps<T>;
    message: FormResponseMessage;
    onHandleSubmit: () => void;
    onHandleCancel: () => void;
};

function useFormModal<T>({
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
            .then(() => {
                formik.resetForm();
                onCompleteRequest();
            })
            .catch((response) => {
                const error = new Error(response.message);
                setMessage({
                    message: getAxiosErrorMessage(error),
                    isError: true,
                });
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
