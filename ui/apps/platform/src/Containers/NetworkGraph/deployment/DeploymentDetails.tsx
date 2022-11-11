import React, { useState } from 'react';
import {
    Button,
    DescriptionList,
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
    Divider,
    ExpandableSection,
    Flex,
    FlexItem,
    Label,
    LabelGroup,
    Stack,
    StackItem,
    Text,
    TextContent,
    TextVariants,
} from '@patternfly/react-core';
import { ExclamationCircleIcon } from '@patternfly/react-icons';
import pluralize from 'pluralize';

import { Deployment, PortConfig } from 'types/deployment.proto';
import { ListenPort } from 'types/networkFlow.proto';
import { getDateTime } from 'utils/dateUtils';

import DeploymentPortConfig from 'Components/DeploymentPortConfig';

type DeploymentDetailsProps = {
    deployment: Deployment;
    numExternalFlows: number;
    numInternalFlows: number;
    listenPorts: ListenPort[];
};

const mockPorts: PortConfig[] = [
    {
        name: 'api',
        containerPort: 8443,
        protocol: 'TCP',
        exposure: 'INTERNAL',
        exposedPort: 0,
        exposureInfos: [
            {
                level: 'INTERNAL',
                serviceName: 'sensor',
                serviceId: 'b830c97f-7ecf-49c7-898c-ea9c68f2e131',
                serviceClusterIp: '10.73.232.126',
                servicePort: 443,
                nodePort: 0,
                externalIps: [],
                externalHostnames: [],
            },
        ],
    },
    {
        name: 'webhook',
        containerPort: 9443,
        protocol: 'TCP',
        exposure: 'INTERNAL',
        exposedPort: 0,
        exposureInfos: [
            {
                level: 'INTERNAL',
                serviceName: 'sensor-webhook',
                serviceId: '3d879536-0521-4a52-9a8e-d33b4aa75ca5',
                serviceClusterIp: '10.73.232.21',
                servicePort: 443,
                nodePort: 0,
                externalIps: [],
                externalHostnames: [],
            },
        ],
    },
];

function DetailSection({ title, children }) {
    const [isExpanded, setIsExpanded] = useState(true);

    const onToggle = (_isExpanded: boolean) => {
        setIsExpanded(_isExpanded);
    };

    return (
        <ExpandableSection
            isExpanded={isExpanded}
            onToggle={onToggle}
            toggleContent={
                <TextContent>
                    <Text component={TextVariants.h1} className="pf-u-font-size-xl">
                        {title}
                    </Text>
                </TextContent>
            }
        >
            <div className="pf-u-px-sm pf-u-pb-md">{children}</div>
        </ExpandableSection>
    );
}

