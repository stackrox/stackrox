import React, { ReactElement, useEffect, useState } from 'react';
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

export type DeferralRequestModalProps = {
    isOpen: boolean;
    onRequestFalsePositive: (values: FalsePositiveFormValues) => Promise<FormResponseMessage>;
    onCompleteFalsePositive: () => void;
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

function FalsePositiveRequestModal({
    isOpen,
    onRequestFalsePositive,
    onCompleteFalsePositive,
    onCancelFalsePositive,
}: DeferralRequestModalProps): ReactElement {
    const [message, setMessage] = useState<FormResponseMessage>(null);
    const formik = useFormik<FalsePositiveFormValues>({
        initialValues: {
            imageAppliesTo: '',
            comment: '',
        },
        validationSchema,
        onSubmit: (values: FalsePositiveFormValues) => {
            const response = onRequestFalsePositive(values);
            return response;
        },
    });

    useEffect(() => {
        // since the modal doesn't disappear, this will clear the modal form data whenever it becomes non-visible
        if (isOpen === false && message !== null) {
            setMessage(null);
            formik.resetForm();
        }
    }, [formik, isOpen, message]);

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
            .then((response) => {
                setMessage(response);
                onCompleteFalsePositive();
            })
            .catch((response) => {
                const error = new Error(response.message);
                setMessage({
                    message: getAxiosErrorMessage(error),
                    isError: true,
                });
            });
    }

    return (
        <Modal
            variant={ModalVariant.small}
            title="Mark CVEs as false positive"
            isOpen={isOpen}
            onClose={onCancelFalsePositive}
            actions={[
                <Button
                    key="confirm"
                    variant="primary"
                    onClick={onHandleSubmit}
                    isDisabled={formik.isSubmitting}
                    isLoading={formik.isSubmitting}
                >
                    Request approval
                </Button>,
                <Button
                    key="cancel"
                    variant="link"
                    onClick={onCancelFalsePositive}
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

export default FalsePositiveRequestModal;
