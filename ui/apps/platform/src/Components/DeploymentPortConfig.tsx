import React, { useState } from 'react';
import {
    Card,
    CardBody,
    DescriptionList,
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
    ExpandableSection,
    Stack,
    StackItem,
} from '@patternfly/react-core';

import { PortConfig } from 'types/deployment.proto';

type DeploymentPortConfigProps = {
    port: PortConfig;
};

function DeploymentPortConfig({ port }: DeploymentPortConfigProps) {
    const [isExpanded, setIsExpanded] = useState(false);

    const onToggle = (_isExpanded: boolean) => {
        setIsExpanded(_isExpanded);
    };

    const toggleText = port.name
        ? `${port.name} — ${port.containerPort}/${port.protocol}`
        : `${port.containerPort}/${port.protocol}`;

    return (
        <ExpandableSection
            toggleText={toggleText}
            onToggle={onToggle}
            isExpanded={isExpanded}
            displaySize="large"
            isWidthLimited
        >
            <Stack hasGutter>
                <StackItem>
                    <DescriptionList columnModifier={{ default: '2Col' }} isCompact>
                        <DescriptionListGroup>
                            <DescriptionListTerm>Container port</DescriptionListTerm>
                            <DescriptionListDescription>
                                {port.containerPort}
                            </DescriptionListDescription>
                        </DescriptionListGroup>
                        <DescriptionListGroup>
                            <DescriptionListTerm>Protocol</DescriptionListTerm>
                            <DescriptionListDescription>{port.protocol}</DescriptionListDescription>
                        </DescriptionListGroup>
                        <DescriptionListGroup>
                            <DescriptionListTerm>Exposure</DescriptionListTerm>
                            <DescriptionListDescription>{port.exposure}</DescriptionListDescription>
                        </DescriptionListGroup>
                        <DescriptionListGroup>
                            <DescriptionListTerm>Exposed port</DescriptionListTerm>
                            <DescriptionListDescription>
                                {port.exposedPort}
                            </DescriptionListDescription>
                        </DescriptionListGroup>
                    </DescriptionList>
                </StackItem>
                <StackItem>
                    <Stack hasGutter>
                        {port.exposureInfos.map((exposureInfo) => {
                            return (
                                <StackItem key={exposureInfo.serviceId}>
                                    <Card isFlat>
                                        <CardBody>
                                            <DescriptionList
                                                columnModifier={{ default: '2Col' }}
                                                isCompact
                                            >
                                                <DescriptionListGroup>
                                                    <DescriptionListTerm>Level</DescriptionListTerm>
                                                    <DescriptionListDescription>
                                                        {exposureInfo.level}
                                                    </DescriptionListDescription>
                                                </DescriptionListGroup>
                                                <DescriptionListGroup>
                                                    <DescriptionListTerm>
                                                        Service Name
                                                    </DescriptionListTerm>
                                                    <DescriptionListDescription>
                                                        {exposureInfo.serviceName}
                                                    </DescriptionListDescription>
                                                </DescriptionListGroup>
                                                <DescriptionListGroup>
                                                    <DescriptionListTerm>
                                                        Service ID
                                                    </DescriptionListTerm>
                                                    <DescriptionListDescription>
                                                        {exposureInfo.serviceId}
                                                    </DescriptionListDescription>
                                                </DescriptionListGroup>
                                                <DescriptionListGroup>
                                                    <DescriptionListTerm>
                                                        Service Cluster IP
                                                    </DescriptionListTerm>
                                                    <DescriptionListDescription>
                                                        {exposureInfo.serviceClusterIp}
                                                    </DescriptionListDescription>
                                                </DescriptionListGroup>
                                                <DescriptionListGroup>
                                                    <DescriptionListTerm>
                                                        Service Port
                                                    </DescriptionListTerm>
                                                    <DescriptionListDescription>
                                                        {exposureInfo.servicePort}
                                                    </DescriptionListDescription>
                                                </DescriptionListGroup>
                                                <DescriptionListGroup>
                                                    <DescriptionListTerm>
                                                        Node Port
                                                    </DescriptionListTerm>
                                                    <DescriptionListDescription>
                                                        {exposureInfo.nodePort}
                                                    </DescriptionListDescription>
                                                </DescriptionListGroup>
                                                <DescriptionListGroup>
                                                    <DescriptionListTerm>
                                                        External IPs
                                                    </DescriptionListTerm>
                                                    <DescriptionListDescription>
                                                        -
                                                    </DescriptionListDescription>
                                                </DescriptionListGroup>
                                                <DescriptionListGroup>
                                                    <DescriptionListTerm>
                                                        External Hostnames
                                                    </DescriptionListTerm>
                                                    <DescriptionListDescription>
                                                        -
                                                    </DescriptionListDescription>
                                                </DescriptionListGroup>
                                            </DescriptionList>
                                        </CardBody>
                                    </Card>
                                </StackItem>
                            );
                        })}
                    </Stack>
                </StackItem>
            </Stack>
        </ExpandableSection>
    );
}

export default DeploymentPortConfig;
