import React, { useState } from 'react';
import { ActionGroup, Alert, Button, Flex, Form, FormGroup } from '@patternfly/react-core';
import { PlusCircleIcon } from '@patternfly/react-icons';
import { useFormik } from 'formik';
import * as yup from 'yup';

import {
    DelegatedRegistryCluster,
    DelegatedRegistryConfig,
    updateDelegatedRegistryConfig,
} from 'services/DelegatedRegistryConfigService';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import DelegatedRegistriesTable, { registriesSchema } from './Components/DelegatedRegistriesTable';
import DelegatedScanningSettings from './Components/DelegatedScanningSettings';
import ToggleDelegatedScanning from './Components/ToggleDelegatedScanning';

const validationSchema = yup.object({
    registries: registriesSchema,
});

export type DelegatedImageScanningFormProps = {
    delegatedRegistryClusters: DelegatedRegistryCluster[];
    delegatedRegistryConfig: DelegatedRegistryConfig;
    isEditing: boolean;
    setDelegatedRegistryConfig: (delegatedRegistryConfig: DelegatedRegistryConfig) => void;
    setIsNotEditing: () => void;
};

function DelegatedImageScanningForm({
    delegatedRegistryClusters,
    delegatedRegistryConfig,
    isEditing,
    setDelegatedRegistryConfig,
    setIsNotEditing,
}) {
    const [errorMessage, setErrorMessage] = useState<string | null>(null);

    const formik = useFormik<DelegatedRegistryConfig>({
        initialValues: delegatedRegistryConfig,
        onSubmit,
        validationSchema,
    });
    const {
        dirty,
        isSubmitting,
        isValid,
        resetForm,
        setFieldValue,
        setSubmitting,
        // setTouched,
        submitForm,
        // touched,
        values,
    } = formik;

    function onSubmit(valuesToSubmit: DelegatedRegistryConfig) {
        updateDelegatedRegistryConfig(valuesToSubmit)
            .then((delegatedRegistryConfigUpdated) => {
                // Reset form state from response although in theory, same as payload.
                resetForm({ values: delegatedRegistryConfigUpdated });
                // Because Page renders Form after initial request,
                // the following update to its state, although consistent,
                // seems redundant with update of form state.
                setDelegatedRegistryConfig(delegatedRegistryConfigUpdated);
                setErrorMessage(null);
                setIsNotEditing();
            })
            .catch((error) => {
                setErrorMessage(getAxiosErrorMessage(error));
            })
            .finally(() => {
                setSubmitting(false);
            });
    }

    function onCancel() {
        resetForm();
        setIsNotEditing();
    }

    function addRegistry() {
        return setFieldValue('registries', [...values.registries, { path: '', clusterId: '' }]);
    }

    function deleteRegistry(indexToDelete) {
        return setFieldValue(
            'registries',
            values.registries.filter((_, index) => index !== indexToDelete)
        ).then(() => {
            // Formik updates errors but not touched.
            /*
            setTouched(
                {
                    registries: Array.isArray(touched.registries)
                        ? touched.registries.filter((_, index) => index !== indexToDelete)
                        : [],
                },
                false // does not affect validation
            ).catch(() => {
                // @typescript-eslint/no-floating-promises
            });
            */
        });
    }

    function setRegistryClusterId(indexToSet: number, clusterId: string) {
        return setFieldValue(
            'registries',
            values.registries.map((registry, index) =>
                index === indexToSet ? { path: registry.path, clusterId } : registry
            )
        );
    }

    function setRegistryPath(indexToSet: number, path: string) {
        return setFieldValue(
            'registries',
            values.registries.map((registry, index) =>
                index === indexToSet ? { path, clusterId: registry.clusterId } : registry
            )
        );
    }

    return (
        <Flex direction={{ default: 'column' }}>
            {typeof errorMessage === 'string' && (
                <Alert
                    title="Unable to save delegated image scanning configuration"
                    component="p"
                    variant="danger"
                    isInline
                >
                    {errorMessage}
                </Alert>
            )}
            <Form>
                <ToggleDelegatedScanning
                    enabledFor={values.enabledFor}
                    isEditing={isEditing}
                    setEnabledFor={(enabledFor) => setFieldValue('enabledFor', enabledFor)}
                />
                {values.enabledFor !== 'NONE' && (
                    <>
                        <DelegatedScanningSettings
                            clusters={delegatedRegistryClusters}
                            defaultClusterId={values.defaultClusterId}
                            isEditing={isEditing}
                            setDefaultClusterId={(defaultClusterId) =>
                                setFieldValue('defaultClusterId', defaultClusterId)
                            }
                        />
                        <FormGroup label="Registries">
                            {values.registries.length > 0 ? (
                                <>
                                    <DelegatedRegistriesTable
                                        clusters={delegatedRegistryClusters}
                                        defaultClusterId={values.defaultClusterId}
                                        deleteRegistry={deleteRegistry}
                                        formik={formik}
                                        isEditing={isEditing}
                                        registries={values.registries}
                                        setRegistryClusterId={setRegistryClusterId}
                                        setRegistryPath={setRegistryPath}
                                    />
                                </>
                            ) : (
                                <p>No registries specified.</p>
                            )}
                            {isEditing && (
                                <Button
                                    variant="link"
                                    isInline
                                    icon={<PlusCircleIcon />}
                                    onClick={addRegistry}
                                    className="pf-v5-u-mt-md"
                                >
                                    Add registry
                                </Button>
                            )}
                        </FormGroup>
                    </>
                )}
                {isEditing && (
                    <ActionGroup>
                        <Button
                            variant="primary"
                            isDisabled={!dirty || !isValid || isSubmitting}
                            isLoading={isSubmitting}
                            onClick={submitForm}
                        >
                            Save
                        </Button>
                        <Button variant="secondary" onClick={onCancel} isDisabled={isSubmitting}>
                            Cancel
                        </Button>
                    </ActionGroup>
                )}
            </Form>
        </Flex>
    );
}

export default DelegatedImageScanningForm;
