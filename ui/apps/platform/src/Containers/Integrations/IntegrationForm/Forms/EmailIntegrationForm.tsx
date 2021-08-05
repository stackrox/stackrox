import React, { ReactElement } from 'react';
import { Checkbox, Form, PageSection, SelectOption, TextInput } from '@patternfly/react-core';
import * as yup from 'yup';

import SelectSingle from 'Components/SelectSingle';
import useIntegrationForm from '../useIntegrationForm';
import { IntegrationFormProps } from '../integrationFormTypes';

import IntegrationFormActions from '../IntegrationFormActions';
import FormCancelButton from '../FormCancelButton';
import FormTestButton from '../FormTestButton';
import FormSaveButton from '../FormSaveButton';
import FormMessage from '../FormMessage';
import FormLabelGroup from '../FormLabelGroup';
import AnnotationKeyLabelIcon from '../AnnotationKeyLabelIcon';

export type EmailIntegration = {
    id?: string;
    name: string;
    email: {
        server: string;
        username: string;
        password: string;
        from: string;
        sender: string;
        disableTLS: boolean;
        startTLSAuthMethod: 'DISABLED' | 'PLAIN' | 'LOGIN';
    };
    labelDefault: string;
    labelKey: string;
    uiEndpoint: string;
    type: 'email';
    enabled: boolean;
};

const startTLSAuthMethods = [
    {
        label: 'Disabled',
        value: 'DISABLED',
    },
    {
        label: 'Plain',
        value: 'PLAIN',
    },
    {
        label: 'Login',
        value: 'LOGIN',
    },
];

const validHostnameRegex = /^(([a-z0-9]|[a-z0-9][a-z0-9-]*[a-z0-9])\.)*([a-z0-9]|[a-z0-9][a-z0-9-]*[a-z0-9])(:[0-9]+)?$/;

export const validationSchema = yup.object().shape({
    name: yup.string().trim().required('Required'),
    labelDefault: yup.string().trim().email().required('A valid default recipient email address'),
    labelKey: yup.string(),
    email: yup.object().shape({
        server: yup
            .string()
            .trim()
            .required('A email server address is required')
            .matches(validHostnameRegex, 'Must be a valid server address'),
        username: yup.string().trim().required('A username is required'),
        password: yup.string().trim().required('A password is required'),
        from: yup.string(),
        sender: yup.string().trim().email().required('A valid sender email address is required'),
        startTLSAuthMethod: yup.string(),
    }),
});

export const defaultValues: EmailIntegration = {
    name: '',
    email: {
        server: '',
        username: '',
        password: '',
        from: '',
        sender: '',
        disableTLS: false,
        startTLSAuthMethod: 'DISABLED',
    },
    labelDefault: '',
    labelKey: '',
    uiEndpoint: window.location.origin,
    type: 'email',
    enabled: true,
};

