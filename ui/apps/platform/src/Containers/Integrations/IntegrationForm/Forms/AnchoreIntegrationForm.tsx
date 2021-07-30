import React, { ReactElement } from 'react';
import { TextInput, PageSection, Form, Switch } from '@patternfly/react-core';
import * as yup from 'yup';

import usePageState from 'Containers/Integrations/hooks/usePageState';
import useIntegrationForm from '../useIntegrationForm';

import IntegrationFormActions from '../IntegrationFormActions';
import FormCancelButton from '../FormCancelButton';
import FormTestButton from '../FormTestButton';
import FormSaveButton from '../FormSaveButton';
import FormMessageBanner from '../FormMessageBanner';
import FormLabelGroup from '../FormLabelGroup';

export type AnchoreIntegration = {
    id?: string;
    name: string;
    categories: 'REGISTRY'[];
    anchore: {
        endpoint: string;
        username: string;
        password: string;
        insecure: boolean;
    };
    skipTestIntegration: boolean;
    type: 'anchore';
    enabled: boolean;
    clusterIds: string[];
};

export type AnchoreIntegrationFormValues = {
    config: AnchoreIntegration;
    updatePassword: boolean;
};

export type AnchoreIntegrationFormProps = {
    initialValues: AnchoreIntegration | null;
    isEdittable?: boolean;
};

export const validationSchema = yup.object().shape({
    config: yup.object().shape({
        name: yup.string().required('Required'),
        categories: yup
            .array()
            .of(yup.string().oneOf(['REGISTRY']))
            .min(1, 'Must have at least one type selected')
            .required('Required'),
        anchore: yup.object().shape({
            endpoint: yup.string().required('Required').min(1),
            username: yup.string(),
            password: yup.string(),
            insecure: yup.bool(),
        }),
        skipTestIntegration: yup.bool(),
        type: yup.string().matches(/anchore/),
        enabled: yup.bool(),
        clusterIds: yup.array().of(yup.string()),
    }),
    updatePassword: yup.bool(),
});

export const defaultValues: AnchoreIntegrationFormValues = {
    config: {
        name: '',
        categories: ['REGISTRY'],
        anchore: {
            endpoint: '',
            username: '',
            password: '',
            insecure: false,
        },
        skipTestIntegration: false,
        type: 'anchore',
        enabled: true,
        clusterIds: [],
    },
    updatePassword: true,
};

function AnchoreIntegrationForm({
    initialValues = null,
    isEdittable = false,
}: AnchoreIntegrationFormProps): ReactElement {
    const formInitialValues = defaultValues;
    if (initialValues) {
        formInitialValues.config = {
            ...formInitialValues.config,
            ...initialValues,
        };
        // We want to clear the password because backend returns '******' to represent that there
        // are currently stored credentials
        formInitialValues.config.anchore.password = '';
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
    } = useIntegrationForm<AnchoreIntegrationFormValues, typeof validationSchema>({
        initialValues: formInitialValues,
        validationSchema,
    });
    const { isCreating } = usePageState();

    function onChange(value, event) {
        return setFieldValue(event.target.id, value, false);
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
                            value={values.config.name}
                            onChange={onChange}
                            isDisabled={!isEdittable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        label="Endpoint"
                        isRequired
                        fieldId="config.anchore.endpoint"
                        errors={errors}
                    >
                        <TextInput
                            isRequired
                            type="text"
                            id="config.anchore.endpoint"
                            name="config.anchore.endpoint"
                            value={values.config.anchore.endpoint}
                            onChange={onChange}
                            isDisabled={!isEdittable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        label="Username"
                        fieldId="config.anchore.username"
                        errors={errors}
                    >
                        <TextInput
                            isRequired
                            type="text"
                            id="config.anchore.username"
                            name="config.anchore.username"
                            value={values.config.anchore.username}
                            onChange={onChange}
                            isDisabled={!isEdittable}
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
                                isDisabled={!isEdittable}
                            />
                        </FormLabelGroup>
                    )}
                    {values.updatePassword && (
                        <FormLabelGroup
                            label="Password"
                            fieldId="config.anchore.password"
                            errors={errors}
                        >
                            <TextInput
                                isRequired
                                type="password"
                                id="config.anchore.password"
                                name="config.anchore.password"
                                value={values.config.anchore.password}
                                onChange={onChange}
                                isDisabled={!isEdittable}
                            />
                        </FormLabelGroup>
                    )}
                    <FormLabelGroup
                        label="Disable TLS Certificate Validation (Insecure)"
                        fieldId="config.anchore.insecure"
                        errors={errors}
                    >
                        <Switch
                            id="config.anchore.insecure"
                            name="config.anchore.insecure"
                            aria-label="disable tls certificate validation"
                            isChecked={Boolean(values.config.anchore.insecure)}
                            onChange={onChange}
                            isDisabled={!isEdittable}
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
                            isChecked={Boolean(values.config.skipTestIntegration)}
                            onChange={onChange}
                            isDisabled={!isEdittable}
                        />
                    </FormLabelGroup>
                </Form>
            </PageSection>
            {isEdittable && (
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

export default AnchoreIntegrationForm;
