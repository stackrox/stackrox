/* eslint-disable @typescript-eslint/no-explicit-any */
import React, { ReactElement, useState } from 'react';
import { Alert, Checkbox, Form, PageSection, Popover, TextInput } from '@patternfly/react-core';
import { SelectOption } from '@patternfly/react-core/deprecated';
import { HelpIcon } from '@patternfly/react-icons';
import { FormikErrors, FormikTouched } from 'formik';

import SelectSingle from 'Components/SelectSingle';
import AnnotationKeyLabelIcon from 'Containers/Integrations/IntegrationForm//AnnotationKeyLabelIcon';
import FormLabelGroup from 'Containers/Integrations/IntegrationForm/FormLabelGroup';
import { EmailIntegrationFormValues } from 'Containers/Integrations/IntegrationForm/Forms/EmailIntegrationForm';

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

export type EmailNotifierFormProps = {
    values: EmailIntegrationFormValues;
    setFieldValue: (field: string, value: any, shouldValidate?: boolean) => void;
    handleBlur: (e: React.FocusEvent<any, Element>) => void;
    errors: FormikErrors<any>;
    touched: FormikTouched<any>;
};

function EmailNotifierForm({
    values,
    setFieldValue,
    handleBlur,
    errors,
    touched,
}: EmailNotifierFormProps): ReactElement {
    const [storedUsername, setStoredUsername] = useState('');
    const { allowUnauthenticatedSmtp } = values.notifier.email;
    function onChange(value, event) {
        setFieldValue(event.target.id, value);
    }

    function updateStartTLSAuthMethodOnChange(value, event) {
        setFieldValue(event.target.id, value);
        if (value === false && values.notifier.email.startTLSAuthMethod !== 'DISABLED') {
            setFieldValue('notifier.email.startTLSAuthMethod', 'DISABLED');
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

    return (
        <>
            <PageSection variant="light" isFilled hasOverflowScroll>
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
                            onChange={(event, value) => onChange(value, event)}
                            onBlur={handleBlur}
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
                            onChange={(event, value) => onChange(value, event)}
                            onBlur={handleBlur}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        label=""
                        fieldId="notifier.email.unauthenticated"
                        errors={errors}
                    >
                        <>
                            <div className="pf-v5-u-display-flex pf-v5-u-align-items-flex-start">
                                <Checkbox
                                    label="Enable unauthenticated SMTP"
                                    id="notifier.email.unauthenticated"
                                    isChecked={allowUnauthenticatedSmtp}
                                    onChange={(_event, isChecked) =>
                                        onUpdateUnauthenticatedChange(isChecked)
                                    }
                                    onBlur={handleBlur}
                                />
                                <Popover
                                    showClose={false}
                                    aria-label="Information about unauthenticated SMTP"
                                    bodyContent="Enable unauthenticated SMTP will allow you to setup an email notifier if you don’t have authenticated email services."
                                >
                                    <button
                                        type="button"
                                        aria-label="More info on unauthenticated SMTP field"
                                        onClick={(e) => e.preventDefault()}
                                        className="pf-v5-c-form__group-label-help"
                                    >
                                        <HelpIcon />
                                    </button>
                                </Popover>
                            </div>
                            {allowUnauthenticatedSmtp && (
                                <Alert
                                    className="pf-v5-u-mt-md"
                                    title="Security Warning"
                                    component="p"
                                    variant="warning"
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
                            onChange={(event, value) => onChange(value, event)}
                            onBlur={handleBlur}
                            isDisabled={allowUnauthenticatedSmtp}
                        />
                    </FormLabelGroup>
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
                            onChange={(event, value) => onChange(value, event)}
                            onBlur={handleBlur}
                            isDisabled={!values.updatePassword || allowUnauthenticatedSmtp}
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
                            <span className="pf-v5-u-font-size-sm">
                                Specifies the email FROM header
                            </span>
                        }
                    >
                        <TextInput
                            type="text"
                            id="notifier.email.from"
                            value={values.notifier.email.from}
                            placeholder="example, Vulnerability Reports"
                            onChange={(event, value) => onChange(value, event)}
                            onBlur={handleBlur}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        isRequired
                        label="Sender"
                        fieldId="notifier.email.sender"
                        touched={touched}
                        errors={errors}
                        helperText={
                            <span className="pf-v5-u-font-size-sm">
                                Specifies the email SENDER header
                            </span>
                        }
                    >
                        <TextInput
                            isRequired
                            type="text"
                            id="notifier.email.sender"
                            value={values.notifier.email.sender}
                            placeholder="example, vulnerability-reports@example.com"
                            onChange={(event, value) => onChange(value, event)}
                            onBlur={handleBlur}
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
                            placeholder="example, vulnerability-reports-recipients@example.com"
                            onChange={(event, value) => onChange(value, event)}
                            onBlur={handleBlur}
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
                            onChange={(event, value) => onChange(value, event)}
                            onBlur={handleBlur}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup label="" fieldId="notifier.email.disableTLS" errors={errors}>
                        <Checkbox
                            label="Disable TLS (insecure)"
                            id="notifier.email.disableTLS"
                            isChecked={values.notifier.email.disableTLS}
                            onChange={(event, value) =>
                                updateStartTLSAuthMethodOnChange(value, event)
                            }
                            onBlur={handleBlur}
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
                            isDisabled={!values.notifier.email.disableTLS}
                            direction="up"
                        >
                            {startTLSAuthMethods.map(({ value, label }) => (
                                <SelectOption key={value} value={value}>
                                    {label}
                                </SelectOption>
                            ))}
                        </SelectSingle>
                    </FormLabelGroup>
                    <FormLabelGroup label="" fieldId="notifier.email.skipTLSVerify" errors={errors}>
                        <Checkbox
                            label="Skip TLS verification"
                            id="notifier.email.skipTLSVerify"
                            isChecked={values.notifier.email.skipTLSVerify}
                            onBlur={handleBlur}
                            isDisabled={
                                values.notifier.email.disableTLS &&
                                values.notifier.email.startTLSAuthMethod === 'DISABLED'
                            }
                        />
                    </FormLabelGroup>
                </Form>
            </PageSection>
        </>
    );
}

export default EmailNotifierForm;