function DeploymentDetails({
    deployment,
    numExternalFlows,
    numInternalFlows,
    listenPorts,
}: DeploymentDetailsProps) {
    return (
        <div className="pf-u-h-100 pf-u-p-md">
            <ul>
                <li>
                    <DetailSection title="Network security">
                        <DescriptionList columnModifier={{ default: '1Col' }}>
                            <DescriptionListGroup>
                                <DescriptionListTerm>Anomalous traffic</DescriptionListTerm>
                                <DescriptionListDescription>
                                    <Flex
                                        direction={{ default: 'row' }}
                                        alignItems={{ default: 'alignItemsCenter' }}
                                    >
                                        <FlexItem>
                                            <Label
                                                variant="outline"
                                                color="red"
                                                icon={<ExclamationCircleIcon />}
                                            >
                                                {numExternalFlows} external{' '}
                                                {pluralize('flow', numExternalFlows)}
                                            </Label>
                                        </FlexItem>
                                        <FlexItem>
                                            <Label
                                                variant="outline"
                                                color="gold"
                                                icon={<ExclamationCircleIcon />}
                                            >
                                                {numInternalFlows} internal{' '}
                                                {pluralize('flow', numInternalFlows)}
                                            </Label>
                                        </FlexItem>
                                    </Flex>
                                </DescriptionListDescription>
                            </DescriptionListGroup>
                            <DescriptionListGroup>
                                <DescriptionListTerm>Network policy rules</DescriptionListTerm>
                                <DescriptionListDescription>
                                    <Flex
                                        direction={{ default: 'row' }}
                                        alignItems={{ default: 'alignItemsCenter' }}
                                    >
                                        <FlexItem>
                                            <Label
                                                variant="outline"
                                                color="gold"
                                                icon={<ExclamationCircleIcon />}
                                            >
                                                0 egress, allowing 325 flows
                                            </Label>
                                        </FlexItem>
                                        <FlexItem>
                                            <Label variant="outline" color="blue">
                                                1 egress
                                            </Label>
                                        </FlexItem>
                                    </Flex>
                                </DescriptionListDescription>
                            </DescriptionListGroup>
                            <DescriptionListGroup>
                                <DescriptionListTerm>Listening ports</DescriptionListTerm>
                                <DescriptionListDescription>
                                    <Flex
                                        direction={{ default: 'row' }}
                                        alignItems={{ default: 'alignItemsCenter' }}
                                    >
                                        <FlexItem>
                                            <LabelGroup>
                                                {listenPorts.map(({ port, l4protocol }) => {
                                                    const protocol = l4protocol.replace(
                                                        'L4_PROTOCOL_',
                                                        ''
                                                    );
                                                    return (
                                                        <Label variant="outline">
                                                            {protocol}: {port}
                                                        </Label>
                                                    );
                                                })}
                                            </LabelGroup>
                                        </FlexItem>
                                    </Flex>
                                </DescriptionListDescription>
                            </DescriptionListGroup>
                        </DescriptionList>
                    </DetailSection>
                </li>
                <Divider component="li" className="pf-u-mb-sm" />
                <li>
                    <DetailSection title="Deployment configuration">
                        <Stack hasGutter>
                            <StackItem>
                                <DescriptionList columnModifier={{ default: '2Col' }}>
                                    <DescriptionListGroup>
                                        <DescriptionListTerm>Name</DescriptionListTerm>
                                        <DescriptionListDescription>
                                            <Button variant="link" isInline>
                                                {deployment.name}
                                            </Button>
                                        </DescriptionListDescription>
                                    </DescriptionListGroup>
                                    <DescriptionListGroup>
                                        <DescriptionListTerm>Created</DescriptionListTerm>
                                        <DescriptionListDescription>
                                            {getDateTime(deployment.created)}
                                        </DescriptionListDescription>
                                    </DescriptionListGroup>
                                    <DescriptionListGroup>
                                        <DescriptionListTerm>Cluster</DescriptionListTerm>
                                        <DescriptionListDescription>
                                            <Button variant="link" isInline>
                                                {deployment.clusterName}
                                            </Button>
                                        </DescriptionListDescription>
                                    </DescriptionListGroup>
                                    <DescriptionListGroup>
                                        <DescriptionListTerm>Namespace</DescriptionListTerm>
                                        <DescriptionListDescription>
                                            <Button variant="link" isInline>
                                                {deployment.namespace}
                                            </Button>
                                        </DescriptionListDescription>
                                    </DescriptionListGroup>
                                    <DescriptionListGroup>
                                        <DescriptionListTerm>Replicas</DescriptionListTerm>
                                        <DescriptionListDescription>
                                            <Button variant="link" isInline>
                                                {deployment.replicas}
                                            </Button>
                                        </DescriptionListDescription>
                                    </DescriptionListGroup>
                                    <DescriptionListGroup>
                                        <DescriptionListTerm>Service account</DescriptionListTerm>
                                        <DescriptionListDescription>
                                            <Button variant="link" isInline>
                                                {deployment.serviceAccount}
                                            </Button>
                                        </DescriptionListDescription>
                                    </DescriptionListGroup>
                                </DescriptionList>
                            </StackItem>
                            <StackItem>
                                <DescriptionList columnModifier={{ default: '1Col' }}>
                                    <DescriptionListGroup>
                                        <DescriptionListTerm>Labels</DescriptionListTerm>
                                        <DescriptionListDescription>
                                            <LabelGroup>
                                                {Object.keys(deployment.labels).map((labelKey) => {
                                                    const labelValue = deployment.labels[labelKey];
                                                    const label = `${labelKey}:${labelValue}`;
                                                    return (
                                                        <Label key={label} color="blue">
                                                            {label}
                                                        </Label>
                                                    );
                                                })}
                                            </LabelGroup>
                                        </DescriptionListDescription>
                                    </DescriptionListGroup>
                                    <DescriptionListGroup>
                                        <DescriptionListTerm>Annotations</DescriptionListTerm>
                                        <DescriptionListDescription>
                                            <LabelGroup>
                                                {Object.keys(deployment.annotations).map(
                                                    (annotationKey) => {
                                                        const annotationValue =
                                                            deployment.annotations[annotationKey];
                                                        const annotation = `${annotationKey}:${annotationValue}`;
                                                        return (
                                                            <Label key={annotationKey} color="blue">
                                                                {annotation}
                                                            </Label>
                                                        );
                                                    }
                                                )}
                                            </LabelGroup>
                                        </DescriptionListDescription>
                                    </DescriptionListGroup>
                                </DescriptionList>
                            </StackItem>
                        </Stack>
                    </DetailSection>
                </li>
                <Divider component="li" className="pf-u-mb-sm" />
                <li>
                    <DetailSection title="Port configurations">
                        <Stack hasGutter>
                            {mockPorts.map((port) => {
                                return (
                                    <StackItem>
                                        <DeploymentPortConfig port={port} />
                                    </StackItem>
                                );
                            })}
                        </Stack>
                    </DetailSection>
                </li>
            </ul>
        </div>
    );
}

export default DeploymentDetails;
