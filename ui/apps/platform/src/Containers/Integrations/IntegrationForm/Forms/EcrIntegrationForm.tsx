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

export type EcrIntegration = {
    id?: string;
    name: string;
    categories: 'REGISTRY'[];
    ecr: {
        registryId: string;
        endpoint: string;
        region: string;
        useIam: boolean;
        accessKeyId: string;
        secretAccessKey: string;
    };
    skipTestIntegration: boolean;
    type: 'ecr';
    enabled: boolean;
    clusterIds: string[];
};

export type EcrIntegrationFormValues = {
    config: EcrIntegration;
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
        ecr: yup.object().shape({
            registryId: yup.string().required('Required'),
            endpoint: yup.string().required('Required'),
            region: yup.string().required('Required'),
            useIam: yup.bool(),
            accessKeyId: yup.string().when('useIam', {
                is: false,
                then: yup.string().required('Required if not using IAM'),
            }),
            secretAccessKey: yup.string(),
        }),
        skipTestIntegration: yup.bool(),
        type: yup.string().matches(/ecr/),
        enabled: yup.bool(),
        clusterIds: yup.array().of(yup.string()),
    }),
    updatePassword: yup.bool(),
});

export const defaultValues: EcrIntegrationFormValues = {
    config: {
        name: '',
        categories: ['REGISTRY'],
        ecr: {
            registryId: '',
            endpoint: '',
            region: '',
            useIam: true,
            accessKeyId: '',
            secretAccessKey: '',
        },
        skipTestIntegration: false,
        type: 'ecr',
        enabled: true,
        clusterIds: [],
    },
    updatePassword: true,
};

function EcrIntegrationForm({
    initialValues = null,
    isEditable = false,
}: IntegrationFormProps<EcrIntegration>): ReactElement {
    const formInitialValues = defaultValues;
    if (initialValues) {
        formInitialValues.config = { ...formInitialValues.config, ...initialValues };
        // We want to clear the password because backend returns '******' to represent that there
        // are currently stored credentials
        formInitialValues.config.ecr.accessKeyId = '';
        formInitialValues.config.ecr.secretAccessKey = '';
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
    } = useIntegrationForm<EcrIntegrationFormValues, typeof validationSchema>({
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
                            value={values.config.name}
                            onChange={onChange}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        label="Registry ID"
                        isRequired
                        fieldId="config.ecr.registryId"
                        errors={errors}
                    >
                        <TextInput
                            type="text"
                            id="config.ecr.registryId"
                            name="config.ecr.registryId"
                            value={values.config.ecr.registryId}
                            onChange={onChange}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        label="Endpoint"
                        isRequired
                        fieldId="config.ecr.endpoint"
                        errors={errors}
                    >
                        <TextInput
                            type="text"
                            id="config.ecr.endpoint"
                            name="config.ecr.endpoint"
                            value={values.config.ecr.endpoint}
                            onChange={onChange}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        label="Region"
                        isRequired
                        fieldId="config.ecr.region"
                        errors={errors}
                    >
                        <TextInput
                            isRequired
                            type="text"
                            id="config.ecr.region"
                            name="config.ecr.region"
                            value={values.config.ecr.region}
                            onChange={onChange}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        label="Use Container IAM Role"
                        fieldId="config.ecr.useIam"
                        errors={errors}
                    >
                        <Switch
                            id="config.ecr.useIam"
                            name="config.ecr.useIam"
                            aria-label="use container iam role"
                            isChecked={values.config.ecr.useIam}
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
                    {values.updatePassword && !values.config.ecr.useIam && (
                        <>
                            <FormLabelGroup
                                label="Access Key ID"
                                fieldId="config.ecr.accessKeyId"
                                isRequired
                                errors={errors}
                            >
                                <TextInput
                                    type="password"
                                    id="config.ecr.accessKeyId"
                                    name="config.ecr.accessKeyId"
                                    value={values.config.ecr.accessKeyId}
                                    onChange={onChange}
                                    isDisabled={!isEditable}
                                />
                            </FormLabelGroup>
                            <FormLabelGroup
                                label="Secret Access Key"
                                fieldId="config.ecr.secretAccessKey"
                                isRequired
                                errors={errors}
                            >
                                <TextInput
                                    type="password"
                                    id="config.ecr.secretAccessKey"
                                    name="config.ecr.secretAccessKey"
                                    value={values.config.ecr.secretAccessKey}
                                    onChange={onChange}
                                    isDisabled={!isEditable}
                                />
                            </FormLabelGroup>
                        </>
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

export default EcrIntegrationForm;
