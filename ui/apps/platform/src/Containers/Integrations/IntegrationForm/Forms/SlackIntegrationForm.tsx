import React, { ReactElement } from 'react';
import { Form, PageSection, TextInput } from '@patternfly/react-core';
import * as yup from 'yup';
import merge from 'lodash/merge';

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

export const validationSchema = yup.object().shape({
    name: yup.string().trim().required('Name is required'),
    labelDefault: yup
        .string()
        .trim()
        .required('Webhook is required, like https://hooks.slack.com/services/EXAMPLE'),
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
    const formInitialValues: SlackIntegration = merge({}, defaultValues, initialValues);
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
                            onChange={(event, value) => onChange(value, event)}
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
                        helperText="For example, https://hooks.slack.com/services/EXAMPLE"
                    >
                        <TextInput
                            isRequired
                            type="text"
                            id="labelDefault"
                            value={values.labelDefault}
                            onChange={(event, value) => onChange(value, event)}
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
                            onChange={(event, value) => onChange(value, event)}
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
