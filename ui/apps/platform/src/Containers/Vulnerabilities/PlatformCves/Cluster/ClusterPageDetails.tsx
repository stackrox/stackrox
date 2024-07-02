/* eslint-disable no-nested-ternary */
import React from 'react';
import {
    Bullseye,
    Card,
    CardBody,
    DescriptionList,
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
    Flex,
    PageSection,
    Spinner,
    Text,
} from '@patternfly/react-core';
import { ExclamationCircleIcon } from '@patternfly/react-icons';

import EmptyStateTemplate from 'Components/EmptyStateTemplate';
import { getDateTime } from 'utils/dateUtils';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import ExpandableLabelSection from '../../components/ExpandableLabelSection';
import useClusterExtendedDetails, { ProviderMetadata } from './useClusterExtendedDetails';
import { displayClusterType } from '../utils/stringUtils';

function getCloudProviderText(providerMetadata: ProviderMetadata | undefined): string | null {
    if (!providerMetadata) {
        return null;
    }
    const { region } = providerMetadata;
    if (providerMetadata.aws) {
        return `AWS ${region}`;
    }
    if (providerMetadata.azure) {
        return `Azure ${region}`;
    }
    if (providerMetadata.google) {
        return `GCP ${region}`;
    }
    return null;
}

export type ClusterPageDetailsProps = {
    clusterId: string;
};

function ClusterPageDetails({ clusterId }: ClusterPageDetailsProps) {
    const { data, loading, error } = useClusterExtendedDetails(clusterId);

    return (
        <>
            <PageSection component="div" variant="light" className="pf-v5-u-py-md pf-v5-u-px-xl">
                <Text>View details about this cluster</Text>
            </PageSection>
            <PageSection isFilled className="pf-v5-u-display-flex pf-v5-u-flex-direction-column">
                <Card>
                    <CardBody>
                        {error ? (
                            <Bullseye>
                                <EmptyStateTemplate
                                    title="There was an error loading the cluster details"
                                    headingLevel="h2"
                                    icon={ExclamationCircleIcon}
                                    iconClassName="pf-v5-u-danger-color-100"
                                >
                                    {getAxiosErrorMessage(error)}
                                </EmptyStateTemplate>
                            </Bullseye>
                        ) : loading ? (
                            <Bullseye>
                                <Spinner size="xl" />
                            </Bullseye>
                        ) : (
                            data && (
                                <Flex
                                    direction={{ default: 'column' }}
                                    spaceItems={{ default: 'spaceItemsXl' }}
                                >
                                    <DescriptionList columnModifier={{ default: '1Col' }}>
                                        <DescriptionListGroup>
                                            <DescriptionListTerm>Cluster type</DescriptionListTerm>
                                            <DescriptionListDescription>
                                                {displayClusterType(data.cluster.type)}
                                            </DescriptionListDescription>
                                        </DescriptionListGroup>
                                        {getCloudProviderText(
                                            data.cluster.status?.providerMetadata
                                        ) && (
                                            <DescriptionListGroup>
                                                <DescriptionListTerm>
                                                    Cloud provider
                                                </DescriptionListTerm>
                                                <DescriptionListDescription>
                                                    {getCloudProviderText(
                                                        data.cluster.status?.providerMetadata
                                                    )}
                                                </DescriptionListDescription>
                                            </DescriptionListGroup>
                                        )}
                                        {data.cluster.status?.orchestratorMetadata?.buildDate && (
                                            <DescriptionListGroup>
                                                <DescriptionListTerm>
                                                    Build date
                                                </DescriptionListTerm>
                                                <DescriptionListDescription>
                                                    {getDateTime(
                                                        data.cluster.status.orchestratorMetadata
                                                            .buildDate
                                                    )}
                                                </DescriptionListDescription>
                                            </DescriptionListGroup>
                                        )}
                                        {data.cluster.status?.orchestratorMetadata?.version && (
                                            <DescriptionListGroup>
                                                <DescriptionListTerm>
                                                    K8s version
                                                </DescriptionListTerm>
                                                <DescriptionListDescription>
                                                    {
                                                        data.cluster.status.orchestratorMetadata
                                                            .version
                                                    }
                                                </DescriptionListDescription>
                                            </DescriptionListGroup>
                                        )}
                                    </DescriptionList>
                                    <ExpandableLabelSection
                                        toggleText="Labels"
                                        labels={data.cluster.labels}
                                    />
                                </Flex>
                            )
                        )}
                    </CardBody>
                </Card>
            </PageSection>
        </>
    );
}

export default ClusterPageDetails;
