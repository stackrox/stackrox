import React, { useState, useEffect } from 'react';
import {
    Alert,
    AlertVariant,
    Breadcrumb,
    BreadcrumbItem,
    Card,
    CardBody,
    Flex,
    FlexItem,
    PageSection,
    Radio,
    Title,
} from '@patternfly/react-core';

import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import FormLabelGroup from 'Components/PatternFly/FormLabelGroup';
import PageTitle from 'Components/PageTitle';
import { clustersBasePath } from 'routePaths';
import { fetchDelegatedRegistryConfig } from 'services/DelegatedRegistryConfigService';
import { DelegatedRegistryConfig } from 'types/dedicatedRegistryConfig.proto';
import ToggleDelegatedScanning from './Components/ToggleDelegatedScanning';

const initialDelegatedState: DelegatedRegistryConfig = {
    enabledFor: 'NONE',
    defaultClusterId: '',
    registries: [],
};

function DelegateScanningPage() {
    const displayedPageTitle = 'Delegate Image Scanning';
    const [delegatedRegistryConfig, setDedicatedRegistryConfig] =
        useState<DelegatedRegistryConfig>(initialDelegatedState);
    const [errMessage, setErrMessage] = useState<string | null>(null);

    useEffect(() => {
        fetchDelegatedRegistryConfig()
            .then((result) => {
                setDedicatedRegistryConfig(result.response);
            })
            .catch((err) => {
                setErrMessage(err.message);
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
                    <Card>
                        <CardBody>
                            <FormLabelGroup
                                label="Delegate scanning for"
                                isRequired
                                fieldId="delegatedRegistryConfig.enabledFor"
                                touched={{}}
                                errors={{}}
                            >
                                <Flex className="pf-u-mt-md pf-u-mb-lg">
                                    <FlexItem>
                                        <Radio
                                            label="All registries"
                                            isChecked={delegatedRegistryConfig.enabledFor === 'ALL'}
                                            id="choose-all-registries"
                                            name="enabledFor"
                                            onChange={() => {}}
                                        />
                                    </FlexItem>
                                    <FlexItem>
                                        <Radio
                                            label="Specified registries"
                                            isChecked={
                                                delegatedRegistryConfig.enabledFor === 'SPECIFIC'
                                            }
                                            id="chose-specified-registries"
                                            name="enabledFor"
                                            onChange={() => {}}
                                        />
                                    </FlexItem>
                                </Flex>
                            </FormLabelGroup>
                        </CardBody>
                    </Card>
                )}
            </PageSection>
        </>
    );
}

export default DelegateScanningPage;
