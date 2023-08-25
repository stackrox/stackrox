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
import { GenericNotifierIntegration as GenericWebhookIntegration } from 'types/notifier.proto';
import useIntegrationForm from '../useIntegrationForm';
import { IntegrationFormProps } from '../integrationFormTypes';

import IntegrationFormActions from '../IntegrationFormActions';
import FormLabelGroup from '../FormLabelGroup';

export type GenericWebhookIntegrationFormValues = {
    notifier: GenericWebhookIntegration;
    updatePassword: boolean;
};

export const validationSchema = yup.object().shape({
    notifier: yup.object().shape({
        name: yup.string().trim().required('Name is required'),
        generic: yup.object().shape({
            endpoint: yup.string().trim().required('Endpoint is required'),
            skipTlsVerify: yup.bool(),
            auditLoggingEnabled: yup.bool(),
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
        }),
    }),
    updatePassword: yup.bool(),
});

export const defaultValues: GenericWebhookIntegrationFormValues = {
    notifier: {
        id: '',
        name: '',
        generic: {
            endpoint: '',
            skipTLSVerify: false,
            auditLoggingEnabled: false,
            caCert: '',
            username: '',
            password: '',
            headers: [],
            extraFields: [],
        },
        labelDefault: '',
        labelKey: '',
        uiEndpoint: window.location.origin,
        type: 'generic',
    },
    updatePassword: true,
};

