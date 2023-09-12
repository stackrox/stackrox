import React, { useEffect, useState } from 'react';
import {
    Button,
    Card,
    CardBody,
    CardFooter,
    CardTitle,
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
    ToggleGroup,
    ToggleGroupItem,
} from '@patternfly/react-core';
import { FormikErrors, FormikTouched, useFormik } from 'formik';
import get from 'lodash/get';
import isEmpty from 'lodash/isEmpty';

import { maxEmailBodyLength, maxEmailSubjectLength } from './useReportFormValues';
import {
    EmailTemplateFormData,
    defaultEmailBodyWithNoCVEsFound,
    emailTemplateValidationSchema,
} from './emailTemplateFormUtils';

export type EmailTemplateFormModalProps = {
    isOpen: boolean;
    onClose: () => void;
    onChange: (formData: EmailTemplateFormData) => void;
    initialEmailSubject: string;
    initialEmailBody: string;
    defaultEmailSubject: string;
    defaultEmailBody: string;
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
    const [selectedPreviewText, setSelectedPreviewText] = useState<string>('CVEs found');

    useEffect(() => {
        if (isOpen) {
            // eslint-disable-next-line @typescript-eslint/no-floating-promises
            setValues({ emailSubject: initialEmailSubject, emailBody: initialEmailBody });
        }
    }, [initialEmailSubject, initialEmailBody, setValues, isOpen]);

    const isApplyDisabled = isSubmitting || !isEmpty(errors);
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
                <Button key="cancel" variant="secondary" onClick={onCloseHandler}>
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
                                    >
                                        Clear to default
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
                                    >
                                        Clear to default
                                    </Button>
                                </FlexItem>
                            </Flex>
                        </FormGroup>
                    </Form>
                </Tab>
                <Tab
                    eventKey={1}
                    title={<TabTitleText>Preview</TabTitleText>}
                    isDisabled={isApplyDisabled}
                >
                    <Flex
                        className="pf-u-py-lg"
                        spaceItems={{ default: 'spaceItemsMd' }}
                        direction={{ default: 'column' }}
                    >
                        <FlexItem>
                            <TextContent>
                                <Text component={TextVariants.small}>
                                    This preview displays modifications to the email subject and
                                    body only. Data shown in the report parameters are sample data
                                    meant solely for illustration. For any actual data, please check
                                    the email attachment in the real report. Please not that an
                                    attachment of the report data will not be provided if no CVEs
                                    are found.
                                </Text>
                            </TextContent>
                        </FlexItem>
                        <FlexItem>
                            <ToggleGroup aria-label="Preview with or without CVEs found">
                                <ToggleGroupItem
                                    text="CVEs found"
                                    isSelected={selectedPreviewText === 'CVEs found'}
                                    onChange={() => setSelectedPreviewText('CVEs found')}
                                />
                                <ToggleGroupItem
                                    text="CVEs not found"
                                    isSelected={selectedPreviewText === 'CVEs not found'}
                                    onChange={() => setSelectedPreviewText('CVEs not found')}
                                />
                            </ToggleGroup>
                        </FlexItem>
                        <FlexItem>
                            <Card isFlat>
                                <CardTitle>{values.emailSubject || defaultEmailSubject}</CardTitle>
                                <CardBody>
                                    {values.emailBody ||
                                        (selectedPreviewText === 'CVEs found'
                                            ? defaultEmailBody
                                            : defaultEmailBodyWithNoCVEsFound)}
                                </CardBody>
                                <CardFooter>
                                    {/* 
                                        NOTE: When using this in plain HTML, replace the style
                                        object with a style string like this: style="padding: 0 0 10px 0;"
                                    */}
                                    <div>
                                        <div style={{ padding: '0 0 10px 0' }}>
                                            <span
                                                style={{ fontWeight: 'bold', marginRight: '10px' }}
                                            >
                                                Number of CVEs found:
                                            </span>
                                            <span>
                                                {selectedPreviewText === 'CVEs found'
                                                    ? '50 in Deployed images; 30 in Watched images'
                                                    : '0 in Deployed images; 0 in Watched images'}
                                            </span>
                                        </div>
                                        <div style={{ padding: '0 0 10px 0' }}>
                                            <span
                                                style={{ fontWeight: 'bold', marginRight: '10px' }}
                                            >
                                                CVE severity:
                                            </span>
                                            <span>Critical, Important, Moderate, Low</span>
                                        </div>
                                        <div style={{ padding: '0 0 10px 0' }}>
                                            <span
                                                style={{ fontWeight: 'bold', marginRight: '10px' }}
                                            >
                                                CVE status:
                                            </span>
                                            <span>Fixable, Not fixable</span>
                                        </div>
                                        <div style={{ padding: '0 0 10px 0' }}>
                                            <span
                                                style={{ fontWeight: 'bold', marginRight: '10px' }}
                                            >
                                                Report scope:
                                            </span>
                                            <span>Collection 1</span>
                                        </div>
                                        <div style={{ padding: '0 0 10px 0' }}>
                                            <span
                                                style={{ fontWeight: 'bold', marginRight: '10px' }}
                                            >
                                                Image type:
                                            </span>
                                            <span>Deployed images, Watched images</span>
                                        </div>
                                        <div style={{ padding: '0 0 10px 0' }}>
                                            <span
                                                style={{ fontWeight: 'bold', marginRight: '10px' }}
                                            >
                                                CVEs discovered since:
                                            </span>
                                            <span>All time</span>
                                        </div>
                                    </div>
                                </CardFooter>
                            </Card>
                        </FlexItem>
                    </Flex>
                </Tab>
            </Tabs>
        </Modal>
    );
}

export default EmailTemplateFormModal;
