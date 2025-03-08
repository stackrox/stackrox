import React, { useState, useEffect, ReactElement, ReactFragment } from 'react';
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
import usePermissions from 'hooks/usePermissions';
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
    body: ReactElement | ReactFragment;
};
const initialDelegatedState: DelegatedRegistryConfig = {
    enabledFor: 'NONE',
    defaultClusterId: '',
    registries: [],
};

function getUuid() {
    const MAX_RANDOM = 1000000;

    const uuid = self?.crypto?.randomUUID() ?? Math.floor(Math.random() * MAX_RANDOM).toString();

    return uuid;
}

function addUuidstoRegistries(config: DelegatedRegistryConfig) {
    const newRegistries = config.registries.map((registry) => {
        const uuid = getUuid();

        return {
            path: registry.path,
            clusterId: registry.clusterId,
            uuid,
        };
    });

    const newState: DelegatedRegistryConfig = {
        ...config,
        registries: newRegistries,
    };

    return newState;
}

function DelegateScanningPage() {
    const displayedPageTitle = 'Delegated Image Scanning';
    const [delegatedRegistryConfig, setDedicatedRegistryConfig] =
        useState<DelegatedRegistryConfig>(initialDelegatedState);
    const [delegatedRegistryClusters, setDelegatedRegistryClusters] = useState<
        DelegatedRegistryCluster[]
    >([]);
    const [alertObj, setAlertObj] = useState<AlertObj | null>(null);

    const [hasLoadedClusters, setHasLoadedClusters] = useState(false);
    const [isEditing, setIsEditing] = useState(false);

    const { hasReadWriteAccess } = usePermissions();
    const hasReadWriteAccessForAdministration = hasReadWriteAccess('Administration');

    useEffect(() => {
        fetchClusters();
        fetchConfig();
    }, []);

    function fetchConfig() {
        setAlertObj(null);
        fetchDelegatedRegistryConfig()
            .then((configFetched) => {
                const configWithUuids = addUuidstoRegistries(configFetched);
                setDedicatedRegistryConfig(configWithUuids);
            })
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
                setHasLoadedClusters(true);
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

    function onClickEdit() {
        setIsEditing(true);
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

        const uuid = getUuid();

        newState.registries.push({
            path: '',
            clusterId: '',
            uuid,
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
                    uuid: registry.uuid,
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
                    uuid: registry.uuid,
                };
            }

            return registry;
        });

        newState.registries = newRegistries;

        setDedicatedRegistryConfig(newState);
    }

    function updateRegistriesOrder(newRegistries) {
        const newState: DelegatedRegistryConfig = { ...delegatedRegistryConfig };

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
                setIsEditing(false);
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
        setIsEditing(false);
        fetchConfig();
    }

    return (
        <>
            <PageTitle title={displayedPageTitle} />
            <PageSection variant="light">
                <Flex direction={{ default: 'column' }}>
                    <FlexItem>
                        <Breadcrumb>
                            <BreadcrumbItemLink to={clustersBasePath}>Clusters</BreadcrumbItemLink>
                            <BreadcrumbItem isActive>{displayedPageTitle}</BreadcrumbItem>
                        </Breadcrumb>
                    </FlexItem>
                    <Flex>
                        <FlexItem flex={{ default: 'flex_1' }}>
                            <Title headingLevel="h1">{displayedPageTitle}</Title>
                        </FlexItem>
                        {hasReadWriteAccessForAdministration && (
                            <FlexItem align={{ default: 'alignRight' }}>
                                <Button
                                    variant="primary"
                                    isDisabled={
                                        delegatedRegistryConfig == null ||
                                        !hasLoadedClusters ||
                                        isEditing
                                    }
                                    onClick={onClickEdit}
                                >
                                    Edit
                                </Button>
                            </FlexItem>
                        )}
                    </Flex>
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
                        isEditing={isEditing}
                        onChangeEnabledFor={onChangeEnabledFor}
                    />
                    {/* TODO: decide who to structure this form, where the `enabledFor` value spans multiple inputs */}
                    {delegatedRegistryConfig.enabledFor !== 'NONE' && (
                        <>
                            <DelegatedScanningSettings
                                clusters={delegatedRegistryClusters}
                                isEditing={isEditing}
                                selectedClusterId={delegatedRegistryConfig.defaultClusterId}
                                setSelectedClusterId={onChangeDefaultCluster}
                            />
                            <DelegatedRegistriesList
                                registries={delegatedRegistryConfig.registries}
                                clusters={delegatedRegistryClusters}
                                selectedClusterId={delegatedRegistryConfig.defaultClusterId}
                                isEditing={isEditing}
                                handlePathChange={handlePathChange}
                                handleClusterChange={handleClusterChange}
                                addRegistryRow={addRegistryRow}
                                deleteRow={deleteRow}
                                // TODO: remove lint override after @typescript-eslint deps can be resolved to ^5.2.x
                                // eslint-disable-next-line @typescript-eslint/ban-ts-comment
                                // @ts-ignore
                                updateRegistriesOrder={updateRegistriesOrder}
                                key="delegated-registries-list"
                            />
                        </>
                    )}
                </Form>
                {isEditing && (
                    <Flex>
                        <FlexItem>
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
                )}
            </PageSection>
        </>
    );
}

export default DelegateScanningPage;