function GenericWebhookIntegrationForm({
    initialValues = null,
    isEditable = false,
}: IntegrationFormProps<GenericWebhookIntegration>): ReactElement {
    const formInitialValues = { ...defaultValues, ...initialValues };
    if (initialValues) {
        formInitialValues.notifier = {
            ...formInitialValues.notifier,
            ...initialValues,
        };
        // We want to clear the password because backend returns '******' to represent that there
        // are currently stored credentials
        formInitialValues.notifier.generic.password = '';

        // Don't assume user wants to change password; that has caused confusing UX.
        formInitialValues.updatePassword = false;
    }
    const formik = useIntegrationForm<GenericWebhookIntegrationFormValues>({
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
        setFieldValue('notifier.generic.password', '');
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
                            fieldId="notifier.name"
                            touched={touched}
                            errors={errors}
                        >
                            <TextInput
                                isRequired
                                type="text"
                                id="notifier.name"
                                value={values.notifier.name}
                                onChange={onChange}
                                onBlur={handleBlur}
                                isDisabled={!isEditable}
                            />
                        </FormLabelGroup>
                        <FormLabelGroup
                            isRequired
                            label="Endpoint"
                            fieldId="notifier.generic.endpoint"
                            touched={touched}
                            errors={errors}
                        >
                            <TextInput
                                isRequired
                                type="text"
                                id="notifier.generic.endpoint"
                                value={values.notifier.generic.endpoint}
                                onChange={onChange}
                                onBlur={handleBlur}
                                isDisabled={!isEditable}
                            />
                        </FormLabelGroup>
                        <FormLabelGroup
                            label=""
                            fieldId="notifier.generic.skipTLSVerify"
                            errors={errors}
                        >
                            <Checkbox
                                label="Skip TLS verification"
                                id="notifier.generic.skipTLSVerify"
                                isChecked={values.notifier.generic.skipTLSVerify}
                                onChange={onChange}
                                onBlur={handleBlur}
                                isDisabled={!isEditable}
                            />
                        </FormLabelGroup>
                        <FormLabelGroup
                            label=""
                            fieldId="notifier.generic.auditLoggingEnabled"
                            errors={errors}
                        >
                            <Checkbox
                                label="Enable audit logging"
                                id="notifier.generic.auditLoggingEnabled"
                                isChecked={values.notifier.generic.auditLoggingEnabled}
                                onChange={onChange}
                                onBlur={handleBlur}
                                isDisabled={!isEditable}
                            />
                        </FormLabelGroup>
                        <FormLabelGroup
                            label="CA certificate (optional)"
                            fieldId="notifier.generic.caCert"
                            touched={touched}
                            errors={errors}
                        >
                            <TextArea
                                className="json-input"
                                type="text"
                                id="notifier.generic.caCert"
                                value={values.notifier.generic.caCert}
                                onChange={onChange}
                                onBlur={handleBlur}
                                isDisabled={!isEditable}
                            />
                        </FormLabelGroup>
                        <FormLabelGroup
                            label={`Username${
                                values.notifier.generic.password ? '' : ' (optional)'
                            }`}
                            isRequired={!!values.notifier.generic.password}
                            fieldId="notifier.generic.username"
                            touched={touched}
                            errors={errors}
                        >
                            <TextInput
                                isRequired={!!values.notifier.generic.password}
                                type="text"
                                id="notifier.generic.username"
                                value={values.notifier.generic.username}
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
                                values.notifier.generic.username || values.updatePassword
                                    ? ''
                                    : ' (optional)'
                            }`}
                            isRequired={values.updatePassword && !!values.notifier.generic.username}
                            fieldId="notifier.generic.password"
                            touched={touched}
                            errors={errors}
                        >
                            <TextInput
                                isRequired={
                                    values.updatePassword && !!values.notifier.generic.username
                                }
                                type="password"
                                id="notifier.generic.password"
                                value={values.notifier.generic.password}
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
                                name="notifier.generic.headers"
                                render={(arrayHelpers) => (
                                    <>
                                        {values.notifier.generic.headers.length === 0 && (
                                            <p>No custom headers defined</p>
                                        )}
                                        {values.notifier.generic.headers.length > 0 &&
                                            values.notifier.generic.headers.map(
                                                (_header, index: number) => (
                                                    <Flex key={`header_${index}`}>
                                                        <FlexItem>
                                                            <FormLabelGroup
                                                                label="Key"
                                                                fieldId={`notifier.generic.headers[${index}].key`}
                                                                touched={touched}
                                                                errors={errors}
                                                            >
                                                                <TextInput
                                                                    isRequired
                                                                    type="text"
                                                                    id={`notifier.generic.headers[${index}].key`}
                                                                    value={
                                                                        values.notifier.generic
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
                                                                fieldId={`notifier.generic.headers[${index}].value`}
                                                                touched={touched}
                                                                errors={errors}
                                                            >
                                                                <TextInput
                                                                    isRequired
                                                                    type="text"
                                                                    id={`notifier.generic.headers[${index}].value`}
                                                                    value={
                                                                        values.notifier.generic
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
                        <FormSection title="Extra Fields" titleElement="h3" className="pf-u-mt-0">
                            <FieldArray
                                name="notifier.generic.extraFields"
                                render={(arrayHelpers) => (
                                    <>
                                        {values.notifier.generic.extraFields.length === 0 && (
                                            <p>No custom extra fields defined</p>
                                        )}
                                        {values.notifier.generic.extraFields.length > 0 &&
                                            values.notifier.generic.extraFields.map(
                                                (_extraField, index: number) => (
                                                    <Flex key={`extraField_${index}`}>
                                                        <FlexItem>
                                                            <FormLabelGroup
                                                                label="Key"
                                                                fieldId={`notifier.generic.extraFields[${index}].key`}
                                                                touched={touched}
                                                                errors={errors}
                                                            >
                                                                <TextInput
                                                                    isRequired
                                                                    type="text"
                                                                    id={`notifier.generic.extraFields[${index}].key`}
                                                                    value={
                                                                        values.notifier.generic
                                                                            .extraFields[`${index}`]
                                                                            .key
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
                                                                fieldId={`notifier.generic.extraFields[${index}].value`}
                                                                touched={touched}
                                                                errors={errors}
                                                            >
                                                                <TextInput
                                                                    isRequired
                                                                    type="text"
                                                                    id={`notifier.generic.extraFields[${index}].value`}
                                                                    value={
                                                                        values.notifier.generic
                                                                            .extraFields[`${index}`]
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
                                                                    aria-label="Delete extra field key/value pair"
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
                                                        Add new extra field
                                                    </Button>
                                                </FlexItem>
                                            </Flex>
                                        )}
                                    </>
                                )}
                            />
                        </FormSection>
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

export default GenericWebhookIntegrationForm;
