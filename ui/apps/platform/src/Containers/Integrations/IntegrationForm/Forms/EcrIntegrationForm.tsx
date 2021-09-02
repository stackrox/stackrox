import React, { ReactElement } from 'react';
import { TextInput, PageSection, Form, Checkbox } from '@patternfly/react-core';
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
        name: yup.string().trim().required('An integration name is required'),
        categories: yup
            .array()
            .of(yup.string().trim().oneOf(['REGISTRY']))
            .min(1, 'Must have at least one type selected')
            .required('A category is required'),
        ecr: yup.object().shape({
            registryId: yup.string().trim().required('A registry id is required'),
            endpoint: yup.string().trim().required('An endpoint is required'),
            region: yup.string().trim().required('An AWS region is required'),
            useIam: yup.bool(),
            accessKeyId: yup.string().when('useIam', {
                is: false,
                then: yup
                    .string()
                    .test(
                        'acessKeyId-test',
                        'An access key id is required',
                        (value, context: yup.TestContext) => {
                            const requirePasswordField =
                                // eslint-disable-next-line @typescript-eslint/ban-ts-comment
                                // @ts-ignore
                                context?.from[2]?.value?.updatePassword || false;

                            if (!requirePasswordField) {
                                return true;
                            }

                            const trimmedValue = value?.trim();
                            return !!trimmedValue;
                        }
                    ),
            }),
            secretAccessKey: yup.string().when('useIam', {
                is: false,
                then: yup
                    .string()
                    .test(
                        'secretAccessKey-test',
                        'A secret access key is required',
                        (value, context: yup.TestContext) => {
                            const requirePasswordField =
                                // eslint-disable-next-line @typescript-eslint/ban-ts-comment
                                // @ts-ignore
                                context?.from[2]?.value?.updatePassword || false;

                            if (!requirePasswordField) {
                                return true;
                            }

                            const trimmedValue = value?.trim();
                            return !!trimmedValue;
                        }
                    ),
            }),
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
    } = useIntegrationForm<EcrIntegrationFormValues, typeof validationSchema>({
        initialValues: formInitialValues,
        validationSchema,
    });
    const { isCreating } = usePageState();

    function onChange(value, event) {
        return setFieldValue(event.target.id, value);
    }

    return (
        <>
            <PageSection variant="light" isFilled hasOverflowScroll>
                {message && <FormMessage message={message} />}
                <Form isWidthLimited>
                    <FormLabelGroup
                        label="Integration name"
                        isRequired
                        fieldId="config.name"
                        touched={touched}
                        errors={errors}
                    >
                        <TextInput
                            type="text"
                            id="config.name"
                            value={values.config.name}
                            onChange={onChange}
                            onBlur={handleBlur}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        label="Registry id"
                        isRequired
                        fieldId="config.ecr.registryId"
                        touched={touched}
                        errors={errors}
                    >
                        <TextInput
                            type="text"
                            id="config.ecr.registryId"
                            value={values.config.ecr.registryId}
                            onChange={onChange}
                            onBlur={handleBlur}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        label="Endpoint"
                        isRequired
                        fieldId="config.ecr.endpoint"
                        touched={touched}
                        errors={errors}
                    >
                        <TextInput
                            type="text"
                            id="config.ecr.endpoint"
                            value={values.config.ecr.endpoint}
                            onChange={onChange}
                            onBlur={handleBlur}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        label="Region"
                        isRequired
                        fieldId="config.ecr.region"
                        touched={touched}
                        errors={errors}
                    >
                        <TextInput
                            isRequired
                            type="text"
                            id="config.ecr.region"
                            value={values.config.ecr.region}
                            onChange={onChange}
                            onBlur={handleBlur}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    {!isCreating && (
                        <FormLabelGroup
                            fieldId="updatePassword"
                            helperText="Leave this off to use the currently stored credentials."
                            errors={errors}
                        >
                            <Checkbox
                                label="Update stored credentials"
                                id="updatePassword"
                                isChecked={values.updatePassword}
                                onChange={onChange}
                                onBlur={handleBlur}
                                isDisabled={!isEditable}
                            />
                        </FormLabelGroup>
                    )}
                    <FormLabelGroup fieldId="config.ecr.useIam" touched={touched} errors={errors}>
                        <Checkbox
                            label="Use container IAM role"
                            id="config.ecr.useIam"
                            aria-label="use container iam role"
                            isChecked={values.config.ecr.useIam}
                            onChange={onChange}
                            onBlur={handleBlur}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    {!values.config.ecr.useIam && (
                        <>
                            <FormLabelGroup
                                isRequired={values.updatePassword}
                                label="Access key id"
                                fieldId="config.ecr.accessKeyId"
                                touched={touched}
                                errors={errors}
                            >
                                <TextInput
                                    isRequired={values.updatePassword}
                                    type="password"
                                    id="config.ecr.accessKeyId"
                                    value={values.config.ecr.accessKeyId}
                                    onChange={onChange}
                                    onBlur={handleBlur}
                                    isDisabled={!isEditable}
                                />
                            </FormLabelGroup>
                            <FormLabelGroup
                                isRequired={values.updatePassword}
                                label="Secret access key"
                                fieldId="config.ecr.secretAccessKey"
                                touched={touched}
                                errors={errors}
                            >
                                <TextInput
                                    isRequired={values.updatePassword}
                                    type="password"
                                    id="config.ecr.secretAccessKey"
                                    value={values.config.ecr.secretAccessKey}
                                    onChange={onChange}
                                    onBlur={handleBlur}
                                    isDisabled={!isEditable}
                                />
                            </FormLabelGroup>
                        </>
                    )}
                    <FormLabelGroup
                        fieldId="config.skipTestIntegration"
                        touched={touched}
                        errors={errors}
                    >
                        <Checkbox
                            label="Create integration without testing"
                            id="config.skipTestIntegration"
                            aria-label="skip test integration"
                            isChecked={values.config.skipTestIntegration}
                            onChange={onChange}
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
                        isValid={isValid}
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
