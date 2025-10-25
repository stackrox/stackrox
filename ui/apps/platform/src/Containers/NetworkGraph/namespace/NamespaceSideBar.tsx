import { useEffect } from 'react';
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
    Title,
} from '@patternfly/react-core';
import uniq from 'lodash/uniq';

import type { QueryValue } from 'hooks/useURLParameter';
import {
    getDeploymentNodesInNamespace,
    getNodeById,
    getNumDeploymentFlows,
} from '../utils/networkGraphUtils';
import type { CustomEdgeModel, CustomNodeModel } from '../types/topology.type';

import { NamespaceIcon } from '../common/NetworkGraphIcons';
import NamespaceDeployments from './NamespaceDeployments';
import NetworkPolicies from '../common/NetworkPolicies';
import { useSidePanelTab } from '../NetworkGraphURLStateContext';

type NamespaceSideBarProps = {
    labelledById: string; // corresponds to aria-labelledby prop of TopologySideBar
    namespaceId: string;
    nodes: CustomNodeModel[];
    edges: CustomEdgeModel[];
    onNodeSelect: (id: string) => void;
};

const NAMESPACE_TABS = ['DEPLOYMENTS', 'NETWORK_POLICIES'] as const;
type NamespaceTabKey = (typeof NAMESPACE_TABS)[number];

const DEFAULT_NAMESPACE_TAB: NamespaceTabKey = 'DEPLOYMENTS';

function isValidNamespaceTab(value: QueryValue): value is NamespaceTabKey {
    return typeof value === 'string' && NAMESPACE_TABS.some((tab) => tab === value);
}

function NamespaceSideBar({
    labelledById,
    namespaceId,
    nodes,
    edges,
    onNodeSelect,
}: NamespaceSideBarProps) {
    const { selectedTabSidePanel, setSelectedTabSidePanel } = useSidePanelTab();

    // derived state
    const namespaceNode = getNodeById(nodes, namespaceId);
    const deploymentNodes = getDeploymentNodesInNamespace(nodes, namespaceId);
    const cluster = namespaceNode?.data.type === 'NAMESPACE' ? namespaceNode.data.cluster : '';

    const deployments = deploymentNodes.map((deploymentNode) => {
        const numFlows = getNumDeploymentFlows(edges, deploymentNode.id);
        return {
            id: deploymentNode.id,
            name: deploymentNode.label as string,
            numFlows,
        };
    });
    const namespacePolicyIds = deploymentNodes.reduce((acc, curr) => {
        const policyIds: string[] = curr?.data?.policyIds || [];
        return [...acc, ...policyIds];
    }, [] as string[]);
    const uniqueNamespacePolicyIds = uniq(namespacePolicyIds);

    const activeTab: NamespaceTabKey = isValidNamespaceTab(selectedTabSidePanel)
        ? selectedTabSidePanel
        : DEFAULT_NAMESPACE_TAB;

    useEffect(() => {
        if (selectedTabSidePanel !== undefined && !isValidNamespaceTab(selectedTabSidePanel)) {
            setSelectedTabSidePanel(DEFAULT_NAMESPACE_TAB, 'replace');
        }
    }, [selectedTabSidePanel, setSelectedTabSidePanel]);

    return (
        <Stack>
            <StackItem>
                <Flex direction={{ default: 'row' }} className="pf-v5-u-p-md pf-v5-u-mb-0">
                    <FlexItem>
                        <NamespaceIcon />
                    </FlexItem>
                    <FlexItem>
                        <Title headingLevel="h2" id={labelledById}>
                            {namespaceNode?.label}
                        </Title>
                        <Text className="pf-v5-u-font-size-sm pf-v5-u-color-200">
                            in &quot;
                            {cluster}
                            &quot;
                        </Text>
                    </FlexItem>
                </Flex>
            </StackItem>
            <StackItem>
                <Tabs
                    activeKey={activeTab}
                    onSelect={(_e, key) => setSelectedTabSidePanel(key.toString())}
                >
                    <Tab
                        eventKey="DEPLOYMENTS"
                        tabContentId="DEPLOYMENTS"
                        title={<TabTitleText>Deployments</TabTitleText>}
                    />
                    <Tab
                        eventKey="NETWORK_POLICIES"
                        tabContentId="NETWORK_POLICIES"
                        title={<TabTitleText>Network policies</TabTitleText>}
                    />
                </Tabs>
            </StackItem>
            <StackItem isFilled style={{ overflow: 'auto' }}>
                <TabContent
                    eventKey="DEPLOYMENTS"
                    id="DEPLOYMENTS"
                    hidden={activeTab !== 'DEPLOYMENTS'}
                >
                    <NamespaceDeployments deployments={deployments} onNodeSelect={onNodeSelect} />
                </TabContent>
                <TabContent
                    eventKey="NETWORK_POLICIES"
                    id="NETWORK_POLICIES"
                    hidden={activeTab !== 'NETWORK_POLICIES'}
                    className="pf-v5-u-h-100"
                >
                    <NetworkPolicies
                        entityName={namespaceNode?.label || ''}
                        policyIds={uniqueNamespacePolicyIds}
                    />
                </TabContent>
            </StackItem>
        </Stack>
    );
}

export default NamespaceSideBar;
