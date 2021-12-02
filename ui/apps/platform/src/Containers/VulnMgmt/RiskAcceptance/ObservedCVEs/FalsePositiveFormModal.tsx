import React, { ReactElement, useState } from 'react';
import { Button, Form, Modal, ModalVariant, Radio, TextArea } from '@patternfly/react-core';
import { useFormik } from 'formik';
import * as yup from 'yup';

import FormLabelGroup from 'Containers/Integrations/IntegrationForm/FormLabelGroup';
import FormMessage, { FormResponseMessage } from 'Components/PatternFly/FormMessage';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

export type FalsePositiveFormValues = {
    imageAppliesTo: string;
    comment: string;
};

export type FalsePositiveFormModalProps = {
    isOpen: boolean;
    numCVEsToBeAssessed: number;
    onSendRequest: (values: FalsePositiveFormValues) => Promise<FormResponseMessage>;
    onCompleteRequest: () => void;
    onCancelFalsePositive: () => void;
};

const IMAGE_APPLIES_TO = {
    ALL_TAGS_WITHIN_IMAGE: 'All tags within image',
    ONLY_THIS_IMAGE_TAG: 'Only this image tag',
};

const validationSchema = yup.object().shape({
    imageAppliesTo: yup
        .string()
        .trim()
        .oneOf(Object.values(IMAGE_APPLIES_TO))
        .required('An image scope is required'),
    comment: yup.string().trim().required('A deferral rationale is required'),
});

function FalsePositiveFormModal({
    isOpen,
    numCVEsToBeAssessed,
    onSendRequest,
    onCompleteRequest,
    onCancelFalsePositive,
}: FalsePositiveFormModalProps): ReactElement {
    const [message, setMessage] = useState<FormResponseMessage>(null);
    const formik = useFormik<FalsePositiveFormValues>({
        initialValues: {
            imageAppliesTo: '',
            comment: '',
        },
        validationSchema,
        onSubmit: (values: FalsePositiveFormValues) => {
            const response = onSendRequest(values);
            return response;
        },
    });

    function onChange(value, event) {
        return formik.setFieldValue(event.target.id, value);
    }

    function onImageAppliesToOnChange(_, event) {
        return formik.setFieldValue('imageAppliesTo', event.target.value);
    }

    async function onHandleSubmit() {
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
        onCancelFalsePositive();
    }

    const title = `Mark CVEs as false positive (${numCVEsToBeAssessed})`;

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
                    isDisabled={formik.isSubmitting || !formik.dirty || !formik.isValid}
                    isLoading={formik.isSubmitting}
                >
                    Request approval
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
                CVEs will be marked as false positive and removed from the vulnerability management
                workflow
            </div>
            <Form>
                <FormLabelGroup
                    label="Mark as false positive for:"
                    isRequired
                    fieldId="imageAppliesTo"
                    touched={formik.touched}
                    errors={formik.errors}
                >
                    <Radio
                        id="appliesToAllTagsWithinImage"
                        name="imageAppliesTo"
                        label={IMAGE_APPLIES_TO.ALL_TAGS_WITHIN_IMAGE}
                        value={IMAGE_APPLIES_TO.ALL_TAGS_WITHIN_IMAGE}
                        isChecked={
                            formik.values.imageAppliesTo === IMAGE_APPLIES_TO.ALL_TAGS_WITHIN_IMAGE
                        }
                        onChange={onImageAppliesToOnChange}
                    />
                    <Radio
                        id="appliesToOnlyThisImage"
                        name="imageAppliesTo"
                        label={IMAGE_APPLIES_TO.ONLY_THIS_IMAGE_TAG}
                        value={IMAGE_APPLIES_TO.ONLY_THIS_IMAGE_TAG}
                        isChecked={
                            formik.values.imageAppliesTo === IMAGE_APPLIES_TO.ONLY_THIS_IMAGE_TAG
                        }
                        onChange={onImageAppliesToOnChange}
                    />
                </FormLabelGroup>
                <FormLabelGroup
                    label="False positive rationale"
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

export default FalsePositiveFormModal;
