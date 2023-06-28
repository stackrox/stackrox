import React, { useState, useEffect } from 'react';
import {
    Alert,
    AlertVariant,
    Breadcrumb,
    BreadcrumbItem,
    Flex,
    FlexItem,
    PageSection,
    Title,
} from '@patternfly/react-core';

import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import PageTitle from 'Components/PageTitle';
import { clustersBasePath } from 'routePaths';
import useFetchClustersForPermissions from 'hooks/useFetchClustersForPermissions';
import { fetchDelegatedRegistryConfig } from 'services/DelegatedRegistryConfigService';
import { DelegatedRegistryConfig } from 'types/dedicatedRegistryConfig.proto';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import ToggleDelegatedScanning from './Components/ToggleDelegatedScanning';
import DelegatedScanningSettings from './Components/DelegatedScanningSettings';
import DelegatedRegistriesList from './Components/DelegatedRegistriesList';

const initialDelegatedState: DelegatedRegistryConfig = {
    enabledFor: 'NONE',
    defaultClusterId: '',
    registries: [],
};

function DelegateScanningPage() {
    const displayedPageTitle = 'Delegate Image Scanning';
    const [delegatedRegistryConfig, setDedicatedRegistryConfig] =
        useState<DelegatedRegistryConfig>(initialDelegatedState);
    const [errMessage, setErrMessage] = useState<string>('');

    const { clusters } = useFetchClustersForPermissions(['Deployment']);

    useEffect(() => {
        setErrMessage('');
        fetchDelegatedRegistryConfig()
            .then((configFetched) => {
                setDedicatedRegistryConfig(configFetched);
            })
            .catch((error) => {
                setErrMessage(getAxiosErrorMessage(error));
                setDedicatedRegistryConfig(initialDelegatedState);
            });
    }, []);

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
                {!!errMessage && (
                    <Alert
                        title="Problem retrieving the delegated scanning configuration from the server"
                        variant={AlertVariant.danger}
                        isInline
                        className="pf-u-mb-lg"
                    >
                        <p>{errMessage}</p>
                        <p>Try reloading the page. If this problem persists, contact support.</p>
                    </Alert>
                )}
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
                            clusters={clusters}
                            selectedClusterId={delegatedRegistryConfig.defaultClusterId}
                            setSelectedClusterId={onChangeCluster}
                        />
                        <DelegatedRegistriesList registries={delegatedRegistryConfig.registries} />
                    </>
                )}
            </PageSection>
        </>
    );
}

export default DelegateScanningPage;
