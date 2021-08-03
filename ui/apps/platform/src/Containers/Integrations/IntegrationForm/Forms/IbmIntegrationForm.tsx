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

export type IbmIntegration = {
    id?: string;
    name: string;
    categories: 'REGISTRY'[];
    ibm: {
        endpoint: string;
        apiKey: string;
    };
    type: 'ibm';
    enabled: boolean;
    clusterIds: string[];
};

export type IbmIntegrationFormValues = {
    config: IbmIntegration;
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
        ibm: yup.object().shape({
            endpoint: yup.string().required('Required'),
            apiKey: yup.string(),
        }),
        type: yup.string().matches(/ibm/),
        enabled: yup.bool(),
        clusterIds: yup.array().of(yup.string()),
    }),
    updatePassword: yup.bool(),
});

export const defaultValues: IbmIntegrationFormValues = {
    config: {
        name: '',
        categories: ['REGISTRY'],
        ibm: {
            endpoint: '',
            apiKey: '',
        },
        type: 'ibm',
        enabled: true,
        clusterIds: [],
    },
    updatePassword: true,
};

function IbmIntegrationForm({
    initialValues = null,
    isEditable = false,
}: IntegrationFormProps<IbmIntegration>): ReactElement {
    const formInitialValues = defaultValues;
    if (initialValues) {
        formInitialValues.config = { ...formInitialValues.config, ...initialValues };
        // We want to clear the password because backend returns '******' to represent that there
        // are currently stored credentials
        formInitialValues.config.ibm.apiKey = '';
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
    } = useIntegrationForm<IbmIntegrationFormValues, typeof validationSchema>({
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
                        label="Endpoint"
                        fieldId="config.ibm.endpoint"
                        isRequired
                        errors={errors}
                    >
                        <TextInput
                            type="text"
                            id="config.ibm.endpoint"
                            name="config.ibm.endpoint"
                            value={values.config.ibm.endpoint}
                            onChange={onChange}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    {!isCreating && (
                        <FormLabelGroup
                            label="Update Password"
                            fieldId="updatePassword"
                            isRequired
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
                        <FormLabelGroup label="API Key" fieldId="config.ibm.apiKey" errors={errors}>
                            <TextInput
                                type="password"
                                id="config.ibm.apiKey"
                                name="config.ibm.apiKey"
                                value={values.config.ibm.apiKey}
                                onChange={onChange}
                                isDisabled={!isEditable}
                            />
                        </FormLabelGroup>
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

export default IbmIntegrationForm;
