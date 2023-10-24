import React, { useEffect, CSSProperties } from 'react';
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
import usePermissions from 'hooks/usePermissions';
import {
    getListenPorts,
    getNodeById,
    getNumAnomalousExternalFlows,
    getNumAnomalousInternalFlows,
} from '../utils/networkGraphUtils';
import { CustomEdgeModel, CustomNodeModel } from '../types/topology.type';

import { DeploymentIcon } from '../common/NetworkGraphIcons';
import DeploymentDetails from './DeploymentDetails';
import DeploymentFlows from './DeploymentFlows';
import DeploymentBaselines from './DeploymentBaseline';
import NetworkPolicies from '../common/NetworkPolicies';
import useSimulation from '../hooks/useSimulation';
import { EdgeState } from '../components/EdgeStateSelect';
import { deploymentTabs } from '../utils/deploymentUtils';
import useFetchNetworkFlows from '../api/useFetchNetworkFlows';

const sidebarHeadingStyleConstant = {
    '--pf-u-max-width--MaxWidth': '26ch',
} as CSSProperties;

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
    const { hasReadAccess } = usePermissions();
    const hasReadAccessForDeploymentExtension = hasReadAccess('DeploymentExtension');
    const hasReadAccessForNetworkPolicy = hasReadAccess('NetworkPolicy');
    const { deployment, isLoading: isLoadingDeployment, error } = useFetchDeployment(deploymentId);
    const { activeKeyTab, onSelectTab, setActiveKeyTab } = useTabs({
        defaultTab: defaultDeploymentTab,
    });
    const { simulation } = useSimulation();
    const isBaselineSimulationOn = simulation.isOn && simulation.type === 'baseline';

    const {
        isLoading: isLoadingNetworkFlows,
        error: networkFlowsError,
        data: { networkFlows },
        refetchFlows,
    } = useFetchNetworkFlows({ nodes, edges, deploymentId, edgeState });

    useEffect(() => {
        if (isBaselineSimulationOn) {
            setActiveKeyTab(deploymentTabs.BASELINE);
        }
    }, [isBaselineSimulationOn, setActiveKeyTab]);

    useEffect(() => {
        setActiveKeyTab(defaultDeploymentTab);
    }, [defaultDeploymentTab, setActiveKeyTab]);

    // derived values
    const deploymentNode = getNodeById(nodes, deploymentId);
    const numAnomalousExternalFlows = getNumAnomalousExternalFlows(networkFlows);
    const numAnomalousInternalFlows = getNumAnomalousInternalFlows(networkFlows);
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

    if (isLoadingDeployment) {
        return (
            <Bullseye>
                <Spinner isSVG size="lg" />
            </Bullseye>
        );
    }

    if (error) {
        return (
            <Stack>
                <StackItem>
                    <Flex direction={{ default: 'row' }} className="pf-u-p-md pf-u-mb-0">
                        <FlexItem>
                            <DeploymentIcon />
                        </FlexItem>
                    </Flex>
                </StackItem>
                <StackItem>
                    <Alert
                        variant={AlertVariant.danger}
                        title={error.toString()}
                        className="pf-u-my-lg pf-u-mx-lg"
                    />
                </StackItem>
            </Stack>
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
                            <Text
                                component={TextVariants.h1}
                                className="pf-u-font-size-xl pf-u-max-width"
                                style={sidebarHeadingStyleConstant}
                                data-testid="drawer-title"
                            >
                                {deployment?.name}
                            </Text>
                        </TextContent>
                        <TextContent>
                            <Text
                                component={TextVariants.h2}
                                className="pf-u-font-size-sm pf-u-color-200"
                                data-testid="drawer-subtitle"
                            >
                                in &quot;{deployment?.clusterName} / {deployment?.namespace}&quot;
                            </Text>
                        </TextContent>
                    </FlexItem>
                </Flex>
            </StackItem>
            {deployment && (
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
                            {hasReadAccessForDeploymentExtension && (
                                <Tab
                                    eventKey={deploymentTabs.BASELINE}
                                    tabContentId={deploymentTabs.BASELINE}
                                    title={<TabTitleText>{deploymentTabs.BASELINE}</TabTitleText>}
                                />
                            )}
                            {hasReadAccessForNetworkPolicy && (
                                <Tab
                                    eventKey={deploymentTabs.NETWORK_POLICIES}
                                    tabContentId={deploymentTabs.NETWORK_POLICIES}
                                    title={
                                        <TabTitleText>
                                            {deploymentTabs.NETWORK_POLICIES}
                                        </TabTitleText>
                                    }
                                    disabled={isBaselineSimulationOn}
                                />
                            )}
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
                                    numAnomalousExternalFlows={numAnomalousExternalFlows}
                                    numAnomalousInternalFlows={numAnomalousInternalFlows}
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
                            {activeKeyTab === deploymentTabs.FLOWS && (
                                <DeploymentFlows
                                    nodes={nodes}
                                    deploymentId={deploymentId}
                                    edgeState={edgeState}
                                    onNodeSelect={onNodeSelect}
                                    isLoadingNetworkFlows={isLoadingNetworkFlows}
                                    networkFlowsError={networkFlowsError}
                                    networkFlows={networkFlows}
                                    refetchFlows={refetchFlows}
                                />
                            )}
                        </TabContent>
                        <TabContent
                            eventKey={deploymentTabs.BASELINE}
                            id={deploymentTabs.BASELINE}
                            hidden={activeKeyTab !== deploymentTabs.BASELINE}
                            className="pf-u-h-100"
                        >
                            {activeKeyTab === deploymentTabs.BASELINE && (
                                <DeploymentBaselines
                                    deployment={deployment}
                                    deploymentId={deploymentId}
                                    onNodeSelect={onNodeSelect}
                                />
                            )}
                        </TabContent>
                        {hasReadAccessForNetworkPolicy && (
                            <TabContent
                                eventKey={deploymentTabs.NETWORK_POLICIES}
                                id={deploymentTabs.NETWORK_POLICIES}
                                hidden={activeKeyTab !== deploymentTabs.NETWORK_POLICIES}
                            >
                                <NetworkPolicies
                                    entityName={deployment.name}
                                    policyIds={deploymentPolicyIds}
                                />
                            </TabContent>
                        )}
                    </StackItem>
                </>
            )}
        </Stack>
    );
}

export default DeploymentSideBar;
