import React, { ReactElement } from 'react';
import { TextInput, SelectOption, PageSection, Form, Switch } from '@patternfly/react-core';
import * as yup from 'yup';

import FormMultiSelect from 'Components/FormMultiSelect';
import useIntegrationForm from '../useIntegrationForm';
import { IntegrationFormProps } from '../integrationFormTypes';

import IntegrationFormActions from '../IntegrationFormActions';
import FormCancelButton from '../FormCancelButton';
import FormTestButton from '../FormTestButton';
import FormSaveButton from '../FormSaveButton';
import FormMessageBanner from '../FormMessageBanner';
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
    name: yup.string().required('Required'),
    categories: yup
        .array()
        .of(yup.string().oneOf(['SCANNER']))
        .min(1, 'Must have at least one type selected')
        .required('Required'),
    clair: yup.object().shape({
        endpoint: yup.string().required('Required').min(1),
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
        errors,
        setFieldValue,
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
                    <FormLabelGroup label="Name" isRequired fieldId="name" errors={errors}>
                        <TextInput
                            isRequired
                            type="text"
                            id="name"
                            name="name"
                            value={values.name}
                            onChange={onChange}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup label="Type" isRequired fieldId="categories" errors={errors}>
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
                        errors={errors}
                    >
                        <TextInput
                            isRequired
                            type="text"
                            id="clair.endpoint"
                            name="clair.endpoint"
                            value={values.clair.endpoint}
                            onChange={onChange}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup label="Insecure" fieldId="clair.insecure" errors={errors}>
                        <Switch
                            id="clair.insecure"
                            name="clair.insecure"
                            aria-label="insecure"
                            isChecked={values.clair.insecure}
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

export default ClairIntegrationForm;
