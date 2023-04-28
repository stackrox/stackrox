/* eslint-disable no-void */
import React, { ReactElement, useState } from 'react';
import {
    Alert,
    AlertVariant,
    Checkbox,
    Form,
    PageSection,
    SelectOption,
    TextInput,
    Popover,
} from '@patternfly/react-core';
import { HelpIcon } from '@patternfly/react-icons';
import * as yup from 'yup';

import { NotifierIntegrationBase } from 'services/NotifierIntegrationsService';

import SelectSingle from 'Components/SelectSingle';
import usePageState from 'Containers/Integrations/hooks/usePageState';
import FormMessage from 'Components/PatternFly/FormMessage';
import FormCancelButton from 'Components/PatternFly/FormCancelButton';
import FormTestButton from 'Components/PatternFly/FormTestButton';
import FormSaveButton from 'Components/PatternFly/FormSaveButton';
import useIntegrationForm from '../useIntegrationForm';
import { IntegrationFormProps } from '../integrationFormTypes';

import IntegrationFormActions from '../IntegrationFormActions';
import FormLabelGroup from '../FormLabelGroup';
import AnnotationKeyLabelIcon from '../AnnotationKeyLabelIcon';

export type EmailIntegration = {
    email: {
        server: string;
        username: string;
        password: string;
        from: string;
        sender: string;
        disableTLS: boolean;
        startTLSAuthMethod: 'DISABLED' | 'PLAIN' | 'LOGIN';
        allowUnauthenticatedSmtp: boolean;
    };
    type: 'email';
} & NotifierIntegrationBase;

