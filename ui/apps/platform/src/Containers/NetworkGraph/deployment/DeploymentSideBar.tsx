import React, { useEffect } from 'react';
import {
    Alert,
    AlertVariant,
    Bullseye,
    Flex,
    FlexItem,
    Spinner,
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
import useFetchDeployment from 'hooks/useFetchDeployment';
import {
    getListenPorts,
    getNodeById,
    getNumExternalFlows,
    getNumInternalFlows,
} from '../utils/networkGraphUtils';
import { CustomEdgeModel, CustomNodeModel } from '../types/topology.type';

import { DeploymentIcon } from '../common/NetworkGraphIcons';
import DeploymentDetails from './DeploymentDetails';
import DeploymentFlows from './DeploymentFlows';
import DeploymentBaselines from './DeploymentBaselines';
import NetworkPolicies from '../common/NetworkPolicies';
import useSimulation from '../hooks/useSimulation';
import DeploymentBaselinesSimulated from './DeploymentBaselinesSimulated';
import { EdgeState } from '../components/EdgeStateSelect';
import { deploymentTabs } from '../utils/deploymentUtils';

type DeploymentSideBarProps = {
    deploymentId: string;
    nodes: CustomNodeModel[];
    edges: CustomEdgeModel[];
    edgeState: EdgeState;
    onNodeSelect: (id: string) => void;
    defaultDeploymentTab: string;
};

function DeploymentSideBar({
    deploymentId,
    nodes,
    edges,
    edgeState,
    onNodeSelect,
    defaultDeploymentTab,
}: DeploymentSideBarProps) {
    // component state
    const { deployment, isLoading, error } = useFetchDeployment(deploymentId);
    const { activeKeyTab, onSelectTab, setActiveKeyTab } = useTabs({
        defaultTab: defaultDeploymentTab,
    });
    const { simulation } = useSimulation();
    const isBaselineSimulationOn = simulation.isOn && simulation.type === 'baseline';

    useEffect(() => {
        if (isBaselineSimulationOn) {
            setActiveKeyTab(deploymentTabs.BASELINES);
        }
    }, [isBaselineSimulationOn, setActiveKeyTab]);

    useEffect(() => {
        setActiveKeyTab(defaultDeploymentTab);
    }, [defaultDeploymentTab, setActiveKeyTab]);

    // derived values
    const deploymentNode = getNodeById(nodes, deploymentId);
    const numExternalFlows = getNumExternalFlows(nodes, edges, deploymentId);
    const numInternalFlows = getNumInternalFlows(nodes, edges, deploymentId);
    const listenPorts = getListenPorts(nodes, deploymentId);
    const deploymentPolicyIds =
        deploymentNode?.data.type === 'DEPLOYMENT' ? deploymentNode?.data?.policyIds : [];
    const networkPolicyState =
        deploymentNode?.data.type === 'DEPLOYMENT'
            ? deploymentNode.data.networkPolicyState
            : 'none';

    const onDeploymentTabsSelect = (tab: string) => {
        setActiveKeyTab(tab);
    };

    if (isLoading) {
        return (
            <Bullseye>
                <Spinner isSVG size="lg" />
            </Bullseye>
        );
    }

    if (error) {
        return (
            <Alert isInline variant={AlertVariant.danger} title={error} className="pf-u-mb-lg" />
        );
    }

    return (
        <Stack>
            <StackItem>
                <Flex direction={{ default: 'row' }} className="pf-u-p-md pf-u-mb-0">
                    <FlexItem>
                        <DeploymentIcon />
                    </FlexItem>
                    <FlexItem>
                        <TextContent>
                            <Text component={TextVariants.h1} className="pf-u-font-size-xl">
                                {deployment?.name}
                            </Text>
                        </TextContent>
                        <TextContent>
                            <Text
                                component={TextVariants.h2}
                                className="pf-u-font-size-sm pf-u-color-200"
                            >
                                in &quot;{deployment?.clusterName} / {deployment?.namespace}&quot;
                            </Text>
                        </TextContent>
                    </FlexItem>
                </Flex>
            </StackItem>
            {isBaselineSimulationOn && (
                <StackItem isFilled style={{ overflow: 'auto' }} className="pf-u-h-100">
                    <DeploymentBaselinesSimulated
                        deploymentId={deploymentId}
                        onNodeSelect={onNodeSelect}
                    />
                </StackItem>
            )}
            {!isBaselineSimulationOn && deployment && (
                <>
                    <StackItem>
                        <Tabs activeKey={activeKeyTab} onSelect={onSelectTab}>
                            <Tab
                                eventKey={deploymentTabs.DETAILS}
                                tabContentId={deploymentTabs.DETAILS}
                                title={<TabTitleText>{deploymentTabs.DETAILS}</TabTitleText>}
                                disabled={isBaselineSimulationOn}
                            />
                            <Tab
                                eventKey={deploymentTabs.FLOWS}
                                tabContentId={deploymentTabs.FLOWS}
                                title={<TabTitleText>{deploymentTabs.FLOWS}</TabTitleText>}
                                disabled={isBaselineSimulationOn}
                            />
                            <Tab
                                eventKey={deploymentTabs.BASELINES}
                                tabContentId={deploymentTabs.BASELINES}
                                title={<TabTitleText>{deploymentTabs.BASELINES}</TabTitleText>}
                            />
                            <Tab
                                eventKey={deploymentTabs.NETWORK_POLICIES}
                                tabContentId={deploymentTabs.NETWORK_POLICIES}
                                title={
                                    <TabTitleText>{deploymentTabs.NETWORK_POLICIES}</TabTitleText>
                                }
                                disabled={isBaselineSimulationOn}
                            />
                        </Tabs>
                    </StackItem>
                    <StackItem isFilled style={{ overflow: 'auto' }}>
                        <TabContent
                            eventKey={deploymentTabs.DETAILS}
                            id={deploymentTabs.DETAILS}
                            hidden={activeKeyTab !== deploymentTabs.DETAILS}
                        >
                            {deployment && (
                                <DeploymentDetails
                                    deployment={deployment}
                                    numExternalFlows={numExternalFlows}
                                    numInternalFlows={numInternalFlows}
                                    listenPorts={listenPorts}
                                    networkPolicyState={networkPolicyState}
                                    onDeploymentTabsSelect={onDeploymentTabsSelect}
                                />
                            )}
                        </TabContent>
                        <TabContent
                            eventKey={deploymentTabs.FLOWS}
                            id={deploymentTabs.FLOWS}
                            hidden={activeKeyTab !== deploymentTabs.FLOWS}
                        >
                            <DeploymentFlows
                                nodes={nodes}
                                edges={edges}
                                deploymentId={deploymentId}
                                edgeState={edgeState}
                                onNodeSelect={onNodeSelect}
                            />
                        </TabContent>
                        <TabContent
                            eventKey={deploymentTabs.BASELINES}
                            id={deploymentTabs.BASELINES}
                            hidden={activeKeyTab !== deploymentTabs.BASELINES}
                            className="pf-u-h-100"
                        >
                            <DeploymentBaselines
                                deployment={deployment}
                                deploymentId={deploymentId}
                                onNodeSelect={onNodeSelect}
                            />
                        </TabContent>
                        <TabContent
                            eventKey={deploymentTabs.NETWORK_POLICIES}
                            id={deploymentTabs.NETWORK_POLICIES}
                            hidden={activeKeyTab !== deploymentTabs.NETWORK_POLICIES}
                        >
                            <NetworkPolicies policyIds={deploymentPolicyIds} />
                        </TabContent>
                    </StackItem>
                </>
            )}
        </Stack>
    );
}

export default DeploymentSideBar;
