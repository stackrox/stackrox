import { useState } from 'react';
import {
    DescriptionList,
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
    Divider,
    EmptyState,
    ExpandableSection,
    Flex,
    FlexItem,
    Label,
    LabelGroup,
    Stack,
    StackItem,
    TextContent,
    Title,
    EmptyStateHeader,
} from '@patternfly/react-core';

import usePermissions from 'hooks/usePermissions';
import type { Deployment } from 'types/deployment.proto';
import type { ListenPort } from 'types/networkFlow.proto';
import { getDateTime } from 'utils/dateUtils';

import DeploymentPortConfig from 'Components/DeploymentPortConfig';
import DeploymentContainerConfig from 'Components/DeploymentContainerConfig';

import BothPolicyRules from 'images/network-graph/both-policy-rules.svg?react';
import EgressOnly from 'images/network-graph/egress-only.svg?react';
import IngressOnly from 'images/network-graph/ingress-only.svg?react';
import NoPolicyRules from 'images/network-graph/no-policy-rules.svg?react';

import type { EdgeState } from '../components/EdgeStateSelect';
import type { CustomEdgeModel, CustomNodeModel, NetworkPolicyState } from '../types/topology.type';

import AnomalousTraffic from './AnomalousTraffic';

import './DeploymentDetails.css';

type DeploymentDetailsProps = {
    deployment: Deployment;
    deploymentId: string;
    edgeState: EdgeState;
    edges: CustomEdgeModel[];
    listenPorts: ListenPort[];
    networkPolicyState: NetworkPolicyState;
    nodes: CustomNodeModel[];
};

function DetailSection({ title, children }) {
    const [isExpanded, setIsExpanded] = useState(true);

    const onToggle = (_isExpanded: boolean) => {
        setIsExpanded(_isExpanded);
    };

    // TextContent so heading has black instead of blue color.
    return (
        <ExpandableSection
            isExpanded={isExpanded}
            onToggle={(_event, _isExpanded: boolean) => onToggle(_isExpanded)}
            toggleContent={
                <TextContent>
                    <Title headingLevel="h2">{title}</Title>
                </TextContent>
            }
        >
            <div className="pf-v5-u-px-sm pf-v5-u-pb-md">{children}</div>
        </ExpandableSection>
    );
}

function DeploymentDetails({
    deployment,
    deploymentId,
    edgeState,
    edges,
    listenPorts,
    networkPolicyState,
    nodes,
}: DeploymentDetailsProps) {
    const labelKeys = Object.keys(deployment.labels);
    const annotationKeys = Object.keys(deployment.annotations);

    const { hasReadAccess } = usePermissions();
    const hasReadAccessForNetworkPolicy = hasReadAccess('NetworkPolicy');

    return (
        <div className="pf-v5-u-h-100 pf-v5-u-p-md">
            <ul>
                <li>
                    <DetailSection title="Network security">
                        <DescriptionList columnModifier={{ default: '1Col' }}>
                            {hasReadAccessForNetworkPolicy && (
                                <AnomalousTraffic
                                    deploymentId={deploymentId}
                                    edgeState={edgeState}
                                    edges={edges}
                                    nodes={nodes}
                                />
                            )}
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
                <li>
                    <Divider className="pf-v5-u-mb-sm" />
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
                <li>
                    <Divider className="pf-v5-u-mb-sm" />
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
                            <EmptyState variant="xs">
                                <EmptyStateHeader
                                    titleText="No ports available"
                                    headingLevel="h4"
                                />
                            </EmptyState>
                        )}
                    </DetailSection>
                </li>
                <li>
                    <Divider className="pf-v5-u-mb-sm" />
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
                            <EmptyState variant="xs">
                                <EmptyStateHeader
                                    titleText="No containers available"
                                    headingLevel="h4"
                                />
                            </EmptyState>
                        )}
                    </DetailSection>
                </li>
            </ul>
        </div>
    );
}

export default DeploymentDetails;
