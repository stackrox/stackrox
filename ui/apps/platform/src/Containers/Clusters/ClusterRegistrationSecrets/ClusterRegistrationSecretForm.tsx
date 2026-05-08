import type { FormEvent, ReactElement } from 'react';
import { useNavigate } from 'react-router-dom-v5-compat';
import {
    ActionGroup,
    Alert,
    Button,
    DatePicker,
    Flex,
    Form,
    PageSection,
    Radio,
    TextInput,
    yyyyMMddFormat,
} from '@patternfly/react-core';
import { useFormik } from 'formik';
import * as yup from 'yup';

import FormLabelGroup from 'Components/PatternFly/FormLabelGroup';
import useAnalytics, { DOWNLOAD_CLUSTER_REGISTRATION_SECRET } from 'hooks/useAnalytics';
import useRestMutation from 'hooks/useRestMutation';
import { generateClusterRegistrationSecretExtended } from 'services/ClustersService';
import type { GenerateClusterRegistrationSecretExtendedRequest } from 'services/ClustersService';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import ClusterRegistrationSecretsHeader from './ClusterRegistrationSecretsHeader';

import { downloadClusterRegistrationSecret } from './ClusterRegistrationSecretForm.utils';

type ValidityMode = 'none' | 'date' | 'hours';

export type ClusterRegistrationSecretFormValues = {
    validityMode: ValidityMode;
} & GenerateClusterRegistrationSecretExtendedRequest;

export const initialValues: ClusterRegistrationSecretFormValues = {
    name: '',
    validityMode: 'none',
    validUntil: undefined,
    validFor: undefined,
    maxRegistrations: undefined,
};

// https://github.com/stackrox/stackrox/blob/master/central/clusterinit/backend/validation.go#L11
const nameValidatorRegExp = /^[a-zA-Z0-9._-]+$/;

const validationSchema: yup.ObjectSchema<ClusterRegistrationSecretFormValues> = yup.object().shape({
    name: yup
        .string()
        .trim()
        .matches(
            nameValidatorRegExp,
            'Name can have only the following characters: letters, digits, period, underscore, hyphen (but no spaces)'
        )
        .required('Cluster registration secret name is required'),
    validityMode: yup
        .string()
        .oneOf(['none', 'date', 'hours'] as const)
        .required(),
    validUntil: yup
        .string()
        .optional()
        .when('validityMode', {
            is: 'date',
            then: (schema) =>
                schema
                    .required('A date is required when using date-based validity')
                    .test('future-date', 'Expiration must be after today', (val) => {
                        if (!val) {
                            return true;
                        }
                        const tomorrow = new Date();
                        tomorrow.setHours(0, 0, 0, 0);
                        tomorrow.setDate(tomorrow.getDate() + 1);
                        return new Date(val) >= tomorrow;
                    }),
        }),
    validFor: yup
        .string()
        .optional()
        .when('validityMode', {
            is: 'hours',
            then: (schema) =>
                schema
                    .required('Hours value is required')
                    .test('positive-integer', 'Must be a number greater than 0', (val) => {
                        const n = parseInt(val ?? '', 10);
                        return Number.isInteger(n) && n > 0;
                    }),
        }),
    // Matches maxRegistrationsUpperLimit in central/clusterinit/backend/backend_impl.go
    maxRegistrations: yup
        .string()
        .optional()
        .test('valid-range', 'Must be a number between 1 and 100', (val) => {
            if (!val) {
                return true;
            }
            const n = parseInt(val, 10);
            return Number.isInteger(n) && n >= 1 && n <= 100;
        }),
});

export function buildRequestData(
    values: ClusterRegistrationSecretFormValues
): GenerateClusterRegistrationSecretExtendedRequest {
    const data: GenerateClusterRegistrationSecretExtendedRequest = { name: values.name };

    if (values.validityMode === 'date') {
        if (!values.validUntil) {
            throw new Error('A date is required when validity mode is "date"');
        }
        data.validUntil = values.validUntil;
    }

    if (values.validityMode === 'hours') {
        if (!values.validFor) {
            throw new Error('An hours value is required when validity mode is "hours"');
        }
        const hours = parseInt(values.validFor, 10);
        if (!Number.isInteger(hours) || hours <= 0) {
            throw new Error('Hours must be a positive integer');
        }
        data.validFor = `${hours * 3600}s`;
    }

    if (values.maxRegistrations) {
        data.maxRegistrations = values.maxRegistrations;
    }

    return data;
}

