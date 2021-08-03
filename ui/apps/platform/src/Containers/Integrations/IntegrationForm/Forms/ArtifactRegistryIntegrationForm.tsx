import React, { ReactElement } from 'react';
import { TextInput, PageSection, Form, Switch } from '@patternfly/react-core';
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

export type ArtifactRegistryIntegration = {
    id?: string;
    name: string;
    categories: 'REGISTRY'[];
    google: {
        endpoint: string;
        project: string;
        serviceAccount: string;
    };
    skipTestIntegration: boolean;
    type: 'artifactregistry';
    enabled: boolean;
    clusterIds: string[];
};

export type ArtifactRegistryIntegrationFormValues = {
    config: ArtifactRegistryIntegration;
    updatePassword: boolean;
};

export const validationSchema = yup.object().shape({
    config: yup.object().shape({
        name: yup.string().required('Required'),
        categories: yup
            .array()
            .of(yup.string().oneOf(['REGISTRY']))
            .min(1, 'Must have at least one type selected')
            .required('Required'),
        google: yup.object().shape({
            endpoint: yup.string().required('Required'),
            project: yup.string().required('Required'),
            serviceAccount: yup.string(),
        }),
        skipTestIntegration: yup.bool(),
        type: yup.string().matches(/artifactregistry/),
        enabled: yup.bool(),
        clusterIds: yup.array().of(yup.string()),
    }),
    updatePassword: yup.bool(),
});

export const defaultValues: ArtifactRegistryIntegrationFormValues = {
    config: {
        name: '',
        categories: ['REGISTRY'],
        google: {
            endpoint: '',
            project: '',
            serviceAccount: '',
        },
        skipTestIntegration: false,
        type: 'artifactregistry',
        enabled: true,
        clusterIds: [],
    },
    updatePassword: true,
};

function ArtifactRegistryIntegrationForm({
    initialValues = null,
    isEditable = false,
}: IntegrationFormProps<ArtifactRegistryIntegration>): ReactElement {
    const formInitialValues = defaultValues;
    if (initialValues) {
        formInitialValues.config = { ...formInitialValues.config, ...initialValues };
        // We want to clear the password because backend returns '******' to represent that there
        // are currently stored credentials
        formInitialValues.config.google.serviceAccount = '';
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
    } = useIntegrationForm<ArtifactRegistryIntegrationFormValues, typeof validationSchema>({
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
                    <FormLabelGroup label="Name" isRequired fieldId="config.name" errors={errors}>
                        <TextInput
                            type="text"
                            id="config.name"
                            name="config.name"
                            placeholder="(ex. Google Artifact Registry)"
                            value={values.config.name}
                            onChange={onChange}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        label="Registry Endpoint"
                        isRequired
                        fieldId="config.google.endpoint"
                        errors={errors}
                    >
                        <TextInput
                            type="text"
                            id="config.google.endpoint"
                            name="config.google.endpoint"
                            placeholder="(ex. us-west1-docker.pkg.dev)"
                            value={values.config.google.endpoint}
                            onChange={onChange}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        label="Project"
                        isRequired
                        fieldId="config.google.project"
                        errors={errors}
                    >
                        <TextInput
                            type="text"
                            id="config.google.project"
                            name="config.google.project"
                            value={values.config.google.project}
                            onChange={onChange}
                            isDisabled={!isEditable}
                        />
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
                        <FormLabelGroup
                            label="Service Account Key (JSON)"
                            isRequired
                            fieldId="config.google.serviceAccount"
                            errors={errors}
                        >
                            <TextInput
                                type="text"
                                id="config.google.serviceAccount"
                                name="config.google.serviceAccount"
                                value={values.config.google.serviceAccount}
                                onChange={onChange}
                                isDisabled={!isEditable}
                            />
                        </FormLabelGroup>
                    )}

                    <FormLabelGroup
                        label="Create Integration Without Testing"
                        fieldId="config.skipTestIntegration"
                        errors={errors}
                    >
                        <Switch
                            id="config.skipTestIntegration"
                            name="config.skipTestIntegration"
                            aria-label="skip test integration"
                            isChecked={values.config.skipTestIntegration}
                            onChange={onChange}
                            isDisabled={!isEditable}
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

export default ArtifactRegistryIntegrationForm;
