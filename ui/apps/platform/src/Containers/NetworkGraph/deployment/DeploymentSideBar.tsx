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
import usePermissions, { HasReadAccess } from 'hooks/usePermissions';
import { QueryValue } from 'hooks/useURLParameter';
import { ensureExhaustive } from 'utils/type.utils';

import { getListenPorts, getNodeById } from '../utils/networkGraphUtils';
import { CustomEdgeModel, CustomNodeModel } from '../types/topology.type';

import { DeploymentIcon } from '../common/NetworkGraphIcons';
import DeploymentDetails from './DeploymentDetails';
import DeploymentFlows from './DeploymentFlows';
import DeploymentBaseline from './DeploymentBaseline';
import NetworkPolicies from '../common/NetworkPolicies';
import useSimulation from '../hooks/useSimulation';
import { EdgeState } from '../components/EdgeStateSelect';
import { DEFAULT_NETWORK_GRAPH_PAGE_SIZE } from '../NetworkGraph.constants';
import {
    usePagination,
    usePaginationSecondary,
    useSidePanelTab,
    useSidePanelToggle,
} from '../NetworkGraphURLStateContext';

const sidebarHeadingStyleConstant = {
    '--pf-v5-u-max-width--MaxWidth': '26ch',
} as CSSProperties;

const DEPLOYMENT_TABS = ['BASELINE', 'DETAILS', 'FLOWS', 'NETWORK_POLICIES'] as const;
type DeploymentTabKey = (typeof DEPLOYMENT_TABS)[number];

const DEFAULT_DEPLOYMENT_TAB: DeploymentTabKey = 'DETAILS';

function hasReadAccessForDeploymentTab(hasReadAccess: HasReadAccess, tabKey: DeploymentTabKey) {
    switch (tabKey) {
        case 'BASELINE':
        case 'FLOWS':
            return hasReadAccess('DeploymentExtension');
        case 'NETWORK_POLICIES':
            return hasReadAccess('NetworkPolicy');
        case 'DETAILS':
            return true;
        default:
            return ensureExhaustive(tabKey);
    }
}

function isValidDeploymentTab(value: QueryValue): value is DeploymentTabKey {
    return typeof value === 'string' && DEPLOYMENT_TABS.some((tab) => tab === value);
}

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
    const { setPerPage: setPerPageAnomalous } = usePagination();
    const { setPerPage: setPerPageBaseline } = usePaginationSecondary();
    const { selectedTabSidePanel, setSelectedTabSidePanel } = useSidePanelTab();
    const { setSelectedToggleSidePanel } = useSidePanelToggle();
    const { hasReadAccess } = usePermissions();
    const { deployment, isLoading: isLoadingDeployment, error } = useFetchDeployment(deploymentId);

    const { simulation } = useSimulation();
    const isBaselineSimulationOn = simulation.isOn && simulation.type === 'baseline';

    const activeTab: DeploymentTabKey =
        isValidDeploymentTab(selectedTabSidePanel) &&
        hasReadAccessForDeploymentTab(hasReadAccess, selectedTabSidePanel)
            ? selectedTabSidePanel
            : DEFAULT_DEPLOYMENT_TAB;

    useEffect(() => {
        if (isBaselineSimulationOn) {
            setSelectedTabSidePanel('BASELINE');
        }
    }, [isBaselineSimulationOn, setSelectedTabSidePanel]);

    useEffect(() => {
        if (selectedTabSidePanel !== undefined && !isValidDeploymentTab(selectedTabSidePanel)) {
            setSelectedTabSidePanel(DEFAULT_DEPLOYMENT_TAB, 'replace');
        }
    }, [selectedTabSidePanel, setSelectedTabSidePanel]);

    // derived values
    const deploymentNode = getNodeById(nodes, deploymentId);
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
        setPerPageAnomalous(DEFAULT_NETWORK_GRAPH_PAGE_SIZE);
        setPerPageBaseline(DEFAULT_NETWORK_GRAPH_PAGE_SIZE);
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
                            activeKey={activeTab}
                            onSelect={(_e, key) => handleSelectTab(key.toString())}
                        >
                            <Tab
                                eventKey={'DETAILS'}
                                tabContentId={'DETAILS'}
                                title={<TabTitleText>Details</TabTitleText>}
                                disabled={isBaselineSimulationOn}
                            />
                            {hasReadAccessForDeploymentTab(hasReadAccess, 'FLOWS') && (
                                <Tab
                                    eventKey={'FLOWS'}
                                    tabContentId={'FLOWS'}
                                    title={<TabTitleText>Flows</TabTitleText>}
                                    disabled={isBaselineSimulationOn}
                                />
                            )}
                            {hasReadAccessForDeploymentTab(hasReadAccess, 'BASELINE') && (
                                <Tab
                                    eventKey={'BASELINE'}
                                    tabContentId={'BASELINE'}
                                    title={<TabTitleText>Baseline</TabTitleText>}
                                />
                            )}
                            {hasReadAccessForDeploymentTab(hasReadAccess, 'NETWORK_POLICIES') && (
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
                                    deploymentId={deploymentId}
                                    edgeState={edgeState}
                                    edges={edges}
                                    listenPorts={listenPorts}
                                    networkPolicyState={networkPolicyState}
                                    nodes={nodes}
                                />
                            )}
                        </TabContent>
                        {hasReadAccessForDeploymentTab(hasReadAccess, 'FLOWS') && (
                            <TabContent
                                eventKey={'FLOWS'}
                                id={'FLOWS'}
                                hidden={activeTab !== 'FLOWS'}
                            >
                                {activeTab === 'FLOWS' && (
                                    <DeploymentFlows
                                        deploymentId={deploymentId}
                                        edgeState={edgeState}
                                        edges={edges}
                                        nodes={nodes}
                                        onNodeSelect={onNodeSelect}
                                    />
                                )}
                            </TabContent>
                        )}
                        {hasReadAccessForDeploymentTab(hasReadAccess, 'BASELINE') && (
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
                        )}
                        {hasReadAccessForDeploymentTab(hasReadAccess, 'NETWORK_POLICIES') && (
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
