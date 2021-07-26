import React, { ReactElement } from 'react';
import { FormGroup, TextInput, SelectOption, PageSection, Form } from '@patternfly/react-core';
import * as yup from 'yup';

import FormMultiSelect from 'Components/FormMultiSelect';
import { Integration } from 'Containers/Integrations/utils/integrationUtils';
import useIntegrationForm from '../useIntegrationForm';

import IntegrationFormToolBar from '../IntegrationFormToolBar';
import FormCancelButton from '../FormCancelButton';
import FormTestButton from '../FormTestButton';
import FormSaveButton from '../FormSaveButton';
import FormMessageBanner from '../FormMessageBanner';

export type ClairifyIntegration = {
    name: string;
    categories: string[];
    clairify: {
        endpoint: string;
        grpcEndpoint: string;
        numConcurrentScans: string;
    };
    uiEndpoint: string;
    type: string;
    enabled: boolean;
    clusterIds: string[];
};

export type ClairifyIntegrationProps = {
    initialValues?: Integration | null;
    isEdittable?: boolean;
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
    uiEndpoint: yup.string(),
    type: yup.string(),
    enabled: yup.bool(),
    clusterIds: yup.array().of(yup.string()),
});

export const defaultValues = {
    name: '',
    categories: [],
    clairify: {
        endpoint: '',
        grpcEndpoint: '',
        numConcurrentScans: '0',
    },
    uiEndpoint: window.location.origin,
    type: 'clairify',
    enabled: true,
    clusterIds: [],
} as ClairifyIntegration;

function ClairifyIntegrationForm({
    initialValues = null,
    isEdittable = false,
}: ClairifyIntegrationProps): ReactElement {
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
            {isEdittable && (
                <IntegrationFormToolBar>
                    <FormCancelButton onCancel={onCancel}>Cancel</FormCancelButton>
                    <FormTestButton
                        onTest={onTest}
                        isSubmitting={isSubmitting}
                        isTesting={isTesting}
                    >
                        Test
                    </FormTestButton>
                    <FormSaveButton
                        onSave={onSave}
                        isSubmitting={isSubmitting}
                        isTesting={isTesting}
                    >
                        Save
                    </FormSaveButton>
                </IntegrationFormToolBar>
            )}
            {message && <FormMessageBanner message={message} />}
            <PageSection variant="light" isFilled hasOverflowScroll>
                <Form isWidthLimited>
                    <FormGroup
                        label="Name"
                        isRequired
                        fieldId="name"
                        helperTextInvalid={errors?.name}
                        validated={errors?.name ? 'error' : 'default'}
                    >
                        <TextInput
                            isRequired
                            type="text"
                            id="name"
                            name="name"
                            value={values?.name}
                            onChange={onChange}
                            isDisabled={!isEdittable}
                        />
                    </FormGroup>
                    <FormGroup
                        label="Type"
                        isRequired
                        fieldId="categories"
                        helperTextInvalid={errors?.categories}
                        validated={errors?.categories ? 'error' : 'default'}
                    >
                        <FormMultiSelect
                            id="categories"
                            values={values?.categories}
                            onChange={onCustomChange}
                            isDisabled={!isEdittable}
                        >
                            <SelectOption key={0} value="SCANNER">
                                Image Scanner
                            </SelectOption>
                            <SelectOption key={1} value="NODE_SCANNER">
                                Node Scanner
                            </SelectOption>
                        </FormMultiSelect>
                    </FormGroup>
                    <FormGroup
                        label="Endpoint"
                        isRequired
                        fieldId="clairify.endpoint"
                        helperTextInvalid={errors?.clairify?.endpoint}
                        validated={errors?.clairify?.endpoint ? 'error' : 'default'}
                    >
                        <TextInput
                            isRequired
                            type="text"
                            id="clairify.endpoint"
                            name="clairify.endpoint"
                            value={values?.clairify?.endpoint}
                            onChange={onChange}
                            isDisabled={!isEdittable}
                        />
                    </FormGroup>
                    <FormGroup
                        label="GRPC Endpoint"
                        fieldId="clairify.grpcEndpoint"
                        helperText="Used For Node Scanning"
                        helperTextInvalid={errors?.clairify?.grpcEndpoint}
                        validated={errors?.clairify?.grpcEndpoint ? 'error' : 'default'}
                    >
                        <TextInput
                            isRequired
                            type="text"
                            id="clairify.grpcEndpoint"
                            name="clairify.grpcEndpoint"
                            value={values?.clairify?.grpcEndpoint}
                            onChange={onChange}
                            isDisabled={!isEdittable}
                        />
                    </FormGroup>
                    <FormGroup
                        label="Max Concurrent Image Scans"
                        fieldId="clairify.numConcurrentScans"
                        helperText="0 for default"
                        helperTextInvalid={errors?.clairify?.numConcurrentScans}
                        validated={errors?.clairify?.numConcurrentScans ? 'error' : 'default'}
                    >
                        <TextInput
                            isRequired
                            type="number"
                            id="clairify.numConcurrentScans"
                            name="clairify.numConcurrentScans"
                            value={values?.clairify?.numConcurrentScans}
                            onChange={onChange}
                            isDisabled={!isEdittable}
                        />
                    </FormGroup>
                </Form>
            </PageSection>
        </>
    );
}

export default ClairifyIntegrationForm;