export type EmailIntegrationFormValues = {
    notifier: EmailIntegration;
    updatePassword: boolean;
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

const validHostnameRegex =
    /^(([a-z0-9]|[a-z0-9][a-z0-9-]*[a-z0-9])\.)*([a-z0-9]|[a-z0-9][a-z0-9-]*[a-z0-9])(:[0-9]+)?$/;

export const validationSchema = yup.object().shape({
    notifier: yup.object().shape({
        name: yup.string().trim().required('Email integration name is required'),
        labelDefault: yup
            .string()
            .trim()
            .required('A default recipient email address is required')
            .email('Must be a valid default recipient email address'),
        labelKey: yup.string(),
        email: yup.object().shape({
            server: yup
                .string()
                .trim()
                .required('A server address is required')
                .matches(validHostnameRegex, 'Must be a valid server address'),
            allowUnauthenticatedSmtp: yup.boolean(),
            username: yup.string().when('allowUnauthenticatedSmtp', {
                is: false,
                then: (usernameSchema) => usernameSchema.trim().required('A username is required'),
            }),
            password: yup
                .string()
                .test(
                    'password-test',
                    'A password is required',
                    (value, context: yup.TestContext) => {
                        const requirePasswordField =
                            context?.from?.[2].value?.updatePassword || false;

                        if (!requirePasswordField || context.parent.allowUnauthenticatedSmtp) {
                            return true;
                        }

                        const trimmedValue = value?.trim();
                        return !!trimmedValue;
                    }
                ),
            from: yup.string(),
            sender: yup
                .string()
                .trim()
                .required('A sender email address is required')
                .email('Must be a valid sender email address'),
            startTLSAuthMethod: yup.string().when('disableTLS', {
                is: true,
                then: (startTLSAuthMethodSchema) => startTLSAuthMethodSchema.required(),
            }),
        }),
    }),
    updatePassword: yup.bool(),
});

export const defaultValues: EmailIntegrationFormValues = {
    notifier: {
        id: '',
        name: '',
        email: {
            server: '',
            username: '',
            password: '',
            from: '',
            sender: '',
            disableTLS: false,
            startTLSAuthMethod: 'DISABLED',
            allowUnauthenticatedSmtp: false,
        },
        labelDefault: '',
        labelKey: '',
        uiEndpoint: window.location.origin,
        type: 'email',
    },
    updatePassword: true,
};

function EmailIntegrationForm({
    initialValues = null,
    isEditable = false,
}: IntegrationFormProps<EmailIntegration>): ReactElement {
    const formInitialValues = { ...defaultValues, ...initialValues };
    const [storedUsername, setStoredUsername] = useState('');
    if (initialValues) {
        formInitialValues.notifier = {
            ...formInitialValues.notifier,
            ...initialValues,
        };
        // We want to clear the password because backend returns '******' to represent that there
        // are currently stored credentials
        formInitialValues.notifier.email.password = '';
        // Don't assume user wants to change password; that has caused confusing UX.
        formInitialValues.updatePassword = false;
    }
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
    } = useIntegrationForm<EmailIntegrationFormValues>({
        initialValues: formInitialValues,
        validationSchema,
    });
    const { isCreating } = usePageState();
    const { allowUnauthenticatedSmtp } = values.notifier.email;

    function onChange(value, event) {
        return setFieldValue(event.target.id, value);
    }

    function updateStartTLSAuthMethodOnChange(value, event) {
        void setFieldValue(event.target.id, value);
        if (value === false && values.notifier.email.startTLSAuthMethod !== 'DISABLED') {
            void setFieldValue('notifier.email.startTLSAuthMethod', 'DISABLED');
        }
    }

    function onUpdateUnauthenticatedChange(isChecked) {
        if (isChecked) {
            setStoredUsername(values.notifier.email.username);
            setFieldValue('notifier.email.username', '');
            setFieldValue('notifier.email.password', '');
        } else {
            setFieldValue('notifier.email.username', storedUsername);
        }
        setFieldValue('notifier.email.allowUnauthenticatedSmtp', isChecked);
    }

    function onUpdateCredentialsChange(value, event) {
        setFieldValue('notifier.email.password', '');
        return setFieldValue(event.target.id, value);
    }

    return (
        <>
            <PageSection variant="light" isFilled hasOverflowScroll>
                <FormMessage message={message} />
                <Form isWidthLimited>
                    <FormLabelGroup
                        label="Integration name"
                        isRequired
                        fieldId="notifier.name"
                        touched={touched}
                        errors={errors}
                    >
                        <TextInput
                            isRequired
                            type="text"
                            id="notifier.name"
                            value={values.notifier.name}
                            placeholder="(example, Email Integration)"
                            onChange={onChange}
                            onBlur={handleBlur}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        label="Email server"
                        isRequired
                        fieldId="notifier.email.server"
                        touched={touched}
                        errors={errors}
                    >
                        <TextInput
                            isRequired
                            type="text"
                            id="notifier.email.server"
                            value={values.notifier.email.server}
                            placeholder="example, smtp.example.com:465"
                            onChange={onChange}
                            onBlur={handleBlur}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        label=""
                        fieldId="notifier.email.unauthenticated"
                        errors={errors}
                    >
                        <>
                            <div className="pf-u-display-flex pf-u-align-items-flex-start">
                                <Checkbox
                                    label="Enable unauthenticated SMTP"
                                    id="notifier.email.unauthenticated"
                                    isChecked={allowUnauthenticatedSmtp}
                                    onChange={onUpdateUnauthenticatedChange}
                                    onBlur={handleBlur}
                                />
                                <Popover
                                    showClose={false}
                                    bodyContent="Enable unauthenticated SMTP will allow you to setup an email notifier if you don’t have authenticated email services."
                                >
                                    <button
                                        type="button"
                                        aria-label="More info on unauthenticated SMTP field"
                                        onClick={(e) => e.preventDefault()}
                                        className="pf-c-form__group-label-help"
                                    >
                                        <HelpIcon />
                                    </button>
                                </Popover>
                            </div>
                            {allowUnauthenticatedSmtp && (
                                <Alert
                                    className="pf-u-mt-md"
                                    title="Security Warning"
                                    variant={AlertVariant.warning}
                                    isInline
                                >
                                    <p>
                                        Unauthenticated SMTP is an insecure configuration and not
                                        generally recommended. Please proceed with caution when
                                        enabling this setting.
                                    </p>
                                </Alert>
                            )}
                        </>
                    </FormLabelGroup>
                    <FormLabelGroup
                        label="Username"
                        isRequired={!allowUnauthenticatedSmtp}
                        fieldId="notifier.email.username"
                        touched={touched}
                        errors={errors}
                    >
                        <TextInput
                            isRequired={!allowUnauthenticatedSmtp}
                            type="text"
                            id="notifier.email.username"
                            value={values.notifier.email.username}
                            placeholder={
                                allowUnauthenticatedSmtp ? '' : 'example, postmaster@example.com'
                            }
                            onChange={onChange}
                            onBlur={handleBlur}
                            isDisabled={!isEditable || allowUnauthenticatedSmtp}
                        />
                    </FormLabelGroup>
                    {!isCreating && isEditable && !allowUnauthenticatedSmtp && (
                        <FormLabelGroup
                            label=""
                            fieldId="updatePassword"
                            helperText="Enable this option to replace currently stored credentials (if any)"
                            errors={errors}
                        >
                            <Checkbox
                                label="Update token"
                                id="updatePassword"
                                isChecked={values.updatePassword}
                                onChange={onUpdateCredentialsChange}
                                onBlur={handleBlur}
                                isDisabled={!isEditable}
                            />
                        </FormLabelGroup>
                    )}
                    <FormLabelGroup
                        label="Password"
                        isRequired={values.updatePassword && !allowUnauthenticatedSmtp}
                        fieldId="notifier.email.password"
                        touched={touched}
                        errors={errors}
                    >
                        <TextInput
                            isRequired={values.updatePassword && !allowUnauthenticatedSmtp}
                            type="password"
                            id="notifier.email.password"
                            value={values.notifier.email.password}
                            onChange={onChange}
                            onBlur={handleBlur}
                            isDisabled={
                                !isEditable || !values.updatePassword || allowUnauthenticatedSmtp
                            }
                            placeholder={
                                values.updatePassword || allowUnauthenticatedSmtp
                                    ? ''
                                    : 'Currently-stored password will be used.'
                            }
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        label="From"
                        fieldId="notifier.email.from"
                        touched={touched}
                        errors={errors}
                        helperText={
                            <span className="pf-u-font-size-sm">
                                Specifies the email FROM header
                            </span>
                        }
                    >
                        <TextInput
                            type="text"
                            id="notifier.email.from"
                            value={values.notifier.email.from}
                            placeholder="example, Security Alerts"
                            onChange={onChange}
                            onBlur={handleBlur}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        isRequired
                        label="Sender"
                        fieldId="notifier.email.sender"
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
                            id="notifier.email.sender"
                            value={values.notifier.email.sender}
                            placeholder="example, security-alerts@example.com"
                            onChange={onChange}
                            onBlur={handleBlur}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        isRequired
                        label="Default recipient"
                        fieldId="notifier.labelDefault"
                        touched={touched}
                        errors={errors}
                    >
                        <TextInput
                            isRequired
                            type="text"
                            id="notifier.labelDefault"
                            value={values.notifier.labelDefault}
                            placeholder="example, security-alerts-recipients@example.com"
                            onChange={onChange}
                            onBlur={handleBlur}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        label="Annotation key for recipient"
                        labelIcon={<AnnotationKeyLabelIcon />}
                        fieldId="notifier.labelKey"
                        touched={touched}
                        errors={errors}
                    >
                        <TextInput
                            type="text"
                            id="notifier.labelKey"
                            value={values.notifier.labelKey}
                            onChange={onChange}
                            onBlur={handleBlur}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup label="" fieldId="notifier.email.disableTLS" errors={errors}>
                        <Checkbox
                            label="Disable TLS certificate validation (insecure)"
                            id="notifier.email.disableTLS"
                            isChecked={values.notifier.email.disableTLS}
                            onChange={updateStartTLSAuthMethodOnChange}
                            onBlur={handleBlur}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        label="Use STARTTLS (requires TLS to be disabled)"
                        fieldId="notifier.email.startTLSAuthMethod"
                        errors={errors}
                    >
                        <SelectSingle
                            id="notifier.email.startTLSAuthMethod"
                            value={values.notifier.email.startTLSAuthMethod as string}
                            handleSelect={setFieldValue}
                            isDisabled={!isEditable || !values.notifier.email.disableTLS}
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
                        isDisabled={!isValid}
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
