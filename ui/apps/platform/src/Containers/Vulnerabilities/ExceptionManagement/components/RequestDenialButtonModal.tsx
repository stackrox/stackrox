import React, { useState } from 'react';
import {
    Alert,
    AlertVariant,
    Button,
    Flex,
    Form,
    Modal,
    Spinner,
    Text,
    TextArea,
} from '@patternfly/react-core';
import * as yup from 'yup';
import { useFormik } from 'formik';
import isEqual from 'lodash/isEqual';

import useModal from 'hooks/useModal';
import {
    VulnerabilityException,
    denyVulnerabilityException,
} from 'services/VulnerabilityExceptionService';
import useRestMutation from 'hooks/useRestMutation';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import useRequestCVEsDetails from 'Containers/Vulnerabilities/ExceptionManagement/hooks/useRequestCVEsDetails';

import FormLabelGroup from 'Components/PatternFly/FormLabelGroup';

type RequestDenialButtonModalProps = {
    exception: VulnerabilityException;
    onSuccess: (vulnerabilityException: VulnerabilityException) => void;
};

const initialValues = {
    rationale: '',
};

const validationSchema = yup.object().shape({
    rationale: yup.string().required('Denial rationale is required'),
});

function RequestDenialButtonModal({ exception, onSuccess }: RequestDenialButtonModalProps) {
    const denyRequestMutation = useRestMutation(denyVulnerabilityException);
    const { isLoading: isRequestCVEsDetailsLoading, totalAffectedImageCount } =
        useRequestCVEsDetails(exception);

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
            denyRequestMutation.mutate(payload, {
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

    const isFormModified = !isEqual(formik.values, formik.initialValues);
    const hasErrors = Object.keys(formik.errors).length > 0;
    const isSubmitDisabled = !isFormModified || hasErrors || formik.isSubmitting;

    return (
        <>
            <Button variant="secondary" onClick={openModal}>
                Deny request
            </Button>
            <Modal
                variant="small"
                title="Deny request"
                isOpen={isModalOpen}
                onClose={onClose}
                actions={[
                    <Button
                        key="approve"
                        variant="danger"
                        isLoading={formik.isSubmitting}
                        isDisabled={isSubmitDisabled}
                        onClick={() => formik.handleSubmit()}
                    >
                        Deny
                    </Button>,
                    <Button key="cancel" variant="link" onClick={onClose}>
                        Cancel
                    </Button>,
                ]}
                showClose={false}
            >
                <Flex direction={{ default: 'column' }}>
                    {errorMessage && (
                        <Alert isInline variant={AlertVariant.danger} title={errorMessage} />
                    )}
                    <Alert
                        variant="warning"
                        isInline
                        title="Denying the request will return the CVEs to the 'Observed' status."
                    >
                        <Text>CVE count: {exception.cves.length}</Text>
                        <Text>
                            Affected images:{' '}
                            {isRequestCVEsDetailsLoading ? (
                                <Spinner
                                    isSVG
                                    size="md"
                                    aria-label="Loading affected images count"
                                />
                            ) : (
                                totalAffectedImageCount
                            )}
                        </Text>
                    </Alert>
                    <Form>
                        <FormLabelGroup
                            isRequired
                            label="Denial rationale"
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
                </Flex>
            </Modal>
        </>
    );
}

export default RequestDenialButtonModal;
