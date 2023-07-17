import React, { useEffect, useState } from 'react';
import {
    Alert,
    Button,
    Flex,
    FlexItem,
    Card,
    CardBody,
    Title,
    Divider,
} from '@patternfly/react-core';
import { TableComposable, Caption, Thead, Tr, Th, Tbody, Td } from '@patternfly/react-table';

import { fetchNetworkPoliciesInNamespace } from 'services/NetworkService';
import { portExposureLabels } from 'messages/common';
import ObjectDescriptionList from 'Components/ObjectDescriptionList';
import useFetchDeployment from 'hooks/useFetchDeployment';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import { NetworkPolicy } from 'types/networkPolicy.proto';
import { Alert as AlertViolation } from '../types/violationTypes';
import DeploymentOverview from './Deployment/DeploymentOverview';
import SecurityContext from './Deployment/SecurityContext';
import ContainerConfiguration from './Deployment/ContainerConfiguration';
import NetworkPolicyModal from './NetworkPolicyModal';

type PortExposure = 'EXTERNAL' | 'NODE' | 'HOST' | 'INTERNAL' | 'UNSET';

type Port = {
    exposure: PortExposure;
    exposureInfos: {
        externalHostnames: string[];
        externalIps: string[];
        level: PortExposure;
        nodePort: number;
        serviceClusterIp: string;
        serviceId: string;
        serviceName: string;
        servicePort: number;
    }[];
    containerPort: number;
    exposedPort: number;
    name: string;
    protocol: string;
};

type FormattedPort = {
    exposure: string;
    exposureInfos: {
        externalHostnames: string[];
        externalIps: string[];
        level: string;
        nodePort: number;
        serviceClusterIp: string;
        serviceId: string;
        serviceName: string;
        servicePort: number;
    }[];
    containerPort: number;
    exposedPort: number;
    name: string;
    protocol: string;
};

export const formatDeploymentPorts = (ports: Port[] = []): FormattedPort[] => {
    const formattedPorts = [] as FormattedPort[];
    ports.forEach(({ exposure, exposureInfos, ...rest }) => {
        const formattedPort = { ...rest } as FormattedPort;
        formattedPort.exposure = portExposureLabels[exposure] || portExposureLabels.UNSET;
        formattedPort.exposureInfos = exposureInfos.map(({ level, ...restInfo }) => {
            return { ...restInfo, level: portExposureLabels[level] };
        });
        formattedPorts.push(formattedPort);
    });
    return formattedPorts;
};

const compareNetworkPolicies = (a: NetworkPolicy, b: NetworkPolicy): number => {
    return a.name.localeCompare(b.name);
};

export type DeploymentDetailsProps = {
    alertDeployment: Pick<
        NonNullable<AlertViolation['deployment']>,
        'id' | 'clusterId' | 'namespace'
    >;
};

