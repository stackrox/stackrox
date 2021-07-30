import React, { ReactElement } from 'react';
import { TextInput, SelectOption, PageSection, Form } from '@patternfly/react-core';
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

export type ClairifyIntegration = {
    id?: string;
    name: string;
    categories: ('NODE_SCANNER' | 'SCANNER')[];
    clairify: {
        endpoint: string;
        grpcEndpoint: string;
        numConcurrentScans: string;
    };
    type: 'clairify';
    enabled: boolean;
    clusterIds: string[];
};

export const validationSchema = yup.object().shape({
    name: yup.string().required('Required'),
    categories: yup
        .array()
        .of(yup.string().oneOf(['NODE_SCANNER', 'SCANNER']))
        .min(1, 'Must have at least one type selected')
        .required('Required'),
    clairify: yup.object().shape({
        endpoint: yup.string().required('Required').min(1),
        grpcEndpoint: yup.string(),
        numConcurrentScans: yup.string(),
    }),
    type: yup.string().matches(/clairify/),
    enabled: yup.bool(),
    clusterIds: yup.array().of(yup.string()),
});

export const defaultValues: ClairifyIntegration = {
    name: '',
    categories: [],
    clairify: {
        endpoint: '',
        grpcEndpoint: '',
        numConcurrentScans: '0',
    },
    type: 'clairify',
    enabled: true,
    clusterIds: [],
};

function ClairifyIntegrationForm({
    initialValues = null,
    isEditable = false,
}: IntegrationFormProps<ClairifyIntegration>): ReactElement {
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
    } = useIntegrationForm<ClairifyIntegration, typeof validationSchema>({
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
                        fieldId="clairify.endpoint"
                        errors={errors}
                    >
                        <TextInput
                            isRequired
                            type="text"
                            id="clairify.endpoint"
                            name="clairify.endpoint"
                            value={values.clairify.endpoint}
                            onChange={onChange}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        label="GRPC Endpoint"
                        fieldId="clairify.grpcEndpoint"
                        helperText="Used For Node Scanning"
                        errors={errors}
                    >
                        <TextInput
                            isRequired
                            type="text"
                            id="clairify.grpcEndpoint"
                            name="clairify.grpcEndpoint"
                            value={values.clairify.grpcEndpoint}
                            onChange={onChange}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        label="Max Concurrent Image Scans"
                        fieldId="clairify.numConcurrentScans"
                        helperText="0 for default"
                        errors={errors}
                    >
                        <TextInput
                            isRequired
                            type="number"
                            id="clairify.numConcurrentScans"
                            name="clairify.numConcurrentScans"
                            value={values.clairify.numConcurrentScans}
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

export default ClairifyIntegrationForm;
