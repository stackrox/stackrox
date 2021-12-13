import React, { ReactElement, useState } from 'react';
import { Button, Form, Modal, ModalVariant, TextArea } from '@patternfly/react-core';
import * as yup from 'yup';

import FormMessage, { FormResponseMessage } from 'Components/PatternFly/FormMessage';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import { useFormik } from 'formik';
import FormLabelGroup from 'Containers/Integrations/IntegrationForm/FormLabelGroup';
import pluralize from 'pluralize';
import { VulnerabilityRequest } from '../vulnerabilityRequests.graphql';

export type ApproveFalsePositiveFormValues = {
    comment: string;
};

export type ApproveFalsePositiveModalProps = {
    isOpen: boolean;
    vulnerabilityRequests: VulnerabilityRequest[];
    onSendRequest: (values: ApproveFalsePositiveFormValues) => Promise<FormResponseMessage>;
    onCompleteRequest: () => void;
    onCancel: () => void;
};

const validationSchema = yup.object().shape({
    comment: yup.string().trim().required('A deferral rationale is required'),
});

function ApproveFalsePositiveModal({
    isOpen,
    vulnerabilityRequests,
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

    const numRequestsToBeAssessed = vulnerabilityRequests.length;
    const numImpactedDeployments = vulnerabilityRequests.reduce((acc, curr) => {
        return acc + curr.deploymentCount;
    }, 0);
    const numImpactedImages = vulnerabilityRequests.reduce((acc, curr) => {
        return acc + curr.imageCount;
    }, 0);

    const title = `Approve false positives (${numRequestsToBeAssessed})`;

    return (
        <Modal
            variant={ModalVariant.small}
            title={title}
            isOpen={isOpen}
            onClose={onCancelHandler}
            actions={[
                <Button
                    key="confirm"
                    variant="primary"
                    onClick={onHandleSubmit}
                    isLoading={formik.isSubmitting}
                    isDisabled={formik.isSubmitting}
                >
                    Submit approval
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
                Marking CVEs as false positive can have cascading effects. For example, a false
                positive of a CVE at a component level can impact findings for any deployment of
                image using it.
            </div>
            <div className="pf-u-pb-md pf-u-danger-color-200">
                This active will impact {numImpactedDeployments}{' '}
                {pluralize('deployment', numImpactedDeployments)} and {numImpactedImages}{' '}
                {pluralize('image', numImpactedImages)}
            </div>
            <Form>
                <FormLabelGroup
                    label="Approval rationale"
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
