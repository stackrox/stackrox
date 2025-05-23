import React, { useEffect, CSSProperties } from 'react';
import {
    Alert,
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
    Title,
} from '@patternfly/react-core';

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
import DeploymentBaseline from './DeploymentBaseline';
import NetworkPolicies from '../common/NetworkPolicies';
import useSimulation from '../hooks/useSimulation';
import { EdgeState } from '../components/EdgeStateSelect';
import useFetchNetworkFlows from '../api/useFetchNetworkFlows';

import { useSidePanelTab, useSidePanelToggle } from '../NetworkGraphURLStateContext';

const sidebarHeadingStyleConstant = {
    '--pf-v5-u-max-width--MaxWidth': '26ch',
} as CSSProperties;

type DeploymentSideBarProps = {
    labelledById: string; // corresponds to aria-labelledby prop of TopologySideBar
    deploymentId: string;
    nodes: CustomNodeModel[];
    edges: CustomEdgeModel[];
    edgeState: EdgeState;
    onNodeSelect: (id: string) => void;
};

function DeploymentSideBar({
    labelledById,
    deploymentId,
    nodes,
    edges,
    edgeState,
    onNodeSelect,
}: DeploymentSideBarProps) {
    // component state
    const { hasReadAccess } = usePermissions();
    const { selectedTabSidePanel, setSelectedTabSidePanel } = useSidePanelTab();
    const { setSelectedToggleSidePanel } = useSidePanelToggle();
    const hasReadAccessForDeploymentExtension = hasReadAccess('DeploymentExtension');
    const hasReadAccessForNetworkPolicy = hasReadAccess('NetworkPolicy');
    const { deployment, isLoading: isLoadingDeployment, error } = useFetchDeployment(deploymentId);

    const { simulation } = useSimulation();
    const isBaselineSimulationOn = simulation.isOn && simulation.type === 'baseline';

    const activeTab = selectedTabSidePanel ?? 'DETAILS';

    const {
        isLoading: isLoadingNetworkFlows,
        error: networkFlowsError,
        data: { networkFlows },
        refetchFlows,
    } = useFetchNetworkFlows({ nodes, edges, deploymentId, edgeState });

    useEffect(() => {
        if (isBaselineSimulationOn) {
            setSelectedTabSidePanel('BASELINE');
        }
    }, [isBaselineSimulationOn, setSelectedTabSidePanel]);

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

    if (isLoadingDeployment) {
        return (
            <Bullseye>
                <Spinner size="lg" />
            </Bullseye>
        );
    }

    function handleSelectTab(key: string) {
        setSelectedTabSidePanel(key);
        setSelectedToggleSidePanel(undefined);
    }

    if (error) {
        return (
            <Stack>
                <StackItem>
                    <Flex direction={{ default: 'row' }} className="pf-v5-u-p-md pf-v5-u-mb-0">
                        <FlexItem>
                            <DeploymentIcon />
                        </FlexItem>
                    </Flex>
                </StackItem>
                <StackItem>
                    <Alert
                        variant="danger"
                        title={error.toString()}
                        component="p"
                        className="pf-v5-u-my-lg pf-v5-u-mx-lg"
                    />
                </StackItem>
            </Stack>
        );
    }

    return (
        <Stack>
            <StackItem>
                <Flex direction={{ default: 'row' }} className="pf-v5-u-p-md pf-v5-u-mb-0">
                    <FlexItem>
                        <DeploymentIcon />
                    </FlexItem>
                    <FlexItem>
                        <Title
                            headingLevel="h2"
                            id={labelledById}
                            className="pf-v5-u-max-width"
                            style={sidebarHeadingStyleConstant}
                            data-testid="drawer-title"
                        >
                            {deployment?.name}
                        </Title>
                        <Text
                            className="pf-v5-u-font-size-sm pf-v5-u-color-200"
                            data-testid="drawer-subtitle"
                        >
                            in &quot;{deployment?.clusterName} / {deployment?.namespace}&quot;
                        </Text>
                    </FlexItem>
                </Flex>
            </StackItem>
            {deployment && (
                <>
                    <StackItem>
                        <Tabs
                            // TODO: don't type case
                            activeKey={activeTab as string}
                            onSelect={(_e, key) => handleSelectTab(key.toString())}
                        >
                            <Tab
                                eventKey={'DETAILS'}
                                tabContentId={'DETAILS'}
                                title={<TabTitleText>Details</TabTitleText>}
                                disabled={isBaselineSimulationOn}
                            />
                            <Tab
                                eventKey={'FLOWS'}
                                tabContentId={'FLOWS'}
                                title={<TabTitleText>Flows</TabTitleText>}
                                disabled={isBaselineSimulationOn}
                            />
                            {hasReadAccessForDeploymentExtension && (
                                <Tab
                                    eventKey={'BASELINE'}
                                    tabContentId={'BASELINE'}
                                    title={<TabTitleText>Baseline</TabTitleText>}
                                />
                            )}
                            {hasReadAccessForNetworkPolicy && (
                                <Tab
                                    eventKey={'NETWORK_POLICIES'}
                                    tabContentId="NETWORK_POLICIES"
                                    title={<TabTitleText>Network Policies</TabTitleText>}
                                    disabled={isBaselineSimulationOn}
                                />
                            )}
                        </Tabs>
                    </StackItem>
                    <StackItem isFilled style={{ overflow: 'auto' }}>
                        <TabContent
                            eventKey={'DETAILS'}
                            id={'DETAILS'}
                            hidden={activeTab !== 'DETAILS'}
                        >
                            {deployment && (
                                <DeploymentDetails
                                    deployment={deployment}
                                    numAnomalousExternalFlows={numAnomalousExternalFlows}
                                    numAnomalousInternalFlows={numAnomalousInternalFlows}
                                    listenPorts={listenPorts}
                                    networkPolicyState={networkPolicyState}
                                />
                            )}
                        </TabContent>
                        <TabContent eventKey={'FLOWS'} id={'FLOWS'} hidden={activeTab !== 'FLOWS'}>
                            {activeTab === 'FLOWS' && (
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
                            eventKey={'BASELINE'}
                            id={'BASELINE'}
                            hidden={activeTab !== 'BASELINE'}
                            className="pf-v5-u-h-100"
                        >
                            {activeTab === 'BASELINE' && (
                                <DeploymentBaseline
                                    deployment={deployment}
                                    deploymentId={deploymentId}
                                    onNodeSelect={onNodeSelect}
                                />
                            )}
                        </TabContent>
                        {hasReadAccessForNetworkPolicy && (
                            <TabContent
                                eventKey={'NETWORK_POLICIES'}
                                id="NETWORK_POLICIES"
                                hidden={activeTab !== 'NETWORK_POLICIES'}
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
