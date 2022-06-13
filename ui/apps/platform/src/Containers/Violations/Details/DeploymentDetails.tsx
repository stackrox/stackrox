import React, { useEffect, useState } from 'react';
import { Alert, Flex, FlexItem, Card, CardBody, Title, Divider } from '@patternfly/react-core';

import { fetchDeployment } from 'services/DeploymentsService';
import { fetchNetworkPoliciesInNamespace } from 'services/NetworkService';
import { portExposureLabels } from 'messages/common';
import ObjectDescriptionList from 'Components/ObjectDescriptionList';
import { TableComposable, Caption, Thead, Tr, Th, Tbody, Td } from '@patternfly/react-table';
import DeploymentOverview from './Deployment/DeploymentOverview';
import SecurityContext from './Deployment/SecurityContext';
import ContainerConfiguration from './Deployment/ContainerConfiguration';

type NetworkPolicy = {
    id: string;
    name: string;
};

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

const sortNetworkPolicies = (a: NetworkPolicy, b: NetworkPolicy): number => {
    return a.name.localeCompare(b.name);
};

const DeploymentDetails = ({ deployment }) => {
    // attempt to fetch related deployment to selected alert
    const [relatedDeployment, setRelatedDeployment] = useState(deployment);
    const [namespacePolicies, setNamespacePolicies] = useState([]);

    useEffect(() => {
        fetchDeployment(deployment.id).then(
            (dep) => setRelatedDeployment(dep),
            () => setRelatedDeployment(null)
        );
    }, [deployment.id, setRelatedDeployment]);

    useEffect(() => {
        fetchNetworkPoliciesInNamespace(deployment.clusterId, deployment.namespace).then(
            (policies) => setNamespacePolicies(policies.response.networkPolicies),
            () => setNamespacePolicies([])
        );
    }, [deployment.namespace, deployment.clusterId, setNamespacePolicies]);

    const deploymentObj = relatedDeployment || deployment;
    const namespacePoliciesList = namespacePolicies || [];

    return (
        <Flex
            direction={{ default: 'column' }}
            flex={{ default: 'flex_1' }}
            data-testid="deployment-details"
        >
            {!relatedDeployment && (
                <Alert
                    variant="warning"
                    isInline
                    title="This data is a snapshot of a deployment that no longer exists."
                    data-testid="deployment-snapshot-warning"
                />
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
                        <Card isFlat data-testid="deployment-overview">
                            <CardBody>
                                <DeploymentOverview deployment={deploymentObj} />
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
                        <Card isFlat data-testid="port-configuration">
                            <CardBody>
                                {deploymentObj?.ports?.length > 0
                                    ? formatDeploymentPorts(deploymentObj.ports).map(
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
                            Network Policy
                        </Title>
                        <Divider component="div" />
                    </FlexItem>
                    <FlexItem>
                        <Card isFlat data-testid="network-policy">
                            <CardBody>
                                {namespacePoliciesList?.length > 0 ? (
                                    <TableComposable aria-label="Simple table" variant="compact">
                                        <Caption>
                                            <Title headingLevel="h3">
                                                All network policies of deployment
                                            </Title>
                                            <br />
                                            in {deploymentObj.namespace} namespace
                                        </Caption>
                                        <Thead>
                                            <Tr>
                                                <Th>Name</Th>
                                            </Tr>
                                        </Thead>
                                        <Tbody>
                                            {namespacePoliciesList
                                                .sort(sortNetworkPolicies)
                                                .map((netpol: NetworkPolicy) => (
                                                    // TODO(ROX-11034): This should be a link to the Network Policy yaml or detail screen. 
                                                    <Tr key={netpol.id}>
                                                        <Td dataLabel="Name">{netpol.name}</Td>
                                                    </Tr>
                                                ))}
                                        </Tbody>
                                    </TableComposable>
                                ) : (
                                    'No network policies found'
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
