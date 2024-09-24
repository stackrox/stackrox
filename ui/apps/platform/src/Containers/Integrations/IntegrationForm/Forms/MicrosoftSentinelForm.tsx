/* eslint-disable no-void */
import React, { ReactElement } from 'react';
import { Checkbox, Form, PageSection, TextInput } from '@patternfly/react-core';
import * as yup from 'yup';

import { NotifierIntegrationBase } from 'services/NotifierIntegrationsService';

import usePageState from 'Containers/Integrations/hooks/usePageState';
import FormMessage from 'Components/PatternFly/FormMessage';
import FormCancelButton from 'Components/PatternFly/FormCancelButton';
import FormTestButton from 'Components/PatternFly/FormTestButton';
import FormSaveButton from 'Components/PatternFly/FormSaveButton';
import useIntegrationForm from '../useIntegrationForm';
import { IntegrationFormProps } from '../integrationFormTypes';

import IntegrationFormActions from '../IntegrationFormActions';
import FormLabelGroup from '../FormLabelGroup';

export type MicrosoftSentinel = {
    microsoftSentinel: {
        logIngestionEndpoint: string;
        directoryTenantId: string;
        applicationClientId: string;
        secret: string;
        alertDcrConfig: {
            dataCollectionRuleId: string;
            streamName: string;
            enabled: boolean;
        };
        auditLogDcrConfig: {
            dataCollectionRuleId: string;
            streamName: string;
            enabled: boolean;
        };
    };
    type: 'microsoftSentinel';
} & NotifierIntegrationBase;

export type MicrosoftSentinelFormValues = {
    notifier: MicrosoftSentinel;
    updatePassword: boolean;
};

export const validationSchema = yup.object().shape({
    notifier: yup.object().shape({
        name: yup.string().trim().required('Email integration name is required'),
        microsoftSentinel: yup.object().shape({
            logIngestionEndpoint: yup
                .string()
                .trim()
                .required('A log ingestion endpoint is required'),
            directoryTenantId: yup.string().trim().required('A directory tenant ID is required'),
            applicationClientId: yup
                .string()
                .trim()
                .required('A application client ID is required'),
            secret: yup.string(),
        }),
    }),
    updatePassword: yup.bool(),
});

export const defaultValues: MicrosoftSentinelFormValues = {
    notifier: {
        id: '',
        name: '',
        microsoftSentinel: {
            logIngestionEndpoint: '',
            directoryTenantId: '',
            applicationClientId: '',
            secret: '',
            alertDcrConfig: {
                dataCollectionRuleId: '',
                streamName: '',
                enabled: false,
            },
            auditLogDcrConfig: {
                dataCollectionRuleId: '',
                streamName: '',
                enabled: false,
            },
        },
        labelDefault: '',
        labelKey: '',
        uiEndpoint: window.location.origin,
        type: 'microsoftSentinel',
    },
    updatePassword: true,
};

function MicrosoftSentinelForm({
    initialValues = null,
    isEditable = false,
}: IntegrationFormProps<MicrosoftSentinel>): ReactElement {
    const formInitialValues = { ...defaultValues, ...initialValues };
    if (initialValues) {
        formInitialValues.notifier = {
            ...formInitialValues.notifier,
            ...initialValues,
        };
        // We want to clear the password because backend returns '******' to represent that there
        // are currently stored credentials
        formInitialValues.notifier.microsoftSentinel.secret = '';
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
    } = useIntegrationForm<MicrosoftSentinelFormValues>({
        initialValues: formInitialValues,
        validationSchema,
    });
    const { isCreating } = usePageState();

    function onChange(value, event) {
        return setFieldValue(event.target.id, value);
    }

    function onUpdateCredentialsChange(value, event) {
        setFieldValue('notifier.microsoftSentinel.secret', '');
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
                        helperText="(example, Microsoft Integration)"
                        errors={errors}
                    >
                        <TextInput
                            isRequired
                            type="text"
                            id="notifier.name"
                            value={values.notifier.name}
                            onChange={(event, value) => onChange(value, event)}
                            onBlur={handleBlur}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        label="Log ingestion endpoint"
                        isRequired
                        fieldId="notifier.microsoftSentinel.logIngestionEndpoint"
                        touched={touched}
                        helperText="(example, https://example-sentinel-ou812.eastus-1.ingest.monitor.azure.com)"
                        errors={errors}
                    >
                        <TextInput
                            isRequired
                            type="text"
                            id="notifier.microsoftSentinel.logIngestionEndpoint"
                            value={values.notifier.microsoftSentinel.logIngestionEndpoint}
                            onChange={(event, value) => onChange(value, event)}
                            onBlur={handleBlur}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        label="Directory tenant ID"
                        isRequired
                        fieldId="notifier.microsoftSentinel.directoryTenantId"
                        touched={touched}
                        helperText="(example, 1234abce-1234-abcd-1234-abcd1234abcd)"
                        errors={errors}
                    >
                        <TextInput
                            isRequired
                            type="text"
                            id="notifier.microsoftSentinel.directoryTenantId"
                            value={values.notifier.microsoftSentinel.directoryTenantId}
                            onChange={(event, value) => onChange(value, event)}
                            onBlur={handleBlur}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        label="Application client ID"
                        isRequired
                        fieldId="notifier.microsoftSentinel.applicationClientId"
                        touched={touched}
                        helperText="(example, abcd1234-abcd-1234-abcd-1234abce1234)"
                        errors={errors}
                    >
                        <TextInput
                            isRequired
                            type="text"
                            id="notifier.microsoftSentinel.applicationClientId"
                            value={values.notifier.microsoftSentinel.applicationClientId}
                            onChange={(event, value) => onChange(value, event)}
                            onBlur={handleBlur}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    {!isCreating && isEditable && (
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
                                onChange={(event, value) => onUpdateCredentialsChange(value, event)}
                                onBlur={handleBlur}
                                isDisabled={!isEditable}
                            />
                        </FormLabelGroup>
                    )}
                    <FormLabelGroup
                        label="Secret"
                        isRequired={values.updatePassword}
                        fieldId="notifier.microsoftSentinel.secret"
                        touched={touched}
                        errors={errors}
                    >
                        <TextInput
                            isRequired={values.updatePassword}
                            type="password"
                            id="notifier.microsoftSentinel.secret"
                            value={values.notifier.microsoftSentinel.secret}
                            onChange={(event, value) => onChange(value, event)}
                            onBlur={handleBlur}
                            isDisabled={!isEditable || !values.updatePassword}
                            placeholder={
                                values.updatePassword ? '' : 'Currently-stored secret will be used.'
                            }
                        />
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

export default MicrosoftSentinelForm;
