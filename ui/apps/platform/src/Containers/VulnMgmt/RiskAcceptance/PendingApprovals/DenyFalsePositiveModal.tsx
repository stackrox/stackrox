import React, { ReactElement, useState } from 'react';
import { Button, Form, Modal, ModalVariant, TextArea } from '@patternfly/react-core';
import * as yup from 'yup';

import FormMessage, { FormResponseMessage } from 'Components/PatternFly/FormMessage';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import { useFormik } from 'formik';
import FormLabelGroup from 'Containers/Integrations/IntegrationForm/FormLabelGroup';

export type ApproveFalsePositiveFormValues = {
    comment: string;
};

export type ApproveFalsePositiveModalProps = {
    isOpen: boolean;
    numRequestsToBeAssessed: number;
    onSendRequest: (values: ApproveFalsePositiveFormValues) => Promise<FormResponseMessage>;
    onCompleteRequest: () => void;
    onCancel: () => void;
};

const validationSchema = yup.object().shape({
    comment: yup.string().trim().required('A deferral rationale is required'),
});

function ApproveFalsePositiveModal({
    isOpen,
    numRequestsToBeAssessed,
    onSendRequest,
    onCompleteRequest,
    onCancel,
}: ApproveFalsePositiveModalProps): ReactElement {
    const [message, setMessage] = useState<FormResponseMessage>(null);
    const formik = useFormik<ApproveFalsePositiveFormValues>({
        initialValues: {
            comment: '',
        },
        validationSchema,
        onSubmit: (values: ApproveFalsePositiveFormValues) => {
            const response = onSendRequest(values);
            return response;
        },
    });

    function onHandleSubmit() {
        setMessage(null);
        formik
            .submitForm()
            .then(() => {
                setMessage(null);
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

    function onChange(value, event) {
        return formik.setFieldValue(event.target.id, value);
    }

    function onCancelHandler() {
        setMessage(null);
        onCancel();
    }

    const title = `Deny false positives (${numRequestsToBeAssessed})`;

    return (
        <Modal
            variant={ModalVariant.small}
            title={title}
            isOpen={isOpen}
            onClose={onCancelHandler}
            actions={[
                <Button
                    key="confirm"
                    variant="danger"
                    onClick={onHandleSubmit}
                    isLoading={formik.isSubmitting}
                    isDisabled={formik.isSubmitting}
                >
                    Submit denial
                </Button>,
                <Button
                    key="cancel"
                    variant="link"
                    onClick={onCancelHandler}
                    isDisabled={formik.isSubmitting}
                >
                    Cancel
                </Button>,
            ]}
        >
            <FormMessage message={message} />
            <Form>
                <FormLabelGroup
                    label="Denial rationale"
                    isRequired
                    fieldId="comment"
                    touched={formik.touched}
                    errors={formik.errors}
                >
                    <TextArea
                        isRequired
                        type="text"
                        id="comment"
                        value={formik.values.comment}
                        onChange={onChange}
                        onBlur={formik.handleBlur}
                    />
                </FormLabelGroup>
            </Form>
        </Modal>
    );
}

export default ApproveFalsePositiveModal;
