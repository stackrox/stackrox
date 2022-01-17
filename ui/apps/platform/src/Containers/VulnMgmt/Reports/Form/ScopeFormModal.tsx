import React, { ReactElement, useState } from 'react';
import {
    Button,
    ButtonVariant,
    Form,
    Modal,
    ModalVariant,
    Radio,
    TextArea,
    Title,
    TitleSizes,
} from '@patternfly/react-core';
import { useFormik } from 'formik';
import * as yup from 'yup';

import FormLabelGroup from 'Containers/Integrations/IntegrationForm/FormLabelGroup';
import FormMessage, { FormResponseMessage } from 'Components/PatternFly/FormMessage';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

export type ScopeFormValues = {
    imageAppliesTo: string;
    comment: string;
};

// TODO remove
const IMAGE_APPLIES_TO = {
    ALL_TAGS_WITHIN_IMAGE: 'All tags within image',
    ONLY_THIS_IMAGE_TAG: 'Only this image tag',
};
// TODO end remove

const validationSchema = yup.object().shape({
    imageAppliesTo: yup.string().trim().required('An image scope is required'),
    comment: yup.string().trim().required('A deferral rationale is required'),
});

export type ScopeFormModalProps = {
    isOpen: boolean;
    onSendRequest: (values: ScopeFormValues) => Promise<FormResponseMessage>;
    onCompleteRequest: () => void;
    onCancelScopeModal: () => void;
};

function ScopeFormModal({
    isOpen,
    onSendRequest,
    onCompleteRequest,
    onCancelScopeModal,
}: ScopeFormModalProps): ReactElement {
    const [message, setMessage] = useState<FormResponseMessage>(null);
    const formik = useFormik<ScopeFormValues>({
        initialValues: {
            imageAppliesTo: '',
            comment: '',
        },
        validationSchema,
        onSubmit: (values: ScopeFormValues) => {
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
        onCancelScopeModal();
    }

    const title = 'Create resource scope';

    const header = (
        <>
            <Title id="custom-header-label" headingLevel="h1" size={TitleSizes.xl}>
                {title}
            </Title>
            <p className="pf-u-pt-sm">
                Add predefined sets of Kubernetes resources that users should be able to access.
            </p>
        </>
    );

    return (
        <Modal
            variant={ModalVariant.large}
            header={header}
            isOpen={isOpen}
            onClose={onCancelHandler}
            actions={[
                <Button
                    key="save-scope"
                    variant={ButtonVariant.primary}
                    onClick={onHandleSubmit}
                    isDisabled={formik.isSubmitting || !formik.dirty || !formik.isValid}
                    isLoading={formik.isSubmitting}
                >
                    Save resource scope
                </Button>,
                <Button
                    key="cancel-modal"
                    variant={ButtonVariant.link}
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
                    label="Scope rationale"
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

export default ScopeFormModal;
