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

export type NexusIntegration = {
    id?: string;
    name: string;
    categories: 'REGISTRY'[];
    docker: {
        endpoint: string;
        username: string;
        password: string;
        insecure: boolean;
    };
    skipTestIntegration: boolean;
    type: 'nexus';
    enabled: boolean;
    clusterIds: string[];
};

export type NexusIntegrationFormValues = {
    config: NexusIntegration;
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
        docker: yup.object().shape({
            endpoint: yup.string().required('Required').min(1),
            username: yup.string(),
            password: yup.string(),
            insecure: yup.bool(),
        }),
        skipTestIntegration: yup.bool(),
        type: yup.string().matches(/nexus/),
        enabled: yup.bool(),
        clusterIds: yup.array().of(yup.string()),
    }),
    updatePassword: yup.bool(),
});

export const defaultValues: NexusIntegrationFormValues = {
    config: {
        name: '',
        categories: ['REGISTRY'],
        docker: {
            endpoint: '',
            username: '',
            password: '',
            insecure: false,
        },
        skipTestIntegration: false,
        type: 'nexus',
        enabled: true,
        clusterIds: [],
    },
    updatePassword: true,
};

function NexusIntegrationForm({
    initialValues = null,
    isEditable = false,
}: IntegrationFormProps<NexusIntegration>): ReactElement {
    const formInitialValues = defaultValues;
    if (initialValues) {
        formInitialValues.config = { ...formInitialValues.config, ...initialValues };
        // We want to clear the password because backend returns '******' to represent that there
        // are currently stored credentials
        formInitialValues.config.docker.password = '';
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
    } = useIntegrationForm<NexusIntegrationFormValues, typeof validationSchema>({
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
                            isRequired
                            type="text"
                            id="config.name"
                            name="config.name"
                            value={values.config.name}
                            onChange={onChange}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        label="Endpoint"
                        isRequired
                        fieldId="config.docker.endpoint"
                        errors={errors}
                    >
                        <TextInput
                            isRequired
                            type="text"
                            id="config.docker.endpoint"
                            name="config.docker.endpoint"
                            value={values.config.docker.endpoint}
                            onChange={onChange}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        label="Username"
                        fieldId="config.docker.username"
                        errors={errors}
                    >
                        <TextInput
                            isRequired
                            type="text"
                            id="config.docker.username"
                            name="config.docker.username"
                            value={values.config.docker.username}
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
                            label="Password"
                            fieldId="config.docker.password"
                            errors={errors}
                        >
                            <TextInput
                                isRequired
                                type="password"
                                id="config.docker.password"
                                name="config.docker.password"
                                value={values.config.docker.password}
                                onChange={onChange}
                                isDisabled={!isEditable}
                            />
                        </FormLabelGroup>
                    )}
                    <FormLabelGroup
                        label="Disable TLS Certificate Validation (Insecure)"
                        fieldId="config.docker.insecure"
                        errors={errors}
                    >
                        <Switch
                            id="config.docker.insecure"
                            name="config.docker.insecure"
                            aria-label="disable tls certificate validation"
                            isChecked={values.config.docker.insecure}
                            onChange={onChange}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
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

export default NexusIntegrationForm;
