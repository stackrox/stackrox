/* eslint-disable react/no-array-index-key */
import React, { ReactElement } from 'react';
import {
    Button,
    Checkbox,
    Flex,
    FlexItem,
    Form,
    FormSection,
    PageSection,
    TextArea,
    TextInput,
} from '@patternfly/react-core';
import { PlusCircleIcon, TrashIcon } from '@patternfly/react-icons';
import * as yup from 'yup';
import { FieldArray, FormikProvider } from 'formik';

import usePageState from 'Containers/Integrations/hooks/usePageState';
import FormMessage from 'Components/PatternFly/FormMessage';
import FormTestButton from 'Components/PatternFly/FormTestButton';
import FormSaveButton from 'Components/PatternFly/FormSaveButton';
import FormCancelButton from 'Components/PatternFly/FormCancelButton';
import useIntegrationForm from '../useIntegrationForm';
import { IntegrationFormProps } from '../integrationFormTypes';

import IntegrationFormActions from '../IntegrationFormActions';
import FormLabelGroup from '../FormLabelGroup';

export type ScopedAccessPluginIntegration = {
    id: string;
    name: string;
    endpointConfig: {
        endpoint: string;
        skipTlsVerify: boolean;
        caCert: string;
        username: string;
        password: string;
        headers: {
            key: string;
            value: string;
        }[];
        clientCertPem: string;
        clientKeyPem: string;
    };
    uiEndpoint: string;
    type: 'scopedAccess';
    enabled: boolean;
};

export type ScopedAccessPluginIntegrationFormValues = {
    config: ScopedAccessPluginIntegration;
    updatePassword: boolean;
};

export const validationSchema = yup.object().shape({
    config: yup.object().shape({
        name: yup.string().trim().required('Integration name is required'),
        enable: yup.bool(),
        endpointConfig: yup.object().shape({
            endpoint: yup.string().trim().required('Endpoint is required'),
            skipTlsVerify: yup.bool(),
            username: yup
                .string()
                .test(
                    'username-test',
                    'A username is required if the integration has a password',
                    (value, context: yup.TestContext) => {
                        const hasPassword = !!context.parent.password;
                        if (!hasPassword) {
                            return true;
                        }
                        const trimmedValue = value?.trim();
                        return !!trimmedValue;
                    }
                ),
            password: yup
                .string()
                .test(
                    'password-test',
                    'A password is required if the integration has a username',
                    (value, context: yup.TestContext) => {
                        const requirePasswordField =
                            // eslint-disable-next-line @typescript-eslint/ban-ts-comment
                            // @ts-ignore
                            context?.from[2]?.value?.updatePassword || false;
                        const hasUsername = !!context.parent.username;

                        if (!requirePasswordField || !hasUsername) {
                            return true;
                        }

                        const trimmedValue = value?.trim();
                        return !!trimmedValue;
                    }
                ),
            caCert: yup.string(),
            clientCertPem: yup.string(),
            clientKeyPem: yup.string(),
        }),
    }),
    updatePassword: yup.bool(),
});

export const defaultValues: ScopedAccessPluginIntegrationFormValues = {
    config: {
        id: '',
        name: '',
        enabled: false,
        endpointConfig: {
            endpoint: '',
            skipTlsVerify: false,
            caCert: '',
            username: '',
            password: '',
            headers: [],
            clientCertPem: '',
            clientKeyPem: '',
        },
        uiEndpoint: window.location.origin,
        type: 'scopedAccess',
    },
    updatePassword: true,
};

