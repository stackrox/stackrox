import React, { useState, useEffect, ReactNode } from 'react';
import {
    Alert,
    Breadcrumb,
    BreadcrumbItem,
    Button,
    Flex,
    FlexItem,
    Form,
    PageSection,
    Title,
} from '@patternfly/react-core';

import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import PageTitle from 'Components/PageTitle';
import { clustersBasePath } from 'routePaths';
import {
    fetchDelegatedRegistryConfig,
    fetchDelegatedRegistryClusters,
    updateDelegatedRegistryConfig,
    DelegatedRegistryConfig,
    DelegatedRegistryCluster,
} from 'services/DelegatedRegistryConfigService';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import ToggleDelegatedScanning from './Components/ToggleDelegatedScanning';
import DelegatedScanningSettings from './Components/DelegatedScanningSettings';
import DelegatedRegistriesList from './Components/DelegatedRegistriesList';

type AlertObj = {
    type: 'danger' | 'success';
    title: string;
    body: ReactNode;
};
const initialDelegatedState: DelegatedRegistryConfig = {
    enabledFor: 'NONE',
    defaultClusterId: '',
    registries: [],
};

function DelegateScanningPage() {
    const displayedPageTitle = 'Delegated Image Scanning';
    const [delegatedRegistryConfig, setDedicatedRegistryConfig] =
        useState<DelegatedRegistryConfig>(initialDelegatedState);
    const [delegatedRegistryClusters, setDelegatedRegistryClusters] = useState<
        DelegatedRegistryCluster[]
    >([]);
    const [alertObj, setAlertObj] = useState<AlertObj | null>(null);

    useEffect(() => {
        fetchClusters();
        fetchConfig();
    }, []);

    function fetchConfig() {
        setAlertObj(null);
        fetchDelegatedRegistryConfig()
            .then(setDedicatedRegistryConfig)
            .catch((error) => {
                const newErrorObj: AlertObj = {
                    type: 'danger',
                    title: 'Problem retrieving the delegated scanning configuration from the server',
                    body: (
                        <>
                            <p>{getAxiosErrorMessage(error)}</p>
                            <p>
                                Try reloading the page. If this problem persists, contact support.
                            </p>
                        </>
                    ),
                };
                setAlertObj(newErrorObj);
                setDedicatedRegistryConfig(initialDelegatedState);
            });
    }

    function fetchClusters() {
        fetchDelegatedRegistryClusters()
            .then((clusters) => {
                const validClusters = clusters.filter((cluster) => cluster.isValid);
                setDelegatedRegistryClusters(validClusters);
            })
            .catch((error) => {
                const newErrorObj: AlertObj = {
                    type: 'danger',
                    title: 'Problem retrieving clusters eligible for delegated scanning',
                    body: (
                        <>
                            <p>{getAxiosErrorMessage(error)}</p>
                            <p>
                                Try reloading the page. If this problem persists, contact support.
                            </p>
                        </>
                    ),
                };
                setAlertObj(newErrorObj);
                setDelegatedRegistryClusters([]);
            });
    }

    function onChangeEnabledFor(newEnabledState) {
        const newState: DelegatedRegistryConfig = { ...delegatedRegistryConfig };

        newState.enabledFor = newEnabledState;

        setDedicatedRegistryConfig(newState);
    }

    function onChangeDefaultCluster(newClusterId) {
        const newState: DelegatedRegistryConfig = { ...delegatedRegistryConfig };

        newState.defaultClusterId = newClusterId;

        setDedicatedRegistryConfig(newState);
    }

    function addRegistryRow() {
        const newState: DelegatedRegistryConfig = { ...delegatedRegistryConfig };

        newState.registries.push({
            path: '',
            clusterId: '',
        });

        setDedicatedRegistryConfig(newState);
    }

    function deleteRow(rowIndex) {
        const newState: DelegatedRegistryConfig = { ...delegatedRegistryConfig };

        const newRegistries = delegatedRegistryConfig.registries.filter((_, i) => i !== rowIndex);

        newState.registries = newRegistries;

        setDedicatedRegistryConfig(newState);
    }

    function handlePathChange(rowIndex: number, value: string) {
        const newState: DelegatedRegistryConfig = { ...delegatedRegistryConfig };

        const newRegistries = delegatedRegistryConfig.registries.map((registry, i) => {
            if (i === rowIndex) {
                return {
                    path: value,
                    clusterId: registry.clusterId,
                };
            }

            return registry;
        });

        newState.registries = newRegistries;

        setDedicatedRegistryConfig(newState);
    }

    function handleClusterChange(rowIndex: number, value: string) {
        const newState: DelegatedRegistryConfig = { ...delegatedRegistryConfig };

        const newRegistries = delegatedRegistryConfig.registries.map((registry, i) => {
            if (i === rowIndex) {
                return {
                    path: registry.path,
                    clusterId: value,
                };
            }

            return registry;
        });

        newState.registries = newRegistries;

        setDedicatedRegistryConfig(newState);
    }

    function onSave() {
        setAlertObj(null);
        updateDelegatedRegistryConfig(delegatedRegistryConfig)
            .then(() => {
                const newSuccessObj: AlertObj = {
                    type: 'success',
                    title: 'Delegated scanning configuration saved successfully',
                    body: <></>,
                };
                setAlertObj(newSuccessObj);
            })
            .catch((error) => {
                const newErrorObj: AlertObj = {
                    type: 'danger',
                    title: 'Problem saving the delegated scanning configuration to the server',
                    body: (
                        <>
                            <p>{getAxiosErrorMessage(error)}</p>
                            <p>
                                Try reloading the page. If this problem persists, contact support.
                            </p>
                        </>
                    ),
                };
                setAlertObj(newErrorObj);
            });
    }

    function onCancel() {
        fetchConfig();
    }

    return (
        <>
            <PageTitle title={displayedPageTitle} />
            <PageSection variant="light" padding={{ default: 'noPadding' }}>
                <Flex direction={{ default: 'column' }} className="pf-v5-u-py-lg pf-v5-u-pl-lg">
                    <FlexItem>
                        <Breadcrumb>
                            <BreadcrumbItemLink to={clustersBasePath}>Clusters</BreadcrumbItemLink>
                            <BreadcrumbItem isActive>{displayedPageTitle}</BreadcrumbItem>
                        </Breadcrumb>
                    </FlexItem>
                    <FlexItem>
                        <Title headingLevel="h1">{displayedPageTitle}</Title>
                    </FlexItem>
                    <FlexItem>
                        Delegating image scanning allows you to use secured clusters for scanning
                        instead of Central services.
                    </FlexItem>
                </Flex>
            </PageSection>
            <PageSection>
                {!!alertObj && (
                    <Alert
                        title={alertObj.title}
                        component="p"
                        variant={alertObj.type}
                        isInline
                        className="pf-v5-u-mb-lg"
                    >
                        {alertObj.body}
                    </Alert>
                )}
                <Form>
                    <ToggleDelegatedScanning
                        enabledFor={delegatedRegistryConfig.enabledFor}
                        onChangeEnabledFor={onChangeEnabledFor}
                    />
                    {/* TODO: decide who to structure this form, where the `enabledFor` value spans multiple inputs */}
                    {delegatedRegistryConfig.enabledFor !== 'NONE' && (
                        <>
                            <DelegatedScanningSettings
                                clusters={delegatedRegistryClusters}
                                selectedClusterId={delegatedRegistryConfig.defaultClusterId}
                                setSelectedClusterId={onChangeDefaultCluster}
                            />
                            <DelegatedRegistriesList
                                registries={delegatedRegistryConfig.registries}
                                clusters={delegatedRegistryClusters}
                                selectedClusterId={delegatedRegistryConfig.defaultClusterId}
                                handlePathChange={handlePathChange}
                                handleClusterChange={handleClusterChange}
                                addRegistryRow={addRegistryRow}
                                deleteRow={deleteRow}
                                key="delegated-registries-list"
                            />
                        </>
                    )}
                </Form>
                <Flex className="pf-v5-u-p-md">
                    <FlexItem align={{ default: 'alignLeft' }}>
                        <Flex>
                            <Button variant="primary" onClick={onSave}>
                                Save
                            </Button>
                            <Button variant="secondary" onClick={onCancel}>
                                Cancel
                            </Button>
                        </Flex>
                    </FlexItem>
                </Flex>
            </PageSection>
        </>
    );
}

export default DelegateScanningPage;