const DeploymentDetails = ({ alertDeployment }: DeploymentDetailsProps) => {
    const [namespacePolicies, setNamespacePolicies] = useState<NetworkPolicy[]>([]);
    const [selectedNetworkPolicy, setSelectedNetworkPolicy] = useState<NetworkPolicy | null>(null);

    // attempt to fetch related deployment to selected alert
    const { deployment: relatedDeployment, error: relatedDeploymentFetchError } =
        useFetchDeployment(alertDeployment.id);

    useEffect(() => {
        fetchNetworkPoliciesInNamespace(alertDeployment.clusterId, alertDeployment.namespace).then(
            // TODO Infer type from response once NetworkService.js is typed
            (policies: NetworkPolicy[]) => setNamespacePolicies(policies ?? []),
            () => setNamespacePolicies([])
        );
    }, [alertDeployment.namespace, alertDeployment.clusterId, setNamespacePolicies]);

    const relatedDeploymentPorts = relatedDeployment?.ports || [];

    return (
        <Flex
            direction={{ default: 'column' }}
            flex={{ default: 'flex_1' }}
            aria-label="Deployment details"
        >
            {!relatedDeployment && relatedDeploymentFetchError && (
                <Alert
                    variant="warning"
                    isInline
                    title="There was an error fetching the deployment details. This deployment may no longer exist."
                >
                    {getAxiosErrorMessage(relatedDeploymentFetchError)}
                </Alert>
            )}
            <Flex flex={{ default: 'flex_1' }}>
                <Flex direction={{ default: 'column' }} flex={{ default: 'flex_1' }}>
                    <FlexItem>
                        <Title headingLevel="h3" className="pf-u-mb-md">
                            Overview
                        </Title>
                        <Divider component="div" />
                    </FlexItem>
                    <FlexItem>
                        <Card isFlat aria-label="Deployment overview">
                            <CardBody>
                                {relatedDeployment && (
                                    <DeploymentOverview deployment={relatedDeployment} />
                                )}
                            </CardBody>
                        </Card>
                    </FlexItem>
                    <FlexItem>
                        <Title headingLevel="h3" className="pf-u-my-md">
                            Port configuration
                        </Title>
                        <Divider component="div" />
                    </FlexItem>
                    <FlexItem>
                        <Card isFlat aria-label="Port configuration">
                            <CardBody>
                                {relatedDeploymentPorts.length > 0
                                    ? formatDeploymentPorts(relatedDeploymentPorts).map(
                                          (port, idx) => (
                                              // eslint-disable-next-line react/no-array-index-key
                                              <ObjectDescriptionList data={port} key={idx} />
                                          )
                                      )
                                    : 'None'}
                            </CardBody>
                        </Card>
                    </FlexItem>
                    <FlexItem>
                        <Title headingLevel="h3" className="pf-u-my-md">
                            Security context
                        </Title>
                        <Divider component="div" />
                    </FlexItem>
                    <FlexItem>
                        <SecurityContext deployment={relatedDeployment} />
                    </FlexItem>
                    <FlexItem>
                        <Title headingLevel="h3" className="pf-u-my-md">
                            Network policy
                        </Title>
                        <Divider component="div" />
                    </FlexItem>
                    <FlexItem>
                        <Card isFlat aria-label="Network policies in namespace">
                            <CardBody>
                                {namespacePolicies.length > 0 ? (
                                    <TableComposable variant="compact">
                                        <Caption>
                                            <Title headingLevel="h3">All network policies</Title>
                                            in &quot;{alertDeployment.namespace}&quot; namespace
                                        </Caption>
                                        <Thead>
                                            <Tr>
                                                <Th>Name</Th>
                                            </Tr>
                                        </Thead>
                                        <Tbody>
                                            {selectedNetworkPolicy && (
                                                <NetworkPolicyModal
                                                    networkPolicy={selectedNetworkPolicy}
                                                    isOpen={selectedNetworkPolicy !== null}
                                                    onClose={() => setSelectedNetworkPolicy(null)}
                                                />
                                            )}
                                            {namespacePolicies
                                                .sort(compareNetworkPolicies)
                                                .map((netpol: NetworkPolicy) => (
                                                    <Tr key={netpol.id}>
                                                        <Td dataLabel="Name">
                                                            <Button
                                                                variant="link"
                                                                onClick={() =>
                                                                    setSelectedNetworkPolicy(netpol)
                                                                }
                                                            >
                                                                {netpol.name}
                                                            </Button>
                                                        </Td>
                                                    </Tr>
                                                ))}
                                        </Tbody>
                                    </TableComposable>
                                ) : (
                                    <>
                                        No network policies found in &quot;
                                        {alertDeployment.namespace}&quot; namespace
                                    </>
                                )}
                            </CardBody>
                        </Card>
                    </FlexItem>
                </Flex>
                <Flex direction={{ default: 'column' }} flex={{ default: 'flex_1' }}>
                    <FlexItem>
                        <Title headingLevel="h3" className="pf-u-mb-md">
                            Container configuration
                        </Title>
                        <Divider component="div" />
                    </FlexItem>
                    <FlexItem>
                        <ContainerConfiguration deployment={relatedDeployment} />
                    </FlexItem>
                </Flex>
            </Flex>
        </Flex>
    );
};

export default DeploymentDetails;