function ScopedAccessPluginIntegrationForm({
    initialValues = null,
    isEditable = false,
}: IntegrationFormProps<ScopedAccessPluginIntegration>): ReactElement {
    const formInitialValues = { ...defaultValues, ...initialValues };
    if (initialValues) {
        formInitialValues.config = {
            ...formInitialValues.config,
            ...initialValues,
        };
        // We want to clear the password because backend returns '******' to represent that there
        // are currently stored credentials
        formInitialValues.config.endpointConfig.password = '';
    }
    const formik = useIntegrationForm<ScopedAccessPluginIntegrationFormValues>({
        initialValues: formInitialValues,
        validationSchema,
    });
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
    } = formik;
    const { isCreating } = usePageState();

    function onChange(value, event) {
        return setFieldValue(event.target.id, value);
    }

    function onUpdateCredentialsChange(value, event) {
        setFieldValue('config.endpointConfig.password', '');
        return setFieldValue(event.target.id, value);
    }

    return (
        <>
            <PageSection variant="light" isFilled hasOverflowScroll>
                <FormMessage message={message} />
                <Form isWidthLimited>
                    <FormikProvider value={formik}>
                        <FormLabelGroup
                            isRequired
                            label="Integration name"
                            fieldId="config.name"
                            touched={touched}
                            errors={errors}
                        >
                            <TextInput
                                isRequired
                                type="text"
                                id="config.name"
                                value={values.config.name}
                                onChange={onChange}
                                onBlur={handleBlur}
                                isDisabled={!isEditable}
                            />
                        </FormLabelGroup>
                        <FormLabelGroup
                            isRequired
                            label="Endpoint"
                            fieldId="config.endpointConfig.endpoint"
                            touched={touched}
                            errors={errors}
                        >
                            <TextInput
                                isRequired
                                type="text"
                                id="config.endpointConfig.endpoint"
                                value={values.config.endpointConfig.endpoint}
                                onChange={onChange}
                                onBlur={handleBlur}
                                isDisabled={!isEditable}
                            />
                        </FormLabelGroup>
                        <FormLabelGroup label="" fieldId="config.enabled" errors={errors}>
                            <Checkbox
                                label="Enabled"
                                id="config.enabled"
                                isChecked={values.config.enabled}
                                onChange={onChange}
                                onBlur={handleBlur}
                                isDisabled={!isEditable}
                            />
                        </FormLabelGroup>
                        <FormLabelGroup
                            label=""
                            fieldId="config.endpointConfig.skipTlsVerify"
                            errors={errors}
                        >
                            <Checkbox
                                label="Skip TLS verification"
                                id="config.endpointConfig.skipTlsVerify"
                                isChecked={values.config.endpointConfig.skipTlsVerify}
                                onChange={onChange}
                                onBlur={handleBlur}
                                isDisabled={!isEditable}
                            />
                        </FormLabelGroup>
                        <FormLabelGroup
                            label="CA certificate (optional)"
                            fieldId="config.endpointConfig.caCert"
                            touched={touched}
                            errors={errors}
                        >
                            <TextArea
                                className="json-input"
                                type="text"
                                id="config.endpointConfig.caCert"
                                value={values.config.endpointConfig.caCert}
                                onChange={onChange}
                                onBlur={handleBlur}
                                isDisabled={!isEditable}
                            />
                        </FormLabelGroup>
                        <FormLabelGroup
                            label={`Username${
                                values.config.endpointConfig.password ? '' : ' (optional)'
                            }`}
                            isRequired={!!values.config.endpointConfig.password}
                            fieldId="config.endpointConfig.username"
                            touched={touched}
                            errors={errors}
                        >
                            <TextInput
                                isRequired={!!values.config.endpointConfig.password}
                                type="text"
                                id="config.endpointConfig.username"
                                value={values.config.endpointConfig.username}
                                placeholder="example, postmaster@example.com"
                                onChange={onChange}
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
                                    label="Update password"
                                    id="updatePassword"
                                    isChecked={values.updatePassword}
                                    onChange={onUpdateCredentialsChange}
                                    onBlur={handleBlur}
                                    isDisabled={!isEditable}
                                />
                            </FormLabelGroup>
                        )}
                        <FormLabelGroup
                            label={`Password${
                                values.config.endpointConfig.username || values.updatePassword
                                    ? ''
                                    : ' (optional)'
                            }`}
                            isRequired={
                                values.updatePassword && !!values.config.endpointConfig.username
                            }
                            fieldId="config.endpointConfig.password"
                            touched={touched}
                            errors={errors}
                        >
                            <TextInput
                                isRequired={
                                    values.updatePassword && !!values.config.endpointConfig.username
                                }
                                type="password"
                                id="config.endpointConfig.password"
                                value={values.config.endpointConfig.password}
                                onChange={onChange}
                                onBlur={handleBlur}
                                isDisabled={!isEditable || !values.updatePassword}
                                placeholder={
                                    values.updatePassword
                                        ? ''
                                        : 'Currently-stored password will be used.'
                                }
                            />
                        </FormLabelGroup>
                        <FormSection title="Headers" titleElement="h3" className="pf-u-mt-0">
                            <FieldArray
                                name="config.endpointConfig.headers"
                                render={(arrayHelpers) => (
                                    <>
                                        {values.config.endpointConfig.headers.length === 0 && (
                                            <p>No custom headers defined</p>
                                        )}
                                        {values.config.endpointConfig.headers.length > 0 &&
                                            values.config.endpointConfig.headers.map(
                                                (_header, index: number) => (
                                                    <Flex key={`header_${index}`}>
                                                        <FlexItem>
                                                            <FormLabelGroup
                                                                label="Key"
                                                                fieldId={`config.endpointConfig.headers[${index}].key`}
                                                                touched={touched}
                                                                errors={errors}
                                                            >
                                                                <TextInput
                                                                    isRequired
                                                                    type="text"
                                                                    id={`config.endpointConfig.headers[${index}].key`}
                                                                    value={
                                                                        values.config.endpointConfig
                                                                            .headers[`${index}`].key
                                                                    }
                                                                    onChange={onChange}
                                                                    onBlur={handleBlur}
                                                                    isDisabled={!isEditable}
                                                                />
                                                            </FormLabelGroup>
                                                        </FlexItem>
                                                        <FlexItem>
                                                            <FormLabelGroup
                                                                label="Value"
                                                                fieldId={`config.endpointConfig.headers[${index}].value`}
                                                                touched={touched}
                                                                errors={errors}
                                                            >
                                                                <TextInput
                                                                    isRequired
                                                                    type="text"
                                                                    id={`config.endpointConfig.headers[${index}].value`}
                                                                    value={
                                                                        values.config.endpointConfig
                                                                            .headers[`${index}`]
                                                                            .value
                                                                    }
                                                                    onChange={onChange}
                                                                    onBlur={handleBlur}
                                                                    isDisabled={!isEditable}
                                                                />
                                                            </FormLabelGroup>
                                                        </FlexItem>
                                                        {isEditable && (
                                                            <FlexItem>
                                                                <Button
                                                                    variant="plain"
                                                                    aria-label="Delete header key/value pair"
                                                                    style={{
                                                                        transform:
                                                                            'translate(0, 42px)',
                                                                    }}
                                                                    onClick={() =>
                                                                        arrayHelpers.remove(index)
                                                                    }
                                                                >
                                                                    <TrashIcon />
                                                                </Button>
                                                            </FlexItem>
                                                        )}
                                                    </Flex>
                                                )
                                            )}
                                        {isEditable && (
                                            <Flex>
                                                <FlexItem>
                                                    <Button
                                                        variant="link"
                                                        isInline
                                                        icon={
                                                            <PlusCircleIcon className="pf-u-mr-sm" />
                                                        }
                                                        onClick={() =>
                                                            arrayHelpers.push({
                                                                key: '',
                                                                value: '',
                                                            })
                                                        }
                                                    >
                                                        Add new header
                                                    </Button>
                                                </FlexItem>
                                            </Flex>
                                        )}
                                    </>
                                )}
                            />
                        </FormSection>
                        <FormLabelGroup
                            label="Client certificate (optional)"
                            fieldId="config.endpointConfig.clientCertPem"
                            touched={touched}
                            errors={errors}
                        >
                            <TextArea
                                className="json-input"
                                type="text"
                                id="config.endpointConfig.clientCertPem"
                                value={values.config.endpointConfig.clientCertPem}
                                // eslint-disable-next-line prettier/prettier
                                placeholder={"example,\n-----BEGIN CERTIFICATE-----\nPEM-encoded client certificate\n-----END CERTIFICATE-----"}
                                onChange={onChange}
                                onBlur={handleBlur}
                                isDisabled={!isEditable}
                            />
                        </FormLabelGroup>
                        <FormLabelGroup
                            label="Client key (optional)"
                            fieldId="config.endpointConfig.clientKeyPem"
                            touched={touched}
                            errors={errors}
                        >
                            <TextArea
                                className="json-input"
                                type="text"
                                id="config.endpointConfig.clientKeyPem"
                                value={values.config.endpointConfig.clientKeyPem}
                                // eslint-disable-next-line prettier/prettier
                                placeholder={"example,\n-----BEGIN CERTIFICATE-----\nPEM-encoded private key\n-----END CERTIFICATE-----"}
                                onChange={onChange}
                                onBlur={handleBlur}
                                isDisabled={!isEditable}
                            />
                        </FormLabelGroup>
                    </FormikProvider>
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

export default ScopedAccessPluginIntegrationForm;
