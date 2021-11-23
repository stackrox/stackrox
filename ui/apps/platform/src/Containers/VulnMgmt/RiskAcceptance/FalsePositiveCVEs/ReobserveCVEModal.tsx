import React, { ReactElement, useState } from 'react';
import { Button, Form, Modal, ModalVariant, TextArea } from '@patternfly/react-core';
import { useFormik } from 'formik';
import * as yup from 'yup';

import FormLabelGroup from 'Containers/Integrations/IntegrationForm/FormLabelGroup';
import FormMessage, { FormResponseMessage } from 'Components/PatternFly/FormMessage';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

export type CancelFormValues = {
    comment: string;
};

export type ReobserveCVEModalProps = {
    isOpen: boolean;
    onSendRequest: (values: CancelFormValues) => Promise<FormResponseMessage>;
    onCompleteRequest: () => void;
    onCancel: () => void;
};

const validationSchema = yup.object().shape({
    comment: yup.string().trim().required('A comment is required'),
});

function ReobserveCVEModal({
    isOpen,
    onSendRequest,
    onCompleteRequest,
    onCancel,
}: ReobserveCVEModalProps): ReactElement {
    const [message, setMessage] = useState<FormResponseMessage>(null);
    const formik = useFormik<CancelFormValues>({
        initialValues: {
            comment: '',
        },
        validationSchema,
        onSubmit: (values: CancelFormValues) => {
            const response = onSendRequest(values);
            return response;
        },
    });

    function onChange(value, event) {
        return formik.setFieldValue(event.target.id, value);
    }

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

    function onCancelHandler() {
        setMessage(null);
        formik.resetForm();
        onCancel();
    }

    return (
        <Modal
            variant={ModalVariant.small}
            title="Reobserve CVE"
            isOpen={isOpen}
            onClose={onCancelHandler}
            actions={[
                <Button
                    key="confirm"
                    variant="primary"
                    onClick={onHandleSubmit}
                    isDisabled={formik.isSubmitting || !formik.dirty || !formik.isValid}
                    isLoading={formik.isSubmitting}
                >
                    Reobserve CVE
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
            <div className="pf-u-pb-md">
                Reobserving a false positive will return the CVE into the vulnerability management
                workflow.
            </div>
            <Form>
                <FormLabelGroup
                    isRequired
                    label="Comment"
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
                        placeholder="Enter an appropriate reason to reobserve"
                    />
                </FormLabelGroup>
            </Form>
        </Modal>
    );
}

export default ReobserveCVEModal;
