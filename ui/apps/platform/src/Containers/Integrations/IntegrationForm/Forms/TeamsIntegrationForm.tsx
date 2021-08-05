import React, { ReactElement } from 'react';
import { TextInput, PageSection, Form } from '@patternfly/react-core';
import * as yup from 'yup';

import useIntegrationForm from '../useIntegrationForm';
import { IntegrationFormProps } from '../integrationFormTypes';

import IntegrationFormActions from '../IntegrationFormActions';
import FormCancelButton from '../FormCancelButton';
import FormTestButton from '../FormTestButton';
import FormSaveButton from '../FormSaveButton';
import FormMessage from '../FormMessage';
import FormLabelGroup from '../FormLabelGroup';
import AnnotationKeyLabelIcon from '../AnnotationKeyLabelIcon';

export type TeamsIntegration = {
    id?: string;
    name: string;
    categories: string[];
    labelDefault: string;
    labelKey: string;
    uiEndpoint: string;
    type: 'teams';
    enabled: boolean;
    clusterIds: string[];
};

export const validationSchema = yup.object().shape({
    name: yup.string().required('Required'),
    categories: yup.array().of(yup.string()),
    labelDefault: yup.string().required('Required'),
    labelKey: yup.string().required('Required'),
    uiEndpoint: yup.string(),
    type: yup.string().matches(/teams/),
    enabled: yup.bool(),
    clusterIds: yup.array().of(yup.string()),
});

export const defaultValues: TeamsIntegration = {
    name: '',
    categories: [],
    labelDefault: '',
    labelKey: '',
    uiEndpoint: window.location.origin,
    type: 'teams',
    enabled: true,
    clusterIds: [],
};

function TeamsIntegrationForm({
    initialValues = null,
    isEditable = false,
}: IntegrationFormProps<TeamsIntegration>): ReactElement {
    const formInitialValues = initialValues
        ? { ...defaultValues, ...initialValues }
        : defaultValues;
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
    } = useIntegrationForm<TeamsIntegration, typeof validationSchema>({
        initialValues: formInitialValues,
        validationSchema,
    });

    function onChange(value, event) {
        return setFieldValue(event.target.id, value, false);
    }

    return (
        <>
            <PageSection variant="light" isFilled hasOverflowScroll>
                {message && <FormMessage message={message} />}
                <Form isWidthLimited>
                    <FormLabelGroup label="Name" isRequired fieldId="name" errors={errors}>
                        <TextInput
                            isRequired
                            type="text"
                            id="name"
                            name="name"
                            value={values.name}
                            placeholder="(ex. Teams Integration)"
                            onChange={onChange}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        label="Default Teams Webhook"
                        isRequired
                        fieldId="labelDefault"
                        errors={errors}
                    >
                        <TextInput
                            isRequired
                            type="text"
                            id="labelDefault"
                            name="labelDefault"
                            value={values.labelDefault}
                            placeholder="(ex. https://outlook.office365.com/webhook/EXAMPLE)"
                            onChange={onChange}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        label="Annotation key for recipient"
                        labelIcon={<AnnotationKeyLabelIcon />}
                        fieldId="labelKey"
                        errors={errors}
                    >
                        <TextInput
                            type="text"
                            id="labelKey"
                            value={values.labelKey}
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

export default TeamsIntegrationForm;
