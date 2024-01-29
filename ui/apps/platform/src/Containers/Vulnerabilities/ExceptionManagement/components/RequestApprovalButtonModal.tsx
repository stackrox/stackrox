import React, { useState } from 'react';
import {
    Alert,
    AlertVariant,
    Button,
    Form,
    Modal,
    TextArea,
    pluralize,
} from '@patternfly/react-core';
import * as yup from 'yup';
import { useFormik } from 'formik';
import isEqual from 'lodash/isEqual';

import useModal from 'hooks/useModal';
import {
    VulnerabilityException,
    approveVulnerabilityException,
} from 'services/VulnerabilityExceptionService';
import FormLabelGroup from 'Components/PatternFly/FormLabelGroup';
import useRestMutation from 'hooks/useRestMutation';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

type RequestApprovalButtonModalProps = {
    exception: VulnerabilityException;
    onSuccess: (vulnerabilityException: VulnerabilityException) => void;
};

const initialValues = {
    rationale: '',
};

const validationSchema = yup.object().shape({
    rationale: yup.string().required('Approval rationale is required'),
});

function RequestApprovalButtonModal({ exception, onSuccess }: RequestApprovalButtonModalProps) {
    const approveRequestMutation = useRestMutation(approveVulnerabilityException);

    const { isModalOpen, openModal, closeModal } = useModal();
    const [errorMessage, setErrorMessage] = useState<string | null>(null);

    const formik = useFormik({
        initialValues,
        validationSchema,
        onSubmit: (values, helpers) => {
            const payload = {
                id: exception.id,
                comment: values.rationale,
            };
            approveRequestMutation.mutate(payload, {
                onSuccess: (exception) => {
                    onSuccess(exception);
                    onClose();
                },
                onError: (error) => {
                    setErrorMessage(getAxiosErrorMessage(error));
                    helpers.setSubmitting(false);
                },
            });
        },
    });

    function onClose() {
        formik.resetForm();
        closeModal();
    }

    const modalTitle = `Approve ${
        exception.targetState === 'DEFERRED' ? 'deferral' : 'false positive'
    } for ${pluralize(exception.cves.length, 'CVE')}`;

    const isFormModified = !isEqual(formik.values, formik.initialValues);
    const hasErrors = Object.keys(formik.errors).length > 0;
    const isSubmitDisabled = !isFormModified || hasErrors || formik.isSubmitting;

    return (
        <>
            <Button variant="primary" onClick={openModal}>
                Approve request
            </Button>
            <Modal
                variant="small"
                title={modalTitle}
                isOpen={isModalOpen}
                onClose={onClose}
                actions={[
                    <Button
                        key="approve"
                        variant="primary"
                        isLoading={formik.isSubmitting}
                        isDisabled={isSubmitDisabled}
                        onClick={() => formik.handleSubmit()}
                    >
                        Approve
                    </Button>,
                    <Button key="cancel" variant="link" onClick={onClose}>
                        Cancel
                    </Button>,
                ]}
                showClose={false}
            >
                {errorMessage && (
                    <Alert isInline variant={AlertVariant.danger} title={errorMessage} />
                )}
                <Form>
                    <FormLabelGroup
                        isRequired
                        label="Approval rationale"
                        fieldId="rationale"
                        errors={formik.errors}
                    >
                        <TextArea
                            type="text"
                            id="rationale"
                            value={formik.values.rationale}
                            onChange={(_, event) => formik.handleChange(event)}
                            onBlur={formik.handleBlur}
                        />
                    </FormLabelGroup>
                </Form>
            </Modal>
        </>
    );
}

export default RequestApprovalButtonModal;
