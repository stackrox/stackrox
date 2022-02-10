import React, { ReactElement, useState } from 'react';
import { Button, Form, Modal, ModalVariant, Radio, TextArea } from '@patternfly/react-core';
import { useFormik } from 'formik';
import * as yup from 'yup';

import FormLabelGroup from 'Containers/Integrations/IntegrationForm/FormLabelGroup';
import FormMessage, { FormResponseMessage } from 'Components/PatternFly/FormMessage';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import { ExpiresOn } from '../utils/vulnRequestFormUtils';

export type DeferralFormValues = {
    expiresOn: ExpiresOn;
    imageAppliesTo: string;
    comment: string;
};

export type DeferralFormModalProps = {
    isOpen: boolean;
    numCVEsToBeAssessed: number;
    onSendRequest: (values: DeferralFormValues) => Promise<FormResponseMessage>;
    onCompleteRequest: () => void;
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

function DeferralFormModal({
    isOpen,
    numCVEsToBeAssessed,
    onSendRequest,
    onCompleteRequest,
    onCancelDeferral,
}: DeferralFormModalProps): ReactElement {
    const [message, setMessage] = useState<FormResponseMessage>(null);
    const formik = useFormik<DeferralFormValues>({
        initialValues: {
            expiresOn: 'Until Fixable',
            imageAppliesTo: '',
            comment: '',
        },
        validationSchema,
        onSubmit: (values: DeferralFormValues) => {
            const response = onSendRequest(values);
            return response;
        },
    });

    function onChange(value, event) {
        return formik.setFieldValue(event.target.id, value);
    }

    function onExpiresOnChange(_, event) {
        return formik.setFieldValue('expiresOn', event.target.value);
    }

    function onImageAppliesToOnChange(_, event) {
        return formik.setFieldValue('imageAppliesTo', event.target.value);
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
        onCancelDeferral();
    }

    const title = `Mark CVEs for deferral (${numCVEsToBeAssessed})`;

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

export default DeferralFormModal;
