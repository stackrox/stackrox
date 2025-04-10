import React, { useState, useEffect } from 'react';
import {
    Alert,
    Breadcrumb,
    BreadcrumbItem,
    Bullseye,
    Button,
    Divider,
    Flex,
    FlexItem,
    PageSection,
    Spinner,
    Title,
} from '@patternfly/react-core';

import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import PageTitle from 'Components/PageTitle';
import usePermissions from 'hooks/usePermissions';
import { clustersBasePath } from 'routePaths';
import {
    fetchDelegatedRegistryConfig,
    fetchDelegatedRegistryClusters,
    DelegatedRegistryConfig,
    DelegatedRegistryCluster,
} from 'services/DelegatedRegistryConfigService';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import DelegatedImageScanningForm from './DelegatedImageScanningForm';

function DelegateScanningPage() {
    const displayedPageTitle = 'Delegated image scanning';

    const [delegatedRegistryConfig, setDelegatedRegistryConfig] =
        useState<DelegatedRegistryConfig | null>(null);
    const [delegatedRegistryClusters, setDelegatedRegistryClusters] = useState<
        DelegatedRegistryCluster[]
    >([]);

    const [errorMessage, setErrorMessage] = useState('');
    const [isLoading, setIsLoading] = useState(false);

    const [isEditing, setIsEditing] = useState(false);

    const { hasReadWriteAccess } = usePermissions();
    const hasReadWriteAccessForAdministration = hasReadWriteAccess('Administration');

    useEffect(() => {
        setIsLoading(true);
        // Form requires responses from both requests.
        Promise.all([fetchDelegatedRegistryClusters(), fetchDelegatedRegistryConfig()])
            .then(([delegatedRegistryClustersFetched, delegatedRegistryConfigFetched]) => {
                setDelegatedRegistryClusters(delegatedRegistryClustersFetched);
                setDelegatedRegistryConfig(delegatedRegistryConfigFetched);
            })
            .catch((error) => {
                setDelegatedRegistryClusters([]);
                setDelegatedRegistryConfig(null);
                setErrorMessage(getAxiosErrorMessage(error));
            })
            .finally(() => {
                setIsLoading(false);
            });
    }, []);

    function onClickEdit() {
        setIsEditing(true);
    }

    function setIsNotEditing() {
        setIsEditing(false); // either Cancel or successful Save
    }

    /* eslint-disable no-nested-ternary */
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
                                    isDisabled={delegatedRegistryConfig === null || isEditing}
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
            <Divider component="div" />
            <PageSection variant="light">
                {isLoading ? (
                    <Bullseye>
                        <Spinner />
                    </Bullseye>
                ) : delegatedRegistryConfig === null ? (
                    <Alert
                        title="Unable to fetch clusters or configuration"
                        component="p"
                        variant="danger"
                        isInline
                    >
                        {errorMessage}
                    </Alert>
                ) : (
                    <DelegatedImageScanningForm
                        delegatedRegistryClusters={delegatedRegistryClusters}
                        delegatedRegistryConfig={delegatedRegistryConfig}
                        isEditing={isEditing}
                        setDelegatedRegistryConfig={setDelegatedRegistryConfig}
                        setIsNotEditing={setIsNotEditing}
                    />
                )}
            </PageSection>
        </>
    );
    /* eslint-enable no-nested-ternary */
}

export default DelegateScanningPage;
