import * as yup from 'yup';
import React, { ReactElement } from 'react';
import {Checkbox, Form, PageSection, TextArea, TextInput} from '@patternfly/react-core';
import { IntegrationFormProps } from '../integrationFormTypes';
import useIntegrationForm from '../useIntegrationForm';
import FormMessage from '../../../../Components/PatternFly/FormMessage';
import FormLabelGroup from '../FormLabelGroup';
import IntegrationFormActions from '../IntegrationFormActions';
import FormSaveButton from '../../../../Components/PatternFly/FormSaveButton';
import FormCancelButton from '../../../../Components/PatternFly/FormCancelButton';
import { CloudSourceIntegration } from '../../../../services/CloudSourceService';
import usePageState from "../../hooks/usePageState";

export const validationSchema = yup.object().shape({
    cloudSource: yup.object().shape({
        name: yup.string().trim().required('Integration name is required'),
        type: yup.string().matches(/TYPE_PALADIN_CLOUD/),
        credentials: yup.object().shape({
            secret: yup
                .string()
                .test(
                    'secret-test',
                    'A token is required',
                    (value, context: yup.TestContext) => {
                        const requireSecretField =
                            // eslint-disable-next-line @typescript-eslint/ban-ts-comment
                            // @ts-ignore
                            context?.from[2]?.value?.updateCredentials || false;

                        if (!requireSecretField) {
                            return true;
                        }

                        const trimmedValue = value?.trim();
                        return !!trimmedValue;
                    }
                ),
        }),
        paladinCloud: yup.object().shape({
            endpoint: yup.string().required('Endpoint is required'),
        }),
        skipTestIntegration: yup.bool(),
    }),
    updatePassword: yup.bool(),
});

export type CloudSourceIntegrationFormValues = {
    cloudSource: CloudSourceIntegration;
    updateCredentials: boolean;
}
export const defaultValues: CloudSourceIntegrationFormValues = {
    cloudSource: {
        id: '',
        name: '',
        type: 'TYPE_PALADIN_CLOUD',
        credentials: {
            secret: '',
        },
        skipTestIntegration: true,
        paladinCloud: {
            endpoint: 'https://api.paladincloud.io',
        },
    },
    updateCredentials: true,
};


function PaladinCloudIntegrationForm({
    initialValues = null,
    isEditable = false,
}: IntegrationFormProps<CloudSourceIntegration>): ReactElement {
    const formInitialValues = { ...defaultValues, ...initialValues };
    if (initialValues) {
        formInitialValues.cloudSource = { ...formInitialValues.cloudSource, ...initialValues };
        formInitialValues.cloudSource.credentials.secret = '';
        formInitialValues.updateCredentials = false;
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
        onCancel,
        message,
    } = useIntegrationForm<CloudSourceIntegrationFormValues>({
        initialValues: formInitialValues,
        validationSchema,
    });

    const { isCreating } = usePageState();

    function onChange(value, event) {
        return setFieldValue(event.target.id, value);
    }

    function onUpdateCredentialsChange(value, event) {
        setFieldValue('cloudSource.credentials.secret', '');
        return setFieldValue(event.target.id, value);
    }

    return (
        <>
            <PageSection variant="light" isFilled hasOverflowScroll>
                <FormMessage message={message} />
                <Form isWidthLimited>
                    <FormLabelGroup
                        isRequired
                        label="Integration name"
                        fieldId="cloudSource.name"
                        touched={touched}
                        errors={errors}
                    >
                        <TextInput
                            isRequired
                            type="text"
                            id="cloudSource.name"
                            value={values.cloudSource.name}
                            onChange={onChange}
                            onBlur={handleBlur}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        isRequired
                        label="Paladin Cloud endpoint"
                        fieldId="cloudSource.paladinCloud.endpoint"
                        touched={touched}
                        errors={errors}
                    >
                        <TextInput
                            isRequired
                            type="text"
                            id="cloudSource.paladinCloud.endpoint"
                            name="cloudSource.paladinCloud.endpoint"
                            value={values.cloudSource.paladinCloud?.endpoint}
                            onChange={onChange}
                            onBlur={handleBlur}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    {!isCreating && isEditable && (
                        <FormLabelGroup
                            fieldId="updateCredentials"
                            helperText="Enable this option to replace currently stored credentials (if any)"
                            errors={errors}
                        >
                            <Checkbox
                                label="Update stored credentials"
                                id="updateCredentials"
                                isChecked={values.updateCredentials}
                                onChange={onUpdateCredentialsChange}
                                onBlur={handleBlur}
                                isDisabled={!isEditable}
                            />
                        </FormLabelGroup>
                    )}
                    <FormLabelGroup
                        isRequired={values.updateCredentials}
                        label="Paladin Cloud token"
                        fieldId="cloudSource.credentials.secret"
                        touched={touched}
                        errors={errors}
                    >
                        <TextArea
                            isRequired={values.updateCredentials}
                            autoResize
                            resizeOrientation="vertical"
                            type="text"
                            id={`cloudSource.credentials.secret`}
                            value={values.cloudSource.credentials.secret}
                            onChange={onChange}
                            onBlur={handleBlur}
                            isDisabled={!isEditable || !values.updateCredentials}
                            placeholder={
                                values.updateCredentials
                                    ? ''
                                    : 'Currently-stored token will be used.'
                            }
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
                    <FormCancelButton onCancel={onCancel}>Cancel</FormCancelButton>
                </IntegrationFormActions>
            )}
        </>
    );
}

export default PaladinCloudIntegrationForm;