function EmailIntegrationForm({
    initialValues = null,
    isEditable = false,
}: IntegrationFormProps<EmailIntegration>): ReactElement {
    const formInitialValues = initialValues
        ? { ...defaultValues, ...initialValues }
        : defaultValues;
    const {
        values,
        touched,
        errors,
        dirty,
        isValid,
        setFieldValue,
        handleBlur,
        isSubmitting,
        isTesting,
        onSave,
        onTest,
        onCancel,
        message,
    } = useIntegrationForm<EmailIntegration, typeof validationSchema>({
        initialValues: formInitialValues,
        validationSchema,
    });

    function onChange(value, event) {
        return setFieldValue(event.target.id, value);
    }

    return (
        <>
            <PageSection variant="light" isFilled hasOverflowScroll>
                {message && <FormMessage message={message} />}
                <Form isWidthLimited>
                    <FormLabelGroup
                        label="Integration name"
                        isRequired
                        fieldId="name"
                        touched={touched}
                        errors={errors}
                    >
                        <TextInput
                            isRequired
                            type="text"
                            id="name"
                            value={values.name}
                            placeholder="(example, Email Integration)"
                            onChange={onChange}
                            onBlur={handleBlur}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        label="Email server"
                        isRequired
                        fieldId="email.server"
                        touched={touched}
                        errors={errors}
                    >
                        <TextInput
                            isRequired
                            type="text"
                            id="email.server"
                            value={values.email.server}
                            placeholder="example, smtp.example.com:465"
                            onChange={onChange}
                            onBlur={handleBlur}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        label="Username"
                        isRequired
                        fieldId="email.username"
                        touched={touched}
                        errors={errors}
                    >
                        <TextInput
                            isRequired
                            type="text"
                            id="email.username"
                            value={values.email.username}
                            placeholder="example, postmaster@example.com"
                            onChange={onChange}
                            onBlur={handleBlur}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        label="Password"
                        isRequired
                        fieldId="email.password"
                        touched={touched}
                        errors={errors}
                    >
                        <TextInput
                            isRequired
                            type="password"
                            id="email.password"
                            value={values.email.password}
                            onChange={onChange}
                            onBlur={handleBlur}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        label="From"
                        fieldId="email.from"
                        touched={touched}
                        errors={errors}
                        helperText={
                            <span className="pf-u-font-size-sm">
                                (optional) Specifies the email FROM header
                            </span>
                        }
                    >
                        <TextInput
                            type="text"
                            id="email.from"
                            value={values.email.from}
                            placeholder="example, Advanced Cluster Management"
                            onChange={onChange}
                            onBlur={handleBlur}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        isRequired
                        label="Sender"
                        fieldId="email.sender"
                        touched={touched}
                        errors={errors}
                        helperText={
                            <span className="pf-u-font-size-sm">
                                Specifies the email SENDER header
                            </span>
                        }
                    >
                        <TextInput
                            isRequired
                            type="text"
                            id="email.sender"
                            value={values.email.sender}
                            placeholder="example, acs-notifier@example.com"
                            onChange={onChange}
                            onBlur={handleBlur}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        isRequired
                        label="Default recipient"
                        fieldId="labelDefault"
                        touched={touched}
                        errors={errors}
                    >
                        <TextInput
                            isRequired
                            type="text"
                            id="labelDefault"
                            value={values.labelDefault}
                            placeholder="example, acs-alerts@example.com"
                            onChange={onChange}
                            onBlur={handleBlur}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        label="Annotation key for recipient"
                        labelIcon={<AnnotationKeyLabelIcon />}
                        fieldId="labelKey"
                        touched={touched}
                        errors={errors}
                    >
                        <TextInput
                            type="text"
                            id="labelKey"
                            value={values.labelKey}
                            onChange={onChange}
                            onBlur={handleBlur}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup label="" fieldId="email.disableTLS" errors={errors}>
                        <Checkbox
                            label="Disable TLS Certificate Validation (Insecure)"
                            id="email.disableTLS"
                            isChecked={values.email.disableTLS}
                            onChange={onChange}
                            onBlur={handleBlur}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        label="Use STARTTLS (requires TLS to be disabled)"
                        fieldId="email.startTLSAuthMethod"
                        isRequired
                        errors={errors}
                    >
                        <SelectSingle
                            id="email.startTLSAuthMethod"
                            value={values.email.startTLSAuthMethod as string}
                            handleSelect={setFieldValue}
                            isDisabled={!isEditable}
                            direction="up"
                        >
                            {startTLSAuthMethods.map(({ value, label }) => (
                                <SelectOption key={value} value={value}>
                                    {label}
                                </SelectOption>
                            ))}
                        </SelectSingle>
                    </FormLabelGroup>
                </Form>
            </PageSection>
            {isEditable && (
                <IntegrationFormActions>
                    <FormSaveButton
                        onSave={onSave}
                        isSubmitting={isSubmitting}
                        isTesting={isTesting}
                        isDisabled={!dirty || !isValid}
                    >
                        Save
                    </FormSaveButton>
                    <FormTestButton
                        onTest={onTest}
                        isSubmitting={isSubmitting}
                        isTesting={isTesting}
                    >
                        Test
                    </FormTestButton>
                    <FormCancelButton onCancel={onCancel}>Cancel</FormCancelButton>
                </IntegrationFormActions>
            )}
        </>
    );
}

export default EmailIntegrationForm;
