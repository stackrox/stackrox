import React, { useState, useEffect, ReactElement, ReactFragment } from 'react';
import {
    Alert,
    AlertVariant,
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
    type: AlertVariant.danger | AlertVariant.success;
    title: string;
    body: ReactElement | ReactFragment;
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
            .then((configFetched) => {
                setDedicatedRegistryConfig(configFetched);
            })
            .catch((error) => {
                const newErrorObj: AlertObj = {
                    type: AlertVariant.danger,
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
                    type: AlertVariant.danger,
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

    function toggleDelegation() {
        const newState: DelegatedRegistryConfig = { ...delegatedRegistryConfig };

        if (delegatedRegistryConfig.enabledFor === 'NONE') {
            if (delegatedRegistryConfig.registries.length > 0) {
                newState.enabledFor = 'SPECIFIC';
            } else {
                newState.enabledFor = 'ALL';
            }
        } else {
            newState.enabledFor = 'NONE';
        }
        setDedicatedRegistryConfig(newState);
    }

    function onChangeEnabledFor(newEnabledState) {
        const newState: DelegatedRegistryConfig = { ...delegatedRegistryConfig };

        newState.enabledFor = newEnabledState;

        setDedicatedRegistryConfig(newState);
    }

    function onChangeCluster(newClusterId) {
        const newState: DelegatedRegistryConfig = { ...delegatedRegistryConfig };

        newState.defaultClusterId = newClusterId;

        setDedicatedRegistryConfig(newState);
    }

    function onSave() {
        setAlertObj(null);
        updateDelegatedRegistryConfig(delegatedRegistryConfig)
            .then(() => {
                const newSuccessObj: AlertObj = {
                    type: AlertVariant.success,
                    title: 'Delegated scanning configuration saved successfully',
                    body: <></>,
                };
                setAlertObj(newSuccessObj);
            })
            .catch((error) => {
                const newErrorObj: AlertObj = {
                    type: AlertVariant.danger,
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
                <Flex direction={{ default: 'column' }} className="pf-u-py-lg pf-u-pl-lg">
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
                        variant={alertObj.type}
                        isInline
                        className="pf-u-mb-lg"
                        component="h2"
                    >
                        {alertObj.body}
                    </Alert>
                )}
                <Form>
                    <ToggleDelegatedScanning
                        enabledFor={delegatedRegistryConfig.enabledFor}
                        toggleDelegation={toggleDelegation}
                    />
                    {/* TODO: decide who to structure this form, where the `enabledFor` value spans multiple inputs */}
                    {delegatedRegistryConfig.enabledFor !== 'NONE' && (
                        <>
                            <DelegatedScanningSettings
                                enabledFor={delegatedRegistryConfig.enabledFor}
                                onChangeEnabledFor={onChangeEnabledFor}
                                clusters={delegatedRegistryClusters}
                                selectedClusterId={delegatedRegistryConfig.defaultClusterId}
                                setSelectedClusterId={onChangeCluster}
                            />
                            <DelegatedRegistriesList
                                registries={delegatedRegistryConfig.registries}
                            />
                        </>
                    )}
                </Form>
                <Flex className="pf-u-p-md">
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
