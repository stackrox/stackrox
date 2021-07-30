import React, { ReactElement } from 'react';
import { TextInput, PageSection, Form, Switch, SelectOption } from '@patternfly/react-core';
import * as yup from 'yup';

import FormMultiSelect from 'Components/FormMultiSelect';
import usePageState from 'Containers/Integrations/hooks/usePageState';
import useIntegrationForm from '../useIntegrationForm';
import { IntegrationFormProps } from '../integrationFormTypes';

import IntegrationFormActions from '../IntegrationFormActions';
import FormCancelButton from '../FormCancelButton';
import FormTestButton from '../FormTestButton';
import FormSaveButton from '../FormSaveButton';
import FormMessageBanner from '../FormMessageBanner';
import FormLabelGroup from '../FormLabelGroup';

export type QuayIntegration = {
    id?: string;
    name: string;
    categories: ('REGISTRY' | 'SCANNER')[];
    quay: {
        endpoint: string;
        oauthToken: string;
        insecure: boolean;
    };
    skipTestIntegration: boolean;
    type: 'quay';
    enabled: boolean;
    clusterIds: string[];
};

export type QuayIntegrationFormValues = {
    config: QuayIntegration;
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
        quay: yup.object().shape({
            endpoint: yup.string().required('Required').min(1),
            oauthToken: yup.string(),
            insecure: yup.bool(),
        }),
        skipTestIntegration: yup.bool(),
        type: yup.string().matches(/quay/),
        enabled: yup.bool(),
        clusterIds: yup.array().of(yup.string()),
    }),
    updatePassword: yup.bool(),
});

export const defaultValues: QuayIntegrationFormValues = {
    config: {
        name: '',
        categories: [],
        quay: {
            endpoint: '',
            oauthToken: '',
            insecure: false,
        },
        skipTestIntegration: false,
        type: 'quay',
        enabled: true,
        clusterIds: [],
    },
    updatePassword: true,
};

function QuayIntegrationForm({
    initialValues = null,
    isEditable = false,
}: IntegrationFormProps<QuayIntegration>): ReactElement {
    const formInitialValues = defaultValues;
    if (initialValues) {
        formInitialValues.config = { ...formInitialValues.config, ...initialValues };
        // We want to clear the password because backend returns '******' to represent that there
        // are currently stored credentials
        formInitialValues.config.quay.oauthToken = '';
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
    } = useIntegrationForm<QuayIntegrationFormValues, typeof validationSchema>({
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
            {message && <FormMessageBanner message={message} />}
            <PageSection variant="light" isFilled hasOverflowScroll>
                <Form isWidthLimited>
                    <FormLabelGroup label="Name" isRequired fieldId="config.name" errors={errors}>
                        <TextInput
                            isRequired
                            type="text"
                            id="config.name"
                            name="config.name"
                            placeholder="(ex. Quay)"
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
                        label="Endpoint"
                        isRequired
                        fieldId="config.quay.endpoint"
                        errors={errors}
                    >
                        <TextInput
                            isRequired
                            type="text"
                            id="config.quay.endpoint"
                            name="config.quay.endpoint"
                            placeholder="(ex. quay.io)"
                            value={values.config.quay.endpoint}
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
                            label="OAuth Token"
                            fieldId="config.quay.oauthToken"
                            isRequired
                            errors={errors}
                        >
                            <TextInput
                                isRequired
                                type="text"
                                id="config.quay.oauthToken"
                                name="config.quay.oauthToken"
                                value={values.config.quay.oauthToken}
                                onChange={onChange}
                                isDisabled={!isEditable}
                            />
                        </FormLabelGroup>
                    )}
                    <FormLabelGroup
                        label="Disable TLS Certificate Validation (Insecure)"
                        fieldId="config.quay.insecure"
                        errors={errors}
                    >
                        <Switch
                            id="config.quay.insecure"
                            name="config.quay.insecure"
                            aria-label="disable tls certificate validation"
                            isChecked={values.config.quay.insecure}
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

export default QuayIntegrationForm;
