import React, { ReactElement } from 'react';
import { Form, PageSection, TextInput } from '@patternfly/react-core';
import * as yup from 'yup';

import { NotifierIntegrationBase } from 'services/NotifierIntegrationsService';

import FormMessage from 'Components/PatternFly/FormMessage';
import FormTestButton from 'Components/PatternFly/FormTestButton';
import FormSaveButton from 'Components/PatternFly/FormSaveButton';
import FormCancelButton from 'Components/PatternFly/FormCancelButton';
import useIntegrationForm from '../useIntegrationForm';
import { IntegrationFormProps } from '../integrationFormTypes';

import IntegrationFormActions from '../IntegrationFormActions';
import FormLabelGroup from '../FormLabelGroup';
import AnnotationKeyLabelIcon from '../AnnotationKeyLabelIcon';

export type SlackIntegration = {
    type: 'slack';
} & NotifierIntegrationBase;

const validWebhookRegex =
    /^((https?):\/\/)?([a-zA-Z0-9\-.]\.)?[a-zA-Z0-9\-.]{1,}\.[a-zA-Z]{2,}(\.[a-zA-Z]{2,})?(\/services)(\/[a-zA-Z0-9-]+)+$/;

export const validationSchema = yup.object().shape({
    name: yup.string().trim().required('Name is required'),
    labelDefault: yup
        .string()
        .trim()
        .required('Slack webhook is required')
        .matches(
            validWebhookRegex,
            'Must be a valid Slack webhook URL, like https://hooks.slack.com/services/EXAMPLE'
        ),
    labelKey: yup.string().trim(),
});

export const defaultValues: SlackIntegration = {
    id: '',
    name: '',
    labelDefault: '',
    labelKey: '',
    uiEndpoint: window.location.origin,
    type: 'slack',
};

function SlackIntegrationForm({
    initialValues = null,
    isEditable = false,
}: IntegrationFormProps<SlackIntegration>): ReactElement {
    const formInitialValues = initialValues
        ? ({ ...defaultValues, ...initialValues } as SlackIntegration)
        : defaultValues;
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
    } = useIntegrationForm<SlackIntegration>({
        initialValues: formInitialValues,
        validationSchema,
    });

    function onChange(value, event) {
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
                        fieldId="name"
                        touched={touched}
                        errors={errors}
                    >
                        <TextInput
                            isRequired
                            type="text"
                            id="name"
                            value={values.name}
                            onChange={onChange}
                            onBlur={handleBlur}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        isRequired
                        label="Default Slack webhook"
                        fieldId="labelDefault"
                        touched={touched}
                        errors={errors}
                        helperText="https://hooks.slack.com/services/EXAMPLE"
                    >
                        <TextInput
                            isRequired
                            type="text"
                            id="labelDefault"
                            value={values.labelDefault}
                            onChange={onChange}
                            onBlur={handleBlur}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        label="Annotation key for Slack webhook"
                        labelIcon={<AnnotationKeyLabelIcon />}
                        fieldId="labelKey"
                        touched={touched}
                        errors={errors}
                    >
                        <TextInput
                            type="text"
                            id="labelKey"
                            value={values.labelKey}
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

export default SlackIntegrationForm;
