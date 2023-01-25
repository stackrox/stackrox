import React from 'react';
import {
    Flex,
    FlexItem,
    Stack,
    StackItem,
    Tab,
    TabContent,
    Tabs,
    TabTitleText,
    Text,
    TextContent,
    TextVariants,
} from '@patternfly/react-core';

import useTabs from 'hooks/patternfly/useTabs';
import { uniq } from 'lodash';
import { getDeploymentNodesInNamespace, getNumDeploymentFlows } from '../utils/networkGraphUtils';
import { CustomEdgeModel, CustomNodeModel } from '../types/topology.type';

import { NamespaceIcon } from '../common/NetworkGraphIcons';
import NamespaceDeployments from './NamespaceDeployments';
import NetworkPolicies from '../common/NetworkPolicies';

type NamespaceSideBarProps = {
    namespaceId: string;
    nodes: CustomNodeModel[];
    edges: CustomEdgeModel[];
};

function NamespaceSideBar({ namespaceId, nodes, edges }: NamespaceSideBarProps) {
    // component state
    const { activeKeyTab, onSelectTab } = useTabs({
        defaultTab: 'Deployments',
    });

    // derived state
    const deploymentNodes = getDeploymentNodesInNamespace(nodes, namespaceId);

    const deployments = deploymentNodes.map((deploymentNode) => {
        const numFlows = getNumDeploymentFlows(edges, deploymentNode.id);
        return {
            name: deploymentNode.label as string,
            numFlows,
        };
    });
    const namespacePolicyIds = deploymentNodes.reduce((acc, curr) => {
        const policyIds: string[] = curr?.data?.policyIds || [];
        return [...acc, ...policyIds];
    }, [] as string[]);
    const uniqueNamespacePolicyIds = uniq(namespacePolicyIds);

    return (
        <Stack>
            <StackItem>
                <Flex direction={{ default: 'row' }} className="pf-u-p-md pf-u-mb-0">
                    <FlexItem>
                        <NamespaceIcon />
                    </FlexItem>
                    <FlexItem>
                        <TextContent>
                            <Text component={TextVariants.h1} className="pf-u-font-size-xl">
                                stackrox
                            </Text>
                        </TextContent>
                        <TextContent>
                            <Text
                                component={TextVariants.h2}
                                className="pf-u-font-size-sm pf-u-color-200"
                            >
                                in &quot;remote&quot;
                            </Text>
                        </TextContent>
                    </FlexItem>
                </Flex>
            </StackItem>
            <StackItem>
                <Tabs activeKey={activeKeyTab} onSelect={onSelectTab}>
                    <Tab
                        eventKey="Deployments"
                        tabContentId="Deployments"
                        title={<TabTitleText>Deployments</TabTitleText>}
                    />
                    <Tab
                        eventKey="Network policies"
                        tabContentId="Network policies"
                        title={<TabTitleText>Network policies</TabTitleText>}
                    />
                </Tabs>
            </StackItem>
            <StackItem isFilled style={{ overflow: 'auto' }}>
                <TabContent
                    eventKey="Deployments"
                    id="Deployments"
                    hidden={activeKeyTab !== 'Deployments'}
                >
                    <NamespaceDeployments deployments={deployments} />
                </TabContent>
                <TabContent
                    eventKey="Network policies"
                    id="Network policies"
                    hidden={activeKeyTab !== 'Network policies'}
                    className="pf-u-h-100"
                >
                    <NetworkPolicies policyIds={uniqueNamespacePolicyIds} />
                </TabContent>
            </StackItem>
        </Stack>
    );
}

export default NamespaceSideBar;
