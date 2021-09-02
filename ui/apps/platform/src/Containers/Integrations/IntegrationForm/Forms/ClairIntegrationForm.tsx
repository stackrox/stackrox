import React, { ReactElement } from 'react';
import { TextInput, SelectOption, PageSection, Form, Checkbox } from '@patternfly/react-core';
import * as yup from 'yup';

import FormMultiSelect from 'Components/FormMultiSelect';
import useIntegrationForm from '../useIntegrationForm';
import { IntegrationFormProps } from '../integrationFormTypes';

import IntegrationFormActions from '../IntegrationFormActions';
import FormCancelButton from '../FormCancelButton';
import FormTestButton from '../FormTestButton';
import FormSaveButton from '../FormSaveButton';
import FormMessage from '../FormMessage';
import FormLabelGroup from '../FormLabelGroup';

export type ClairIntegration = {
    id?: string;
    name: string;
    categories: 'SCANNER'[];
    clair: {
        endpoint: string;
        insecure: boolean;
    };
    type: 'clair';
    enabled: boolean;
    clusterIds: string[];
};

export const validationSchema = yup.object().shape({
    name: yup.string().required('An integration name is required'),
    categories: yup
        .array()
        .of(yup.string().oneOf(['SCANNER']))
        .min(1, 'Must have at least one type selected')
        .required('A category is required'),
    clair: yup.object().shape({
        endpoint: yup.string().required('An endpoint is required').min(1),
        insecure: yup.bool(),
    }),
    type: yup.string().matches(/clair/),
    enabled: yup.bool(),
    clusterIds: yup.array().of(yup.string()),
});

export const defaultValues: ClairIntegration = {
    name: '',
    categories: [],
    clair: {
        endpoint: '',
        insecure: false,
    },
    type: 'clair',
    enabled: true,
    clusterIds: [],
};

function ClairIntegrationForm({
    initialValues = null,
    isEditable = false,
}: IntegrationFormProps<ClairIntegration>): ReactElement {
    const formInitialValues = initialValues
        ? ({ ...defaultValues, ...initialValues } as ClairIntegration)
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
    } = useIntegrationForm<ClairIntegration, typeof validationSchema>({
        initialValues: formInitialValues,
        validationSchema,
    });

    function onChange(value, event) {
        return setFieldValue(event.target.id, value);
    }

    function onCustomChange(id, value) {
        return setFieldValue(id, value);
    }

    return (
        <>
            <PageSection variant="light" isFilled hasOverflowScroll>
                {message && <FormMessage message={message} />}
                <Form isWidthLimited>
                    <FormLabelGroup
                        label="Integration name"
                        isRequired
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
                        label="Type"
                        isRequired
                        fieldId="categories"
                        touched={touched}
                        errors={errors}
                    >
                        <FormMultiSelect
                            id="categories"
                            values={values.categories}
                            onChange={onCustomChange}
                            isDisabled={!isEditable}
                        >
                            <SelectOption key={0} value="SCANNER">
                                Image Scanner
                            </SelectOption>
                            <SelectOption key={1} value="NODE_SCANNER">
                                Node Scanner
                            </SelectOption>
                        </FormMultiSelect>
                    </FormLabelGroup>
                    <FormLabelGroup
                        label="Endpoint"
                        isRequired
                        fieldId="clair.endpoint"
                        touched={touched}
                        errors={errors}
                    >
                        <TextInput
                            isRequired
                            type="text"
                            id="clair.endpoint"
                            value={values.clair.endpoint}
                            onChange={onChange}
                            onBlur={handleBlur}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup fieldId="clair.insecure" touched={touched} errors={errors}>
                        <Checkbox
                            label="Disable TLS certificate validation (insecure)"
                            id="clair.insecure"
                            isChecked={values.clair.insecure}
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

export default ClairIntegrationForm;
