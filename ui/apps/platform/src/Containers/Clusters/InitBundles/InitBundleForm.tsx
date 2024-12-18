import React, { ReactElement, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
    ActionGroup,
    Alert,
    Button,
    Divider,
    Flex,
    Form,
    FormGroup,
    FormHelperText,
    HelperText,
    HelperTextItem,
    PageSection,
    Radio,
    TextInput,
} from '@patternfly/react-core';
import { Select, SelectOption } from '@patternfly/react-core/deprecated';
import { useFormik } from 'formik';
import * as yup from 'yup';

import FormLabelGroup from 'Components/PatternFly/FormLabelGroup';
import useSelectToggle from 'hooks/patternfly/useSelectToggle';
import useAnalytics, { DOWNLOAD_INIT_BUNDLE } from 'hooks/useAnalytics';
import { generateClusterInitBundle } from 'services/ClustersService'; // ClusterInitBundle
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import InitBundlesHeader from './InitBundlesHeader';

import {
    InstallationKey,
    PlatformKey,
    downloadBundle,
    installationOptions,
    platformOptions,
} from './InitBundleForm.utils';

export type InitBundleFormValues = {
    installation: InstallationKey;
    name: string;
    platform: PlatformKey;
};

export const initialValues: InitBundleFormValues = {
    installation: 'Operator',
    name: '',
    platform: 'OpenShift',
};

// https://github.com/stackrox/stackrox/blob/master/central/clusterinit/backend/validation.go#L11
const nameValidatorRegExp = /^[a-zA-Z0-9._-]+$/;

const validationSchema: yup.ObjectSchema<InitBundleFormValues> = yup.object().shape({
    name: yup
        .string()
        .trim()
        .matches(
            nameValidatorRegExp,
            'Name can have only the following characters: letters, digits, period, underscore, hyphen (but no spaces)'
        )
        .required('Bundle name is required'),
    installation: yup.string().trim().required(), // Select
    platform: yup.string().trim().required(), // Radio
});

function InitBundleForm(): ReactElement {
    const { analyticsTrack } = useAnalytics();
    const navigate = useNavigate();
    const [errorMessage, setErrorMessage] = useState('');
    const {
        errors,
        handleBlur,
        isSubmitting,
        isValid,
        setFieldValue,
        setValues,
        submitForm,
        touched,
        values,
    } = useFormik({
        initialValues,
        onSubmit: (values, { setSubmitting }) => {
            setSubmitting(true);
            const { installation, name } = values;
            generateClusterInitBundle({ name })
                .then(({ response }) => {
                    setErrorMessage('');
                    downloadBundle(installation, name, response); // TODO try catch?
                    setSubmitting(false);
                    goBack();
                })
                .catch((error) => {
                    setErrorMessage(getAxiosErrorMessage(error));
                    setSubmitting(false);
                });
        },
        validateOnMount: true, // disable Next when Name is empty
        validationSchema,
    });
    const { isOpen, onToggle } = useSelectToggle();

    function goBack() {
        navigate(-1); // to InputBundlesTable or NoClustersPage
    }

    // return setWhatever solves problem reported by typescript-eslint no-floating-promises

    function onChangeTextInput(value, event) {
        return setFieldValue(event.target.id, value);
    }

    function onChangePlatform(value) {
        return setValues({
            installation: value === 'OpenShift' ? 'Operator' : 'Helm',
            name: values.name, // redundant but function requires all values
            platform: value,
        });
    }

    function onSelectInstallation(value) {
        return setFieldValue('installation', value);
    }

    return (
        <>
            <InitBundlesHeader title="Create bundle" />
            <Divider component="div" />
            <PageSection variant="light">
                <Flex direction={{ default: 'column' }}>
                    <Form>
                        <FormLabelGroup
                            fieldId="name"
                            label="Name"
                            isRequired
                            errors={errors}
                            touched={touched}
                        >
                            <TextInput
                                type="text"
                                id="name"
                                name="name"
                                isRequired
                                value={values.name}
                                onBlur={handleBlur}
                                onChange={(event, value) => onChangeTextInput(value, event)}
                            />
                        </FormLabelGroup>
                        <FormGroup
                            fieldId="platform"
                            label="Platform of secured clusters"
                            isRequired
                        >
                            <Flex
                                direction={{ default: 'column' }}
                                spaceItems={{ default: 'spaceItemsSm' }}
                            >
                                {Object.entries(platformOptions).map(
                                    ([platformKey, platformLabel]) => (
                                        <Radio
                                            key={platformKey}
                                            name={platformKey}
                                            value={platformKey}
                                            onChange={() => onChangePlatform(platformKey)}
                                            label={platformLabel}
                                            id={platformKey}
                                            isChecked={values.platform === platformKey}
                                        />
                                    )
                                )}
                            </Flex>
                        </FormGroup>
                        <FormGroup
                            fieldId="installation"
                            label="Installation method for secured cluster services"
                            isRequired
                        >
                            <Select
                                variant="single"
                                toggleAriaLabel="Installation method menu toggle"
                                aria-label="Select an installation method"
                                isDisabled={values.platform !== 'OpenShift'}
                                onToggle={(_e, v) => onToggle(v)}
                                onSelect={(_event, value) => onSelectInstallation(value)}
                                selections={values.installation}
                                isOpen={isOpen}
                                // className="pf-v5-u-flex-basis-0"
                            >
                                {Object.entries(installationOptions)
                                    .filter(
                                        ([installationKey]) =>
                                            values.platform === 'OpenShift' ||
                                            installationKey !== 'Operator'
                                    )
                                    .map(([installationKey, installationLabel]) => (
                                        <SelectOption key={installationKey} value={installationKey}>
                                            {installationLabel}
                                        </SelectOption>
                                    ))}
                            </Select>
                            <FormHelperText>
                                <HelperText>
                                    <HelperTextItem>
                                        You can use one bundle to secure multiple clusters that have
                                        the same installation method.
                                    </HelperTextItem>
                                </HelperText>
                            </FormHelperText>
                        </FormGroup>
                    </Form>
                    <Alert variant="info" isInline title="Download YAML file" component="p">
                        <p>
                            You can download the YAML file only once, when you create an init
                            bundle.
                        </p>
                        <p>Store the YAML file securely because it contains secrets.</p>
                    </Alert>
                    {errorMessage && (
                        <Alert
                            variant="danger"
                            isInline
                            title="Unable to create or download bundle"
                            component="p"
                        >
                            {errorMessage}
                        </Alert>
                    )}
                    <ActionGroup>
                        <Button
                            variant="primary"
                            isDisabled={isSubmitting || !isValid}
                            isLoading={isSubmitting}
                            onClick={() => {
                                analyticsTrack(DOWNLOAD_INIT_BUNDLE);
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

export default InitBundleForm;
