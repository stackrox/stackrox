import React, { ReactElement, useEffect } from 'react';
import { Button, Modal, Tab, TabTitleText, Tabs } from '@patternfly/react-core';
import { useFormik } from 'formik';
import isEmpty from 'lodash/isEmpty';

import { EmailTemplateFormData, emailTemplateValidationSchema } from './EmailTemplate.utils';
import EmailTemplateForm from './EmailTemplateForm';

export type TemplatePreviewArgs = {
    customBody: string;
    customSubject: string;
    customSubjectDefault: string;
};

export type EmailTemplateModalProps = {
    isOpen: boolean;
    onClose: () => void;
    onChange: (formData: EmailTemplateFormData) => void;
    customBodyDefault: string;
    customBodyInitial: string;
    customSubjectDefault: string;
    customSubjectInitial: string;
    renderTemplatePreview?: (args: TemplatePreviewArgs) => ReactElement;
};

function EmailTemplateModal({
    isOpen,
    onClose,
    onChange,
    customBodyDefault,
    customBodyInitial,
    customSubjectDefault,
    customSubjectInitial,
    renderTemplatePreview,
}: EmailTemplateModalProps) {
    const formik = useFormik<EmailTemplateFormData>({
        initialValues: { customSubject: customSubjectInitial, customBody: customBodyInitial },
        validationSchema: emailTemplateValidationSchema,
        onSubmit: (formValues) => {
            onChange(formValues);
            onCloseHandler();
        },
    });
    const { errors, isSubmitting, resetForm, setValues, submitForm, values } = formik;
    const { customBody, customSubject } = values;

    useEffect(() => {
        if (isOpen) {
            // eslint-disable-next-line @typescript-eslint/no-floating-promises
            setValues({ customSubject: customSubjectInitial, customBody: customBodyInitial });
        }
    }, [customSubjectInitial, customBodyInitial, setValues, isOpen]);

    const isApplyDisabled =
        isSubmitting ||
        !isEmpty(errors) ||
        (customSubject === customSubjectInitial && customBody === customBodyInitial);
    const isPreviewDisabled = isSubmitting || !isEmpty(errors);

    function onCloseHandler() {
        resetForm();
        onClose();
    }

    const emailTemplateForm = (
        <EmailTemplateForm
            customBodyDefault={customBodyDefault}
            customSubjectDefault={customSubjectDefault}
            formik={formik}
        />
    );

    return (
        <Modal
            variant="medium"
            title="Edit email template"
            description="Customize the email subject and body as needed, or leave it empty to use the default template."
            isOpen={isOpen}
            onClose={onCloseHandler}
            actions={[
                <Button
                    key="confirm"
                    variant="primary"
                    onClick={submitForm}
                    isDisabled={isApplyDisabled}
                    isLoading={isSubmitting}
                >
                    Apply
                </Button>,
                <Button key="cancel" variant="plain" isInline onClick={onCloseHandler}>
                    Cancel
                </Button>,
            ]}
        >
            {renderTemplatePreview ? (
                <Tabs defaultActiveKey={0} role="region">
                    <Tab eventKey={0} title={<TabTitleText>Edit</TabTitleText>}>
                        {emailTemplateForm}
                    </Tab>
                    <Tab
                        eventKey={1}
                        title={<TabTitleText>Preview</TabTitleText>}
                        isDisabled={isPreviewDisabled}
                    >
                        {renderTemplatePreview({
                            customBody,
                            customSubject,
                            customSubjectDefault,
                        })}
                    </Tab>
                </Tabs>
            ) : (
                emailTemplateForm
            )}
        </Modal>
    );
}

export default EmailTemplateModal;
