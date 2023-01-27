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

type DeploymentSideBarProps = {
    deploymentId: string;
    nodes: CustomNodeModel[];
    edges: CustomEdgeModel[];
    edgeState: EdgeState;
};

function DeploymentSideBar({ deploymentId, nodes, edges, edgeState }: DeploymentSideBarProps) {
    // component state
    const { deployment, isLoading, error } = useFetchDeployment(deploymentId);
    const { activeKeyTab, onSelectTab, setActiveKeyTab } = useTabs({
        defaultTab: 'Details',
    });
    const { simulation } = useSimulation();
    const isBaselineSimulationOn = simulation.isOn && simulation.type === 'baseline';

    useEffect(() => {
        if (isBaselineSimulationOn) {
            setActiveKeyTab('Baselines');
        }
    }, [isBaselineSimulationOn, setActiveKeyTab]);

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
                    <DeploymentBaselinesSimulated deploymentId={deploymentId} />
                </StackItem>
            )}
            {!isBaselineSimulationOn && deployment && (
                <>
                    <StackItem>
                        <Tabs activeKey={activeKeyTab} onSelect={onSelectTab}>
                            <Tab
                                eventKey="Details"
                                tabContentId="Details"
                                title={<TabTitleText>Details</TabTitleText>}
                                disabled={isBaselineSimulationOn}
                            />
                            <Tab
                                eventKey="Flows"
                                tabContentId="Flows"
                                title={<TabTitleText>Flows</TabTitleText>}
                                disabled={isBaselineSimulationOn}
                            />
                            <Tab
                                eventKey="Baselines"
                                tabContentId="Baselines"
                                title={<TabTitleText>Baselines</TabTitleText>}
                            />
                            <Tab
                                eventKey="Network policies"
                                tabContentId="Network policies"
                                title={<TabTitleText>Network policies</TabTitleText>}
                                disabled={isBaselineSimulationOn}
                            />
                        </Tabs>
                    </StackItem>
                    <StackItem isFilled style={{ overflow: 'auto' }}>
                        <TabContent
                            eventKey="Details"
                            id="Details"
                            hidden={activeKeyTab !== 'Details'}
                        >
                            {deployment && (
                                <DeploymentDetails
                                    deployment={deployment}
                                    numExternalFlows={numExternalFlows}
                                    numInternalFlows={numInternalFlows}
                                    listenPorts={listenPorts}
                                    networkPolicyState={networkPolicyState}
                                />
                            )}
                        </TabContent>
                        <TabContent eventKey="Flows" id="Flows" hidden={activeKeyTab !== 'Flows'}>
                            <DeploymentFlows
                                nodes={nodes}
                                edges={edges}
                                deploymentId={deploymentId}
                                edgeState={edgeState}
                            />
                        </TabContent>
                        <TabContent
                            eventKey="Baselines"
                            id="Baselines"
                            hidden={activeKeyTab !== 'Baselines'}
                            className="pf-u-h-100"
                        >
                            <DeploymentBaselines
                                deployment={deployment}
                                deploymentId={deploymentId}
                            />
                        </TabContent>
                        <TabContent
                            eventKey="Network policies"
                            id="Network policies"
                            hidden={activeKeyTab !== 'Network policies'}
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