function ClusterRegistrationSecretForm(): ReactElement {
    const { analyticsTrack } = useAnalytics();
    const navigate = useNavigate();

    const { mutate, error } = useRestMutation(
        async (formValues: ClusterRegistrationSecretFormValues) => {
            const response = await generateClusterRegistrationSecretExtended(
                buildRequestData(formValues)
            );
            return { name: formValues.name, response };
        },
        {
            onSuccess: ({ name, response }) => {
                downloadClusterRegistrationSecret(name, response);
                goBack();
            },
            onSettled: () => setSubmitting(false),
        }
    );
    const {
        errors,
        handleBlur,
        isSubmitting,
        isValid,
        setFieldTouched,
        setFieldValue,
        setValues,
        submitForm,
        touched,
        values,
        setSubmitting,
    } = useFormik({
        initialValues,
        onSubmit: (formValues, { setSubmitting: setFormSubmitting }) => {
            setFormSubmitting(true);
            mutate(formValues);
        },
        validateOnMount: true,
        validationSchema,
    });

    function goBack() {
        navigate(-1);
    }

    function onDateChange(
        _event: FormEvent<HTMLInputElement>,
        _dateString: string,
        date: Date | undefined
    ) {
        if (date) {
            const now = new Date();
            date.setHours(now.getHours(), now.getMinutes(), 0, 0);
            return setFieldValue('validUntil', date.toISOString());
        }
        return setFieldValue('validUntil', undefined);
    }

    const datePickerValue = values.validUntil ? yyyyMMddFormat(new Date(values.validUntil)) : '';

    return (
        <>
            <ClusterRegistrationSecretsHeader title="Create cluster registration secret" />
            <PageSection>
                <Flex direction={{ default: 'column' }} gap={{ default: 'gapLg' }}>
                    <Form isWidthLimited>
                        <FormLabelGroup
                            fieldId="name"
                            label="Name"
                            isRequired
                            errors={errors}
                            touched={touched}
                        >
                            <TextInput
                                id="name"
                                type="text"
                                name="name"
                                isRequired
                                value={values.name}
                                onBlur={handleBlur}
                                onChange={(_event, value) => setFieldValue('name', value)}
                            />
                        </FormLabelGroup>
                        <FormLabelGroup
                            fieldId="validityMode"
                            label="Validity period"
                            errors={errors}
                            touched={touched}
                        >
                            <Flex direction={{ default: 'column' }} gap={{ default: 'gapSm' }}>
                                <Radio
                                    id="validity-none"
                                    name="validityMode"
                                    label="Default"
                                    description="Server-defined expiration (currently 24 hours)"
                                    isChecked={values.validityMode === 'none'}
                                    onChange={() => {
                                        return setValues({
                                            ...values,
                                            validityMode: 'none',
                                            validUntil: undefined,
                                            validFor: undefined,
                                        });
                                    }}
                                />
                                <Radio
                                    id="validity-date"
                                    name="validityMode"
                                    label="By date"
                                    isChecked={values.validityMode === 'date'}
                                    onChange={() => {
                                        return setValues({
                                            ...values,
                                            validityMode: 'date',
                                            validFor: undefined,
                                        });
                                    }}
                                    body={
                                        values.validityMode === 'date' && (
                                            <FormLabelGroup
                                                fieldId="validUntil"
                                                errors={errors}
                                                touched={touched}
                                            >
                                                <DatePicker
                                                    value={datePickerValue}
                                                    onChange={onDateChange}
                                                    onBlur={() =>
                                                        setFieldTouched('validUntil', true)
                                                    }
                                                    validators={[
                                                        (date) => {
                                                            const tomorrow = new Date();
                                                            tomorrow.setHours(0, 0, 0, 0);
                                                            tomorrow.setDate(
                                                                tomorrow.getDate() + 1
                                                            );
                                                            if (date < tomorrow) {
                                                                return 'Date must be after today';
                                                            }
                                                            return '';
                                                        },
                                                    ]}
                                                />
                                            </FormLabelGroup>
                                        )
                                    }
                                />
                                <Radio
                                    id="validity-hours"
                                    name="validityMode"
                                    label="By hours"
                                    isChecked={values.validityMode === 'hours'}
                                    onChange={() => {
                                        return setValues({
                                            ...values,
                                            validityMode: 'hours',
                                            validUntil: undefined,
                                        });
                                    }}
                                    body={
                                        values.validityMode === 'hours' && (
                                            <FormLabelGroup
                                                fieldId="validFor"
                                                errors={errors}
                                                touched={touched}
                                            >
                                                <TextInput
                                                    id="validFor"
                                                    type="number"
                                                    name="validFor"
                                                    value={values.validFor ?? ''}
                                                    min={1}
                                                    step={1}
                                                    placeholder="24"
                                                    onBlur={handleBlur}
                                                    onChange={(_event, value) =>
                                                        setFieldValue(
                                                            'validFor',
                                                            value || undefined
                                                        )
                                                    }
                                                />
                                            </FormLabelGroup>
                                        )
                                    }
                                />
                            </Flex>
                        </FormLabelGroup>
                        <FormLabelGroup
                            fieldId="maxRegistrations"
                            label="Max registrations"
                            helperText="Maximum clusters that can register with this secret (1-100). Leave blank for unlimited."
                            errors={errors}
                            touched={touched}
                        >
                            <TextInput
                                id="maxRegistrations"
                                type="number"
                                name="maxRegistrations"
                                value={values.maxRegistrations ?? ''}
                                min={1}
                                max={100}
                                placeholder="Unlimited"
                                onBlur={handleBlur}
                                onChange={(_event, value) =>
                                    setFieldValue('maxRegistrations', value || undefined)
                                }
                            />
                        </FormLabelGroup>
                    </Form>
                    <Alert variant="info" isInline title="Download YAML file" component="p">
                        <p>
                            You can download the YAML file only once, when you create a cluster
                            registration secret.
                        </p>
                        <p>Store the YAML file securely because it contains secrets.</p>
                    </Alert>
                    {error !== undefined && (
                        <Alert
                            variant="danger"
                            isInline
                            title="Unable to create or download cluster registration secret"
                            component="p"
                        >
                            {getAxiosErrorMessage(error)}
                        </Alert>
                    )}
                    <ActionGroup>
                        <Button
                            variant="primary"
                            isDisabled={isSubmitting || !isValid}
                            isLoading={isSubmitting}
                            onClick={() => {
                                analyticsTrack(DOWNLOAD_CLUSTER_REGISTRATION_SECRET);
                                return submitForm();
                            }}
                        >
                            Download
                        </Button>
                        <Button variant="link" isDisabled={isSubmitting} onClick={goBack}>
                            Cancel
                        </Button>
                    </ActionGroup>
                </Flex>
            </PageSection>
        </>
    );
}

export default ClusterRegistrationSecretForm;
