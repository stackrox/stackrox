import React, { useEffect } from 'react';
import {
    Button,
    Flex,
    FlexItem,
    Form,
    FormGroup,
    Modal,
    Tab,
    TabTitleText,
    Tabs,
    Text,
    TextArea,
    TextContent,
    TextInput,
    TextVariants,
} from '@patternfly/react-core';
import { FormikErrors, FormikTouched, useFormik } from 'formik';
import get from 'lodash/get';
import isEmpty from 'lodash/isEmpty';

import {
    ReportParametersFormValues,
    maxEmailBodyLength,
    maxEmailSubjectLength,
} from './useReportFormValues';
import { EmailTemplateFormData, emailTemplateValidationSchema } from './emailTemplateFormUtils';
import EmailTemplatePreview from '../components/EmailTemplatePreview';

export type EmailTemplateFormModalProps = {
    isOpen: boolean;
    onClose: () => void;
    onChange: (formData: EmailTemplateFormData) => void;
    initialEmailSubject: string;
    initialEmailBody: string;
    defaultEmailSubject: string;
    defaultEmailBody: string;
    reportParameters: ReportParametersFormValues;
};

function getFieldValidated(
    errors: FormikErrors<EmailTemplateFormData>,
    touched: FormikTouched<EmailTemplateFormData>,
    fieldId: string
) {
    const isFieldInvalid = !!(get(errors, fieldId) && get(touched, fieldId));
    const fieldValidated = isFieldInvalid ? 'error' : 'default';
    return fieldValidated;
}

function EmailTemplateFormModal({
    isOpen,
    onClose,
    onChange,
    initialEmailSubject,
    initialEmailBody,
    defaultEmailSubject,
    defaultEmailBody,
    reportParameters,
}: EmailTemplateFormModalProps) {
    const {
        values,
        errors,
        touched,
        handleChange,
        handleBlur,
        handleSubmit,
        submitForm,
        isSubmitting,
        resetForm,
        setValues,
        setFieldValue,
    } = useFormik<EmailTemplateFormData>({
        initialValues: { emailSubject: initialEmailSubject, emailBody: initialEmailBody },
        validationSchema: emailTemplateValidationSchema,
        onSubmit: (formValues) => {
            onChange(formValues);
            onCloseHandler();
        },
    });

    useEffect(() => {
        if (isOpen) {
            // eslint-disable-next-line @typescript-eslint/no-floating-promises
            setValues({ emailSubject: initialEmailSubject, emailBody: initialEmailBody });
        }
    }, [initialEmailSubject, initialEmailBody, setValues, isOpen]);

    const isApplyDisabled =
        isSubmitting ||
        !isEmpty(errors) ||
        (values.emailSubject === initialEmailSubject && values.emailBody === initialEmailBody);
    const isPreviewDisabled = isSubmitting || !isEmpty(errors);
    const emailSubjectValidated = getFieldValidated(errors, touched, 'emailSubject');
    const emailBodyValidated = getFieldValidated(errors, touched, 'emailBody');

    function onCloseHandler() {
        resetForm();
        onClose();
    }

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
            <Tabs defaultActiveKey={0} role="region">
                <Tab eventKey={0} title={<TabTitleText>Edit</TabTitleText>}>
                    <Form className="pf-u-py-lg pf-u-px-lg" onSubmit={handleSubmit}>
                        <FormGroup
                            label="Email subject"
                            fieldId="emailSubject"
                            validated={emailSubjectValidated}
                            helperTextInvalid={errors.emailSubject}
                        >
                            <TextInput
                                id="emailSubject"
                                type="text"
                                value={values.emailSubject}
                                validated={emailSubjectValidated}
                                onChange={(_, e) => handleChange(e)}
                                onBlur={handleBlur}
                                isDisabled={isSubmitting}
                                placeholder={defaultEmailSubject}
                            />
                            <Flex>
                                <FlexItem flex={{ default: 'flex_1' }}>
                                    <TextContent>
                                        <Text
                                            component={TextVariants.p}
                                            className="pf-u-font-size-sm"
                                        >
                                            {values.emailSubject.length} / {maxEmailSubjectLength}{' '}
                                            characters
                                        </Text>
                                    </TextContent>
                                </FlexItem>
                                <FlexItem>
                                    <Button
                                        className="pf-u-mt-sm"
                                        variant="link"
                                        isInline
                                        isSmall
                                        onClick={() => setFieldValue('emailSubject', '')}
                                        isDisabled={values.emailSubject.length === 0}
                                    >
                                        Reset to default
                                    </Button>
                                </FlexItem>
                            </Flex>
                        </FormGroup>
                        <FormGroup
                            label="Email body"
                            fieldId="emailBody"
                            validated={emailBodyValidated}
                            helperTextInvalid={errors.emailBody}
                        >
                            <TextArea
                                id="emailBody"
                                type="text"
                                value={values.emailBody}
                                validated={emailBodyValidated}
                                onChange={(_, e) => handleChange(e)}
                                onBlur={handleBlur}
                                isDisabled={isSubmitting}
                                style={{ minHeight: '250px' }}
                                placeholder={defaultEmailBody}
                            />
                            <Flex>
                                <FlexItem flex={{ default: 'flex_1' }}>
                                    <TextContent>
                                        <Text
                                            component={TextVariants.p}
                                            className="pf-u-font-size-sm"
                                        >
                                            {values.emailBody.length} / {maxEmailBodyLength}{' '}
                                            characters
                                        </Text>
                                    </TextContent>
                                </FlexItem>
                                <FlexItem>
                                    <Button
                                        className="pf-u-mt-sm"
                                        variant="link"
                                        isInline
                                        isSmall
                                        onClick={() => setFieldValue('emailBody', '')}
                                        isDisabled={values.emailBody.length === 0}
                                    >
                                        Reset to default
                                    </Button>
                                </FlexItem>
                            </Flex>
                        </FormGroup>
                    </Form>
                </Tab>
                <Tab
                    eventKey={1}
                    title={<TabTitleText>Preview</TabTitleText>}
                    isDisabled={isPreviewDisabled}
                >
                    <EmailTemplatePreview
                        emailSubject={values.emailSubject}
                        emailBody={values.emailBody}
                        defaultEmailSubject={defaultEmailSubject}
                        reportParameters={reportParameters}
                    />
                </Tab>
            </Tabs>
        </Modal>
    );
}

export default EmailTemplateFormModal;
