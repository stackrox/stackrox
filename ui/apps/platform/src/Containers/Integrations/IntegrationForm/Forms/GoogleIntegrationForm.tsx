import React, { ReactElement } from 'react';
import { TextInput, PageSection, Form, SelectOption, Switch } from '@patternfly/react-core';
import * as yup from 'yup';

import FormMultiSelect from 'Components/FormMultiSelect';
import usePageState from 'Containers/Integrations/hooks/usePageState';
import useIntegrationForm from '../useIntegrationForm';
import { IntegrationFormProps } from '../integrationFormTypes';

import IntegrationFormActions from '../IntegrationFormActions';
import FormCancelButton from '../FormCancelButton';
import FormTestButton from '../FormTestButton';
import FormSaveButton from '../FormSaveButton';
import FormMessage from '../FormMessage';
import FormLabelGroup from '../FormLabelGroup';

export type GoogleIntegration = {
    id?: string;
    name: string;
    categories: ('REGISTRY' | 'SCANNER')[];
    google: {
        endpoint: string;
        project: string;
        serviceAccount: string;
    };
    skipTestIntegration: boolean;
    type: 'google';
    enabled: boolean;
    clusterIds: string[];
};

export type GoogleIntegrationFormValues = {
    config: GoogleIntegration;
    updatePassword: boolean;
};

export const validationSchema = yup.object().shape({
    config: yup.object().shape({
        name: yup.string().required('Required'),
        categories: yup
            .array()
            .of(yup.string().oneOf(['REGISTRY', 'SCANNER']))
            .min(1, 'Must have at least one type selected')
            .required('Required'),
        google: yup.object().shape({
            endpoint: yup.string().required('Required'),
            project: yup.string().required('Required'),
            serviceAccount: yup.string(),
        }),
        skipTestIntegration: yup.bool(),
        type: yup.string().matches(/google/),
        enabled: yup.bool(),
        clusterIds: yup.array().of(yup.string()),
    }),
    updatePassword: yup.bool(),
});

export const defaultValues: GoogleIntegrationFormValues = {
    config: {
        name: '',
        categories: [],
        google: {
            endpoint: '',
            project: '',
            serviceAccount: '',
        },
        skipTestIntegration: false,
        type: 'google',
        enabled: true,
        clusterIds: [],
    },
    updatePassword: true,
};

function DockerIntegrationForm({
    initialValues = null,
    isEditable = false,
}: IntegrationFormProps<GoogleIntegration>): ReactElement {
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
    } = useIntegrationForm<GoogleIntegrationFormValues, typeof validationSchema>({
        initialValues: formInitialValues,
        validationSchema,
    });

    const { isCreating } = usePageState();

    function onChange(value, event) {
        return setFieldValue(event.target.id, value, false);
    }

    function onCustomChange(id, value) {
        return setFieldValue(id, value, false);
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
                            placeholder="(ex. Google Registry and Scanner)"
                            value={values.config.name}
                            onChange={onChange}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        label="Type"
                        isRequired
                        fieldId="config.categories"
                        errors={errors}
                    >
                        <FormMultiSelect
                            id="config.categories"
                            values={values.config.categories}
                            onChange={onCustomChange}
                            isDisabled={!isEditable}
                        >
                            <SelectOption key={0} value="REGISTRY">
                                Registry
                            </SelectOption>
                            <SelectOption key={1} value="SCANNER">
                                Scanner
                            </SelectOption>
                        </FormMultiSelect>
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
                            placeholder="(ex. gcr.io)"
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

export default DockerIntegrationForm;
