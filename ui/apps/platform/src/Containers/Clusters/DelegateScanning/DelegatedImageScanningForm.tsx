import React, { useState } from 'react';
import { ActionGroup, Alert, Button, Flex, Form } from '@patternfly/react-core';
import { useFormik } from 'formik';

import {
    DelegatedRegistryCluster,
    DelegatedRegistryConfig,
    updateDelegatedRegistryConfig,
} from 'services/DelegatedRegistryConfigService';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import DelegatedRegistriesList from './Components/DelegatedRegistriesList';
import DelegatedScanningSettings from './Components/DelegatedScanningSettings';
import ToggleDelegatedScanning from './Components/ToggleDelegatedScanning';

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
    const [errorMessage, setErrorMessage] = useState('');

    const {
        dirty,
        // errors,
        isSubmitting,
        isValid,
        resetForm,
        setFieldValue,
        setSubmitting,
        submitForm,
        values,
    } = useFormik<DelegatedRegistryConfig>({
        initialValues: delegatedRegistryConfig,
        // validationSchema,
        onSubmit: (valuesToSubmit) => {
            updateDelegatedRegistryConfig(valuesToSubmit)
                .then((delegatedRegistryConfigUpdated) => {
                    // Reset form state from response although in theory, same as payload.
                    resetForm({ values: delegatedRegistryConfigUpdated });
                    // Because Page renders Form after initial request,
                    // the following update to its state, although consistent,
                    // seems redundant with update of form state.
                    setDelegatedRegistryConfig(delegatedRegistryConfigUpdated);
                    setIsNotEditing();
                })
                .catch((error) => {
                    setErrorMessage(getAxiosErrorMessage(error));
                })
                .finally(() => {
                    setSubmitting(false);
                });
        },
    });

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
        );
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
            {errorMessage.length !== 0 && (
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
                        <DelegatedRegistriesList
                            addRegistry={addRegistry}
                            clusters={delegatedRegistryClusters}
                            defaultClusterId={values.defaultClusterId}
                            deleteRegistry={deleteRegistry}
                            isEditing={isEditing}
                            registries={values.registries}
                            setRegistryPath={setRegistryPath}
                            setRegistryClusterId={setRegistryClusterId}
                        />
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
