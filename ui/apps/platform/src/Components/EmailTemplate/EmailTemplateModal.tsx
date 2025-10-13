import React from 'react';
import type { ReactElement } from 'react';
import { Button, Modal, Tab, TabTitleText, Tabs } from '@patternfly/react-core';
import { useFormik } from 'formik';
import isEmpty from 'lodash/isEmpty';

import { emailTemplateValidationSchema } from './EmailTemplate.utils';
import type { EmailTemplateFormData } from './EmailTemplate.utils';
import EmailTemplateForm from './EmailTemplateForm';

export type TemplatePreviewArgs = {
    customBody: string;
    customSubject: string;
    customSubjectDefault: string;
};

export type EmailTemplateModalProps = {
    customBodyDefault: string;
    customBodyInitial: string;
    customSubjectDefault: string;
    customSubjectInitial: string;
    onChange: ((formData: EmailTemplateFormData) => void) | null;
    onClose: () => void;
    renderTemplatePreview?: (args: TemplatePreviewArgs) => ReactElement;
    title: string;
};

function EmailTemplateModal({
    customBodyDefault,
    customBodyInitial,
    customSubjectDefault,
    customSubjectInitial,
    onChange,
    onClose,
    renderTemplatePreview,
    title,
}: EmailTemplateModalProps) {
    const formik = useFormik<EmailTemplateFormData>({
        initialValues: { customSubject: customSubjectInitial, customBody: customBodyInitial },
        validationSchema: emailTemplateValidationSchema,
        onSubmit: (formValues) => {
            if (onChange) {
                onChange(formValues);
            }
            onClose();
        },
    });
    const { errors, isSubmitting, submitForm, values } = formik;
    const { customBody, customSubject } = values;

    const isApplyDisabled =
        isSubmitting ||
        !isEmpty(errors) ||
        (customSubject === customSubjectInitial && customBody === customBodyInitial);
    const isPreviewDisabled = isSubmitting || !isEmpty(errors);

    const emailTemplateForm = (
        <EmailTemplateForm
            customBodyDefault={customBodyDefault}
            customSubjectDefault={customSubjectDefault}
            formik={formik}
            isReadOnly={!onChange}
        />
    );

    const actions = onChange
        ? [
              <Button
                  key="confirm"
                  variant="primary"
                  onClick={submitForm}
                  isDisabled={isApplyDisabled}
                  isLoading={isSubmitting}
              >
                  Apply
              </Button>,
              <Button key="cancel" variant="link" isInline onClick={onClose}>
                  Cancel
              </Button>,
          ]
        : [
              <Button key="cancel" variant="secondary" isInline onClick={onClose}>
                  Close
              </Button>,
          ];

    return (
        <Modal
            variant="medium"
            title={title}
            description="Customize the email subject and body as needed, or leave it empty to use the default template."
            isOpen
            onClose={onClose}
            actions={actions}
        >
            {renderTemplatePreview ? (
                <Tabs defaultActiveKey={0}>
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
