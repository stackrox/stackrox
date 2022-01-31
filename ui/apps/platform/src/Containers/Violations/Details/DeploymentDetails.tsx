import React, { useEffect, useState } from 'react';
import { Alert, Flex, FlexItem, Card, CardBody, Title } from '@patternfly/react-core';

import { fetchDeployment } from 'services/DeploymentsService';
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

    useEffect(() => {
        fetchDeployment(deployment.id).then(
            (dep) => setRelatedDeployment(dep),
            () => setRelatedDeployment(null)
        );
    }, [deployment.id, setRelatedDeployment]);

    const deploymentObj = relatedDeployment || deployment;

    return (
        <Flex
            className="pf-u-mt-md"
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
                        <Title headingLevel="h3">Overview</Title>
                    </FlexItem>
                    <FlexItem>
                        <Card isFlat data-testid="deployment-overview">
                            <CardBody>
                                <DeploymentOverview deployment={deploymentObj} />
                            </CardBody>
                        </Card>
                    </FlexItem>
                    <FlexItem>
                        <Title headingLevel="h3">Port configuration</Title>
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
                        <Title headingLevel="h3">Security context</Title>
                    </FlexItem>
                    <FlexItem>
                        <SecurityContext deployment={relatedDeployment} />
                    </FlexItem>
                </Flex>
                <Flex direction={{ default: 'column' }} flex={{ default: 'flex_1' }}>
                    <FlexItem>
                        <Title headingLevel="h3">Container configuration</Title>
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
