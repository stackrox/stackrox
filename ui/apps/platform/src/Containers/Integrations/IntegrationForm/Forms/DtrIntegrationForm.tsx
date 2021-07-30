import React, { ReactElement } from 'react';
import { TextInput, PageSection, Form, Switch, SelectOption } from '@patternfly/react-core';
import * as yup from 'yup';

import FormMultiSelect from 'Components/FormMultiSelect';
import usePageState from 'Containers/Integrations/hooks/usePageState';
import useIntegrationForm from '../useIntegrationForm';

import IntegrationFormActions from '../IntegrationFormActions';
import FormCancelButton from '../FormCancelButton';
import FormTestButton from '../FormTestButton';
import FormSaveButton from '../FormSaveButton';
import FormMessageBanner from '../FormMessageBanner';
import FormLabelGroup from '../FormLabelGroup';

export type DtrIntegration = {
    id?: string;
    name: string;
    categories: ('REGISTRY' | 'SCANNER')[];
    dtr: {
        endpoint: string;
        username: string;
        password: string;
        insecure: boolean;
    };
    skipTestIntegration: boolean;
    type: 'dtr';
    enabled: boolean;
    clusterIds: string[];
};

export type DtrIntegrationFormValues = {
    config: DtrIntegration;
    updatePassword: boolean;
};

export type DtrIntegrationFormProps = {
    initialValues: DtrIntegration | null;
    isEdittable?: boolean;
};

export const validationSchema = yup.object().shape({
    config: yup.object().shape({
        name: yup.string().required('Required'),
        categories: yup
            .array()
            .of(yup.string().oneOf(['REGISTRY', 'SCANNER']))
            .min(1, 'Must have at least one type selected')
            .required('Required'),
        dtr: yup.object().shape({
            endpoint: yup.string().required('Required').min(1),
            username: yup.string(),
            password: yup.string(),
            insecure: yup.bool(),
        }),
        skipTestIntegration: yup.bool(),
        type: yup.string().matches(/dtr/),
        enabled: yup.bool(),
        clusterIds: yup.array().of(yup.string()),
    }),
    updatePassword: yup.bool(),
});

export const defaultValues: DtrIntegrationFormValues = {
    config: {
        name: '',
        categories: [],
        dtr: {
            endpoint: '',
            username: '',
            password: '',
            insecure: false,
        },
        skipTestIntegration: false,
        type: 'dtr',
        enabled: true,
        clusterIds: [],
    },
    updatePassword: true,
};

function DtrIntegrationForm({
    initialValues = null,
    isEdittable = false,
}: DtrIntegrationFormProps): ReactElement {
    const formInitialValues = defaultValues;
    if (initialValues) {
        formInitialValues.config = { ...formInitialValues.config, ...initialValues };
        // We want to clear the password because backend returns '******' to represent that there
        // are currently stored credentials
        formInitialValues.config.dtr.password = '';
    }
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
    } = useIntegrationForm<DtrIntegrationFormValues, typeof validationSchema>({
        initialValues: formInitialValues,
        validationSchema,
    });
    const { isCreating } = usePageState();

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
                    <FormLabelGroup label="Name" isRequired fieldId="config.name" errors={errors}>
                        <TextInput
                            isRequired
                            type="text"
                            id="config.name"
                            name="config.name"
                            placeholder="(ex. Prod Docker Trusted Registry)"
                            value={values.config.name}
                            onChange={onChange}
                            isDisabled={!isEdittable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        label="Type"
                        isRequired
                        fieldId="config.categories"
                        errors={errors}
                    >
                        <FormMultiSelect
                            id="config.categories"
                            values={values.config.categories}
                            onChange={onCustomChange}
                            isDisabled={!isEdittable}
                        >
                            <SelectOption key={0} value="REGISTRY">
                                Registry
                            </SelectOption>
                            <SelectOption key={1} value="SCANNER">
                                Scanner
                            </SelectOption>
                        </FormMultiSelect>
                    </FormLabelGroup>
                    <FormLabelGroup
                        label="Endpoint"
                        isRequired
                        fieldId="config.dtr.endpoint"
                        errors={errors}
                    >
                        <TextInput
                            isRequired
                            type="text"
                            id="config.dtr.endpoint"
                            name="config.dtr.endpoint"
                            placeholder="(ex. dtr.example.com)"
                            value={values.config.dtr.endpoint}
                            onChange={onChange}
                            isDisabled={!isEdittable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup label="Username" fieldId="config.dtr.username" errors={errors}>
                        <TextInput
                            isRequired
                            type="text"
                            id="config.dtr.username"
                            name="config.dtr.username"
                            value={values.config.dtr.username}
                            onChange={onChange}
                            isDisabled={!isEdittable}
                        />
                    </FormLabelGroup>
                    {!isCreating && (
                        <FormLabelGroup
                            label="Update Password"
                            fieldId="updatePassword"
                            helperText="Setting this to false will use the currently stored credentials, if they exist."
                            errors={errors}
                        >
                            <Switch
                                id="updatePassword"
                                name="updatePassword"
                                aria-label="update password"
                                isChecked={values.updatePassword}
                                onChange={onChange}
                                isDisabled={!isEdittable}
                            />
                        </FormLabelGroup>
                    )}
                    {values.updatePassword && (
                        <FormLabelGroup
                            label="Password"
                            fieldId="config.dtr.password"
                            errors={errors}
                        >
                            <TextInput
                                isRequired
                                type="password"
                                id="config.dtr.password"
                                name="config.dtr.password"
                                value={values.config.dtr.password}
                                onChange={onChange}
                                isDisabled={!isEdittable}
                            />
                        </FormLabelGroup>
                    )}
                    <FormLabelGroup
                        label="Disable TLS Certificate Validation (Insecure)"
                        fieldId="config.dtr.insecure"
                        errors={errors}
                    >
                        <Switch
                            id="config.dtr.insecure"
                            name="config.dtr.insecure"
                            aria-label="disable tls certificate validation"
                            isChecked={values.config.dtr.insecure}
                            onChange={onChange}
                            isDisabled={!isEdittable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        label="Create Integration Without Testing"
                        fieldId="config.skipTestIntegration"
                        errors={errors}
                    >
                        <Switch
                            id="config.skipTestIntegration"
                            name="config.skipTestIntegration"
                            aria-label="skip test integration"
                            isChecked={values.config.skipTestIntegration}
                            onChange={onChange}
                            isDisabled={!isEdittable}
                        />
                    </FormLabelGroup>
                </Form>
            </PageSection>
            {isEdittable && (
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

export default DtrIntegrationForm;
