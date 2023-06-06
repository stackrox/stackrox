import React, { useState } from 'react';
import {
    Button,
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
import { ExclamationCircleIcon, ExclamationTriangleIcon } from '@patternfly/react-icons';
import pluralize from 'pluralize';

import { Deployment } from 'types/deployment.proto';
import { ListenPort } from 'types/networkFlow.proto';
import { getDateTime } from 'utils/dateUtils';
import { NetworkPolicyState } from 'Containers/NetworkGraph/types/topology.type';

import DeploymentPortConfig from 'Components/DeploymentPortConfig';
import DeploymentContainerConfig from 'Components/DeploymentContainerConfig';

import { ReactComponent as BothPolicyRules } from 'images/network-graph/both-policy-rules.svg';
import { ReactComponent as EgressOnly } from 'images/network-graph/egress-only.svg';
import { ReactComponent as IngressOnly } from 'images/network-graph/ingress-only.svg';
import { ReactComponent as NoPolicyRules } from 'images/network-graph/no-policy-rules.svg';
import { deploymentTabs } from '../utils/deploymentUtils';

import './DeploymentDetails.css';

type DeploymentDetailsProps = {
    deployment: Deployment;
    numAnomalousExternalFlows: number;
    numAnomalousInternalFlows: number;
    listenPorts: ListenPort[];
    networkPolicyState: NetworkPolicyState;
    onDeploymentTabsSelect: (tab: string) => void;
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
                    <Text component={TextVariants.h1} className="pf-u-font-size-lg">
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
    numAnomalousExternalFlows,
    numAnomalousInternalFlows,
    listenPorts,
    networkPolicyState,
    onDeploymentTabsSelect,
}: DeploymentDetailsProps) {
    const labelKeys = Object.keys(deployment.labels);
    const annotationKeys = Object.keys(deployment.annotations);

    const onNetworkFlowsTabSelect = () => {
        onDeploymentTabsSelect(deploymentTabs.FLOWS);
    };

    const onNetworkPoliciesTabSelect = () => {
        onDeploymentTabsSelect(deploymentTabs.NETWORK_POLICIES);
    };

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
                                        {numAnomalousExternalFlows === 0 &&
                                            numAnomalousInternalFlows === 0 &&
                                            'None'}
                                        {numAnomalousExternalFlows !== 0 && (
                                            <FlexItem>
                                                <Button
                                                    variant="link"
                                                    isInline
                                                    onClick={onNetworkFlowsTabSelect}
                                                >
                                                    <Label
                                                        variant="outline"
                                                        color="red"
                                                        icon={<ExclamationCircleIcon />}
                                                    >
                                                        {numAnomalousExternalFlows} external{' '}
                                                        {pluralize(
                                                            'flow',
                                                            numAnomalousExternalFlows
                                                        )}
                                                    </Label>
                                                </Button>
                                            </FlexItem>
                                        )}
                                        {numAnomalousInternalFlows !== 0 && (
                                            <FlexItem>
                                                <Button
                                                    variant="link"
                                                    isInline
                                                    onClick={onNetworkFlowsTabSelect}
                                                >
                                                    <Label
                                                        variant="outline"
                                                        color="gold"
                                                        icon={<ExclamationTriangleIcon />}
                                                    >
                                                        {numAnomalousInternalFlows} internal{' '}
                                                        {pluralize(
                                                            'flow',
                                                            numAnomalousInternalFlows
                                                        )}
                                                    </Label>
                                                </Button>
                                            </FlexItem>
                                        )}
                                    </Flex>
                                </DescriptionListDescription>
                            </DescriptionListGroup>
                            <DescriptionListGroup>
                                <DescriptionListTerm>Network policy rules</DescriptionListTerm>
                                <DescriptionListDescription>
                                    {networkPolicyState === 'both' && (
                                        <Button
                                            variant="link"
                                            isInline
                                            onClick={onNetworkPoliciesTabSelect}
                                        >
                                            <Label
                                                variant="outline"
                                                color="blue"
                                                icon={
                                                    <BothPolicyRules width="22px" height="22px" />
                                                }
                                            >
                                                1 or more policies regulating bidirectional traffic
                                            </Label>
                                        </Button>
                                    )}
                                    {networkPolicyState === 'egress' && (
                                        <LabelGroup>
                                            <Button
                                                variant="link"
                                                isInline
                                                onClick={onNetworkPoliciesTabSelect}
                                            >
                                                <Label
                                                    variant="outline"
                                                    color="red"
                                                    icon={
                                                        <NoPolicyRules width="22px" height="22px" />
                                                    }
                                                >
                                                    A missing policy is allowing all ingress traffic
                                                </Label>
                                            </Button>
                                            <Button
                                                variant="link"
                                                isInline
                                                onClick={onNetworkPoliciesTabSelect}
                                            >
                                                <Label
                                                    variant="outline"
                                                    color="blue"
                                                    icon={<EgressOnly width="22px" height="22px" />}
                                                >
                                                    1 or more policies regulating egress traffic
                                                </Label>
                                            </Button>
                                        </LabelGroup>
                                    )}
                                    {networkPolicyState === 'ingress' && (
                                        <LabelGroup>
                                            <Button
                                                variant="link"
                                                isInline
                                                onClick={onNetworkPoliciesTabSelect}
                                            >
                                                <Label
                                                    variant="outline"
                                                    icon={
                                                        <NoPolicyRules width="22px" height="22px" />
                                                    }
                                                >
                                                    A missing policy is allowing all egress traffic
                                                </Label>
                                            </Button>
                                            <Button
                                                variant="link"
                                                isInline
                                                onClick={onNetworkPoliciesTabSelect}
                                            >
                                                <Label
                                                    variant="outline"
                                                    color="blue"
                                                    icon={
                                                        <IngressOnly width="22px" height="22px" />
                                                    }
                                                >
                                                    1 or more policies regulating ingress traffic
                                                </Label>
                                            </Button>
                                        </LabelGroup>
                                    )}
                                    {networkPolicyState === 'none' && (
                                        <Button
                                            variant="link"
                                            isInline
                                            onClick={onNetworkPoliciesTabSelect}
                                        >
                                            <Label
                                                variant="outline"
                                                icon={<NoPolicyRules width="22px" height="22px" />}
                                            >
                                                A missing policy is allowing all network traffic
                                            </Label>
                                        </Button>
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
                                                        <Label
                                                            variant="outline"
                                                            key={`${port}-${protocol}`}
                                                        >
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
                    <DetailSection title="Deployment overview">
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
                                            {labelKeys.length === 0 ? (
                                                'None'
                                            ) : (
                                                <LabelGroup>
                                                    {labelKeys.map((labelKey) => {
                                                        const labelValue =
                                                            deployment.labels[labelKey];
                                                        const label = `${labelKey}:${labelValue}`;
                                                        return (
                                                            <Label key={label} color="blue">
                                                                {label}
                                                            </Label>
                                                        );
                                                    })}
                                                </LabelGroup>
                                            )}
                                        </DescriptionListDescription>
                                    </DescriptionListGroup>
                                    <DescriptionListGroup>
                                        <DescriptionListTerm>Annotations</DescriptionListTerm>
                                        <DescriptionListDescription>
                                            {annotationKeys.length === 0 ? (
                                                'None'
                                            ) : (
                                                <LabelGroup>
                                                    {annotationKeys.map((annotationKey) => {
                                                        const annotationValue =
                                                            deployment.annotations[annotationKey];
                                                        const annotation = `${annotationKey}:${annotationValue}`;
                                                        return (
                                                            <Label key={annotationKey} color="blue">
                                                                {annotation}
                                                            </Label>
                                                        );
                                                    })}
                                                </LabelGroup>
                                            )}
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
                                        <StackItem key={port.name}>
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
                <Divider component="li" className="pf-u-mb-sm" />
                <li>
                    <DetailSection title="Container configurations">
                        {deployment.containers.length ? (
                            <Stack hasGutter>
                                {deployment.containers.map((container) => {
                                    return (
                                        <StackItem key={container.id}>
                                            <DeploymentContainerConfig container={container} />
                                        </StackItem>
                                    );
                                })}
                            </Stack>
                        ) : (
                            <EmptyState variant={EmptyStateVariant.xs}>
                                <Title headingLevel="h4" size="md">
                                    No containers available
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
