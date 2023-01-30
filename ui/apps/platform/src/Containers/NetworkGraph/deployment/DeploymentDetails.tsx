import React, { useState } from 'react';
import {
    DescriptionList,
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
    Divider,
    EmptyState,
    EmptyStateVariant,
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
    Title,
} from '@patternfly/react-core';
import { ExclamationCircleIcon } from '@patternfly/react-icons';
import pluralize from 'pluralize';

import { Deployment } from 'types/deployment.proto';
import { ListenPort } from 'types/networkFlow.proto';
import { getDateTime } from 'utils/dateUtils';
import { NetworkPolicyState } from 'Containers/NetworkGraph/types/topology.type';

import DeploymentPortConfig from 'Components/DeploymentPortConfig';

import { ReactComponent as BothPolicyRules } from 'images/network-graph/both-policy-rules.svg';
import { ReactComponent as EgressOnly } from 'images/network-graph/egress-only.svg';
import { ReactComponent as IngressOnly } from 'images/network-graph/ingress-only.svg';
import { ReactComponent as NoPolicyRules } from 'images/network-graph/no-policy-rules.svg';

type DeploymentDetailsProps = {
    deployment: Deployment;
    numExternalFlows: number;
    numInternalFlows: number;
    listenPorts: ListenPort[];
    networkPolicyState: NetworkPolicyState;
};

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
    networkPolicyState,
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
                                    {networkPolicyState === 'both' && (
                                        <Label
                                            variant="outline"
                                            color="blue"
                                            icon={<BothPolicyRules width="22px" height="22px" />}
                                        >
                                            1 or more policies regulating bidirectional traffic
                                        </Label>
                                    )}
                                    {networkPolicyState === 'egress' && (
                                        <LabelGroup>
                                            <Label
                                                variant="outline"
                                                color="red"
                                                icon={<NoPolicyRules width="22px" height="22px" />}
                                            >
                                                A missing policy is allowing all ingress traffic
                                            </Label>
                                            <Label
                                                variant="outline"
                                                color="blue"
                                                icon={<EgressOnly width="22px" height="22px" />}
                                            >
                                                1 or more policies regulating egress traffic
                                            </Label>
                                        </LabelGroup>
                                    )}
                                    {networkPolicyState === 'ingress' && (
                                        <LabelGroup>
                                            <Label
                                                variant="outline"
                                                icon={<NoPolicyRules width="22px" height="22px" />}
                                            >
                                                A missing policy is allowing all egress traffic
                                            </Label>
                                            <Label
                                                variant="outline"
                                                color="blue"
                                                icon={<IngressOnly width="22px" height="22px" />}
                                            >
                                                1 or more policies regulating ingress traffic
                                            </Label>
                                        </LabelGroup>
                                    )}
                                    {networkPolicyState === 'none' && (
                                        <Label
                                            variant="outline"
                                            icon={<NoPolicyRules width="22px" height="22px" />}
                                        >
                                            A missing policy is allowing all network traffic
                                        </Label>
                                    )}
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
                                            {deployment.name}
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
                                            {deployment.clusterName}
                                        </DescriptionListDescription>
                                    </DescriptionListGroup>
                                    <DescriptionListGroup>
                                        <DescriptionListTerm>Namespace</DescriptionListTerm>
                                        <DescriptionListDescription>
                                            {deployment.namespace}
                                        </DescriptionListDescription>
                                    </DescriptionListGroup>
                                    <DescriptionListGroup>
                                        <DescriptionListTerm>Replicas</DescriptionListTerm>
                                        <DescriptionListDescription>
                                            {deployment.replicas}
                                        </DescriptionListDescription>
                                    </DescriptionListGroup>
                                    <DescriptionListGroup>
                                        <DescriptionListTerm>Service account</DescriptionListTerm>
                                        <DescriptionListDescription>
                                            {deployment.serviceAccount}
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
                        {deployment.ports.length ? (
                            <Stack hasGutter>
                                {deployment.ports.map((port) => {
                                    return (
                                        <StackItem>
                                            <DeploymentPortConfig port={port} />
                                        </StackItem>
                                    );
                                })}
                            </Stack>
                        ) : (
                            <EmptyState variant={EmptyStateVariant.xs}>
                                <Title headingLevel="h4" size="md">
                                    No ports available
                                </Title>
                            </EmptyState>
                        )}
                    </DetailSection>
                </li>
            </ul>
        </div>
    );
}

export default DeploymentDetails;
