import React, { ReactElement, useEffect, useState } from 'react';
import { Button, Form, Modal, ModalVariant, Radio, TextArea } from '@patternfly/react-core';
import { useFormik } from 'formik';
import * as yup from 'yup';

import FormLabelGroup from 'Containers/Integrations/IntegrationForm/FormLabelGroup';
import FormMessage, { FormResponseMessage } from 'Components/PatternFly/FormMessage';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

export type DeferralFormValues = {
    expiresOn: string;
    imageAppliesTo: string;
    comment: string;
};

export type DeferralRequestModalProps = {
    isOpen: boolean;
    onCompleteDeferral: (values: DeferralFormValues) => Promise<FormResponseMessage>;
    onCancelDeferral: () => void;
};

const EXPIRES_ON = {
    UNTIL_FIXABLE: 'Until Fixable',
    TWO_WEEKS: '2 weeks',
    THIRTY_DAYS: '30 days',
    NINETY_DAYS: '90 days',
    INDEFINITELY: 'Indefinitely',
};

const IMAGE_APPLIES_TO = {
    ALL_TAGS_WITHIN_IMAGE: 'All tags within image',
    ONLY_THIS_IMAGE_TAG: 'Only this image tag',
};

const validationSchema = yup.object().shape({
    expiresOn: yup
        .string()
        .trim()
        .oneOf(Object.values(EXPIRES_ON))
        .required('An expiry time is required'),
    imageAppliesTo: yup
        .string()
        .trim()
        .oneOf(Object.values(IMAGE_APPLIES_TO))
        .required('An image scope is required'),
    comment: yup.string().trim().required('A deferral rationale is required'),
});

function DeferralRequestModal({
    isOpen,
    onCompleteDeferral,
    onCancelDeferral,
}: DeferralRequestModalProps): ReactElement {
    const [message, setMessage] = useState<FormResponseMessage>(null);
    const formik = useFormik<DeferralFormValues>({
        initialValues: {
            expiresOn: '',
            imageAppliesTo: '',
            comment: '',
        },
        validationSchema,
        onSubmit: (values: DeferralFormValues) => {
            const response = onCompleteDeferral(values);
            return response;
        },
    });

    useEffect(() => {
        // since the modal doesn't disappear, this will clear the modal form data whenever it becomes non-visible
        if (isOpen === false) {
            setMessage(null);
            formik.resetForm();
        }
    }, [formik, isOpen]);

    function onChange(value, event) {
        return formik.setFieldValue(event.target.id, value);
    }

    function onExpiresOnChange(_, event) {
        return formik.setFieldValue('expiresOn', event.target.value);
    }

    function onImageAppliesToOnChange(_, event) {
        return formik.setFieldValue('imageAppliesTo', event.target.value);
    }

    async function onHandleSubmit() {
        setMessage(null);
        const response = await formik.submitForm().catch((error) => {
            setMessage({
                message: getAxiosErrorMessage(error),
                isError: true,
            });
        });
        if (response) {
            setMessage(response);
        }
    }

    return (
        <Modal
            variant={ModalVariant.small}
            title="Mark CVEs for deferral"
            isOpen={isOpen}
            onClose={onCancelDeferral}
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
                    onClick={onCancelDeferral}
                    isDisabled={formik.isSubmitting}
                >
                    Cancel
                </Button>,
            ]}
        >
            <FormMessage message={message} />
            <div className="pf-u-pb-md">
                CVEs will be marked as deferred after approval by your security lead.
            </div>
            <Form>
                <FormLabelGroup
                    label="How long should the CVEs be deferred?"
                    isRequired
                    fieldId="expiresOn"
                    touched={formik.touched}
                    errors={formik.errors}
                >
                    <Radio
                        id="expiresOnUntilFixable"
                        name="expiresOn"
                        label={EXPIRES_ON.UNTIL_FIXABLE}
                        value={EXPIRES_ON.UNTIL_FIXABLE}
                        isChecked={formik.values.expiresOn === EXPIRES_ON.UNTIL_FIXABLE}
                        onChange={onExpiresOnChange}
                    />
                    <Radio
                        id="expiresOnTwoWeeks"
                        name="expiresOn"
                        label={EXPIRES_ON.TWO_WEEKS}
                        value={EXPIRES_ON.TWO_WEEKS}
                        isChecked={formik.values.expiresOn === EXPIRES_ON.TWO_WEEKS}
                        onChange={onExpiresOnChange}
                    />
                    <Radio
                        id="expiresOnThirtyDays"
                        name="expiresOn"
                        label={EXPIRES_ON.THIRTY_DAYS}
                        value={EXPIRES_ON.THIRTY_DAYS}
                        isChecked={formik.values.expiresOn === EXPIRES_ON.THIRTY_DAYS}
                        onChange={onExpiresOnChange}
                    />
                    <Radio
                        id="expiresOnNinetyDays"
                        name="expiresOn"
                        label={EXPIRES_ON.NINETY_DAYS}
                        value={EXPIRES_ON.NINETY_DAYS}
                        isChecked={formik.values.expiresOn === EXPIRES_ON.NINETY_DAYS}
                        onChange={onExpiresOnChange}
                    />
                    <Radio
                        id="expiresOnIndefinitely"
                        name="expiresOn"
                        label={EXPIRES_ON.INDEFINITELY}
                        value={EXPIRES_ON.INDEFINITELY}
                        isChecked={formik.values.expiresOn === EXPIRES_ON.INDEFINITELY}
                        onChange={onExpiresOnChange}
                    />
                </FormLabelGroup>
                <FormLabelGroup
                    label="What should the deferral apply to?"
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
                    label="Deferral rationale"
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

export default DeferralRequestModal;
