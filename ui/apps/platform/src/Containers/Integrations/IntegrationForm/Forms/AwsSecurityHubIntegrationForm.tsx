import React, { ReactElement } from 'react';
import { TextInput, PageSection, Form, FormSelect, Switch } from '@patternfly/react-core';
import * as yup from 'yup';

import usePageState from 'Containers/Integrations/hooks/usePageState';
import useIntegrationForm from '../useIntegrationForm';
import { IntegrationFormProps } from '../integrationFormTypes';

import IntegrationFormActions from '../IntegrationFormActions';
import FormCancelButton from '../FormCancelButton';
import FormTestButton from '../FormTestButton';
import FormSaveButton from '../FormSaveButton';
import FormMessage from '../FormMessage';
import FormLabelGroup from '../FormLabelGroup';
import AwsRegionOptions from '../AwsRegionOptions';

export type AwsSecurityHubIntegration = {
    id?: string;
    name: string;
    awsSecurityHub: {
        accountId: string;
        region: string;
        credentials: {
            accessKeyId: string;
            secretAccessKey: string;
        };
    };
    uiEndpoint: string;
    type: 'awsSecurityHub';
    enabled: boolean;
};

export type AwsSecurityHubIntegrationFormValues = {
    notifier: AwsSecurityHubIntegration;
    updatePassword: boolean;
};

export const validationSchema = yup.object().shape({
    notifier: yup.object().shape({
        name: yup.string().required('Required'),
        awsSecurityHub: yup.object().shape({
            accountId: yup.string().required('Required'),
            region: yup.string().required('Required'),
            credentials: yup.object().shape({
                accessKeyId: yup.string(),
                secretAccessKey: yup.string(),
            }),
        }),
        uiEndpoint: yup.string(),
        type: yup.string().matches(/awsSecurityHub/),
        enabled: yup.bool(),
    }),
    updatePassword: yup.bool(),
});

export const defaultValues: AwsSecurityHubIntegrationFormValues = {
    notifier: {
        name: '',
        awsSecurityHub: {
            accountId: '',
            region: '',
            credentials: {
                accessKeyId: '',
                secretAccessKey: '',
            },
        },
        uiEndpoint: window.location.origin,
        type: 'awsSecurityHub',
        enabled: true,
    },
    updatePassword: true,
};

function AwsSecurityHubIntegrationForm({
    initialValues = null,
    isEditable = false,
}: IntegrationFormProps<AwsSecurityHubIntegration>): ReactElement {
    const formInitialValues = defaultValues;
    if (initialValues) {
        formInitialValues.notifier = {
            ...formInitialValues.notifier,
            ...initialValues,
        };
        // We want to clear the password because backend returns '******' to represent that there
        // are currently stored credentials
        formInitialValues.notifier.awsSecurityHub.credentials.accessKeyId = '';
        formInitialValues.notifier.awsSecurityHub.credentials.secretAccessKey = '';
    }
    const {
        values,
        errors,
        setFieldValue,
        isSubmitting,
        isTesting,
        onSave,
        onTest,
        onCancel,
        message,
    } = useIntegrationForm<AwsSecurityHubIntegrationFormValues, typeof validationSchema>({
        initialValues: formInitialValues,
        validationSchema,
    });
    const { isCreating } = usePageState();

    function onChange(value, event) {
        return setFieldValue(event.target.id, value, false);
    }

    return (
        <>
            <PageSection variant="light" isFilled hasOverflowScroll>
                {message && <FormMessage message={message} />}
                <Form isWidthLimited>
                    <FormLabelGroup isRequired label="Name" fieldId="notifier.name" errors={errors}>
                        <TextInput
                            type="text"
                            id="notifier.name"
                            name="notifier.name"
                            value={values.notifier.name}
                            onChange={onChange}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        isRequired
                        label="AWS Account Number"
                        fieldId="notifier.awsSecurityHub.accountId"
                        errors={errors}
                    >
                        <TextInput
                            type="text"
                            id="notifier.awsSecurityHub.accountId"
                            name="notifier.awsSecurityHub.accountId"
                            value={values.notifier.awsSecurityHub.accountId}
                            onChange={onChange}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        isRequired
                        label="AWS Region"
                        fieldId="notifier.awsSecurityHub.region"
                        errors={errors}
                    >
                        <FormSelect
                            id="notifier.awsSecurityHub.region"
                            value={values.notifier.awsSecurityHub.region}
                            onChange={onChange}
                            isDisabled={!isEditable}
                        >
                            <AwsRegionOptions />
                        </FormSelect>
                    </FormLabelGroup>
                    {!isCreating && (
                        <FormLabelGroup
                            label="Update Password"
                            fieldId="updatePassword"
                            helperText="Setting this to false will use the currently stored credentials, if they exist."
                            errors={errors}
                        >
                            <Switch
                                id="updatePassword"
                                name="updatePassword"
                                aria-label="update password"
                                isChecked={values.updatePassword}
                                onChange={onChange}
                                isDisabled={!isEditable}
                            />
                        </FormLabelGroup>
                    )}
                    {values.updatePassword && (
                        <>
                            <FormLabelGroup
                                label="Access Key ID"
                                fieldId="notifier.awsSecurityHub.credentials.accessKeyId"
                                errors={errors}
                            >
                                <TextInput
                                    type="password"
                                    id="notifier.awsSecurityHub.credentials.accessKeyId"
                                    name="notifier.awsSecurityHub.credentials.accessKeyId"
                                    value={values.notifier.awsSecurityHub.credentials.accessKeyId}
                                    onChange={onChange}
                                    isDisabled={!isEditable}
                                />
                            </FormLabelGroup>
                            <FormLabelGroup
                                label="Secret Access Key"
                                fieldId="notifier.awsSecurityHub.credentials.secretAccessKey"
                                errors={errors}
                            >
                                <TextInput
                                    type="password"
                                    id="notifier.awsSecurityHub.credentials.secretAccessKey"
                                    name="notifier.awsSecurityHub.credentials.secretAccessKey"
                                    value={
                                        values.notifier.awsSecurityHub.credentials.secretAccessKey
                                    }
                                    onChange={onChange}
                                    isDisabled={!isEditable}
                                />
                            </FormLabelGroup>
                        </>
                    )}
                </Form>
            </PageSection>
            {isEditable && (
                <IntegrationFormActions>
                    <FormSaveButton
                        onSave={onSave}
                        isSubmitting={isSubmitting}
                        isTesting={isTesting}
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

export default AwsSecurityHubIntegrationForm;
