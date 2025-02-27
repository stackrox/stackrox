import React, { ReactElement } from 'react';
import { TextInput, PageSection, Form, Checkbox } from '@patternfly/react-core';
import * as yup from 'yup';
import merge from 'lodash/merge';

import { ImageIntegrationBase } from 'services/ImageIntegrationsService';

import usePageState from 'Containers/Integrations/hooks/usePageState';
import FormMessage from 'Components/PatternFly/FormMessage';
import FormTestButton from 'Components/PatternFly/FormTestButton';
import FormSaveButton from 'Components/PatternFly/FormSaveButton';
import FormCancelButton from 'Components/PatternFly/FormCancelButton';
import useIntegrationForm from '../useIntegrationForm';
import { IntegrationFormProps } from '../integrationFormTypes';

import IntegrationFormActions from '../IntegrationFormActions';
import FormLabelGroup from '../FormLabelGroup';

export type DockerIntegration = {
    categories: 'REGISTRY'[];
    docker: {
        endpoint: string;
        username: string;
        password: string;
        insecure: boolean;
    };
    type: 'docker';
} & ImageIntegrationBase;

export type DockerIntegrationFormValues = {
    config: DockerIntegration;
    updatePassword: boolean;
};

export const validationSchema = yup.object().shape({
    config: yup.object().shape({
        name: yup.string().trim().required('An integration name is required'),
        categories: yup
            .array()
            .of(yup.string().trim().oneOf(['REGISTRY']))
            .min(1, 'Must have at least one type selected')
            .required('A category is required'),
        docker: yup.object().shape({
            endpoint: yup.string().trim().required('An endpoint is required'),
            username: yup.string(),
            password: yup.string(),
            insecure: yup.bool(),
        }),
        skipTestIntegration: yup.bool(),
        type: yup.string().matches(/docker/),
    }),
    updatePassword: yup.bool(),
});

export const defaultValues: DockerIntegrationFormValues = {
    config: {
        id: '',
        name: '',
        categories: ['REGISTRY'],
        docker: {
            endpoint: '',
            username: '',
            password: '',
            insecure: false,
        },
        autogenerated: false,
        clusterId: '',
        clusters: [],
        skipTestIntegration: false,
        type: 'docker',
    },
    updatePassword: true,
};

function DockerIntegrationForm({
    initialValues = null,
    isEditable = false,
}: IntegrationFormProps<DockerIntegration>): ReactElement {
    const formInitialValues = structuredClone(defaultValues);
    if (initialValues) {
        merge(formInitialValues.config, initialValues);

        // We want to clear the password because backend returns '******' to represent that there
        // are currently stored credentials
        formInitialValues.config.docker.password = '';

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
    } = useIntegrationForm<DockerIntegrationFormValues>({
        initialValues: formInitialValues,
        validationSchema,
    });
    const { isCreating } = usePageState();

    function onChange(value, event) {
        return setFieldValue(event.target.id, value);
    }

    function onUpdateCredentialsChange(value, event) {
        setFieldValue('config.docker.password', '');
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
                        fieldId="config.name"
                        touched={touched}
                        errors={errors}
                    >
                        <TextInput
                            isRequired
                            type="text"
                            id="config.name"
                            value={values.config.name}
                            onChange={(event, value) => onChange(value, event)}
                            onBlur={handleBlur}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        label="Endpoint"
                        isRequired
                        fieldId="config.docker.endpoint"
                        touched={touched}
                        errors={errors}
                    >
                        <TextInput
                            isRequired
                            type="text"
                            id="config.docker.endpoint"
                            value={values.config.docker.endpoint}
                            onChange={(event, value) => onChange(value, event)}
                            onBlur={handleBlur}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        label="Username"
                        fieldId="config.docker.username"
                        touched={touched}
                        errors={errors}
                    >
                        <TextInput
                            isRequired
                            type="text"
                            id="config.docker.username"
                            value={values.config.docker.username}
                            onChange={(event, value) => onChange(value, event)}
                            onBlur={handleBlur}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    {!isCreating && isEditable && (
                        <FormLabelGroup
                            fieldId="updatePassword"
                            helperText="Enable this option to replace currently stored credentials (if any)"
                            errors={errors}
                        >
                            <Checkbox
                                label="Update stored credentials"
                                id="updatePassword"
                                isChecked={values.updatePassword}
                                onChange={(event, value) => onUpdateCredentialsChange(value, event)}
                                onBlur={handleBlur}
                                isDisabled={!isEditable}
                            />
                        </FormLabelGroup>
                    )}
                    <FormLabelGroup
                        label="Password"
                        fieldId="config.docker.password"
                        touched={touched}
                        errors={errors}
                    >
                        <TextInput
                            type="password"
                            id="config.docker.password"
                            value={values.config.docker.password}
                            onChange={(event, value) => onChange(value, event)}
                            onBlur={handleBlur}
                            isDisabled={!isEditable || !values.updatePassword}
                            placeholder={
                                values.updatePassword
                                    ? ''
                                    : 'Currently-stored password will be used.'
                            }
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        fieldId="config.docker.insecure"
                        touched={touched}
                        errors={errors}
                    >
                        <Checkbox
                            label="Disable TLS certificate validation (insecure)"
                            id="config.docker.insecure"
                            isChecked={values.config.docker.insecure}
                            onChange={(event, value) => onChange(value, event)}
                            onBlur={handleBlur}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        fieldId="config.skipTestIntegration"
                        touched={touched}
                        errors={errors}
                    >
                        <Checkbox
                            label="Create integration without testing"
                            id="config.skipTestIntegration"
                            isChecked={values.config.skipTestIntegration}
                            onChange={(event, value) => onChange(value, event)}
                            onBlur={handleBlur}
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

export default DockerIntegrationForm;
