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

export type TenableIntegration = {
    id?: string;
    name: string;
    categories: ('REGISTRY' | 'SCANNER')[];
    tenable: {
        accessKey: string;
        secretKey: string;
    };
    skipTestIntegration: boolean;
    type: 'tenable';
    enabled: boolean;
    clusterIds: string[];
};

export type TenableIntegrationFormValues = {
    config: TenableIntegration;
    updatePassword: boolean;
};

export type TenableIntegrationFormProps = {
    initialValues: TenableIntegration | null;
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
        tenable: yup.object().shape({
            accessKey: yup.string(),
            secretKey: yup.string(),
        }),
        skipTestIntegration: yup.bool(),
        type: yup.string().matches(/tenable/),
        enabled: yup.bool(),
        clusterIds: yup.array().of(yup.string()),
    }),
    updatePassword: yup.bool(),
});

export const defaultValues: TenableIntegrationFormValues = {
    config: {
        name: '',
        categories: [],
        tenable: {
            accessKey: '',
            secretKey: '',
        },
        skipTestIntegration: false,
        type: 'tenable',
        enabled: true,
        clusterIds: [],
    },
    updatePassword: true,
};

function TenableIntegrationForm({
    initialValues = null,
    isEdittable = false,
}: TenableIntegrationFormProps): ReactElement {
    const formInitialValues = defaultValues;
    if (initialValues) {
        formInitialValues.config = { ...formInitialValues.config, ...initialValues };
        // We want to clear the password because backend returns '******' to represent that there
        // are currently stored credentials
        formInitialValues.config.tenable.accessKey = '';
        formInitialValues.config.tenable.secretKey = '';
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
    } = useIntegrationForm<TenableIntegrationFormValues, typeof validationSchema>({
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
                        <>
                            <FormLabelGroup
                                label="Access Key"
                                fieldId="config.tenable.accessKey"
                                errors={errors}
                            >
                                <TextInput
                                    isRequired
                                    type="text"
                                    id="config.tenable.accessKey"
                                    name="config.tenable.accessKey"
                                    value={values.config.tenable.accessKey}
                                    onChange={onChange}
                                    isDisabled={!isEdittable}
                                />
                            </FormLabelGroup>
                            <FormLabelGroup
                                label="Secret Key"
                                fieldId="config.tenable.secretKey"
                                errors={errors}
                            >
                                <TextInput
                                    isRequired
                                    type="password"
                                    id="config.tenable.secretKey"
                                    name="config.tenable.secretKey"
                                    value={values.config.tenable.secretKey}
                                    onChange={onChange}
                                    isDisabled={!isEdittable}
                                />
                            </FormLabelGroup>
                        </>
                    )}
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

export default TenableIntegrationForm;
