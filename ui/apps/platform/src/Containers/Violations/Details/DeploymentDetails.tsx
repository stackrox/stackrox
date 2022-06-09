import React, { useEffect, useState } from 'react';
import {
    Alert,
    Flex,
    FlexItem,
    Card,
    CardBody,
    Title,
    Divider,
    List,
    ListItem,
} from '@patternfly/react-core';

import { fetchDeployment } from 'services/DeploymentsService';
import { fetchNetworkPoliciesInNamespace } from 'services/NetworkService';
import { portExposureLabels } from 'messages/common';
import ObjectDescriptionList from 'Components/ObjectDescriptionList';
import DeploymentOverview from './Deployment/DeploymentOverview';
import SecurityContext from './Deployment/SecurityContext';
import ContainerConfiguration from './Deployment/ContainerConfiguration';

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
                            All Network Policies for {deploymentObj.namespace}
                        </Title>
                        <Divider component="div" />
                    </FlexItem>
                    <FlexItem>
                        <Card isFlat data-testid="network-policy">
                            <CardBody>
                                {namespacePoliciesList?.length > 0 ? (
                                    <List>
                                        {namespacePoliciesList?.map((netpol: any) => (
                                            // eslint-disable-next-line react/no-array-index-key
                                            <ListItem key={netpol.id}>{netpol.name}</ListItem>
                                        ))}
                                    </List>
                                ) : (
                                    'No Network Policies found'
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
