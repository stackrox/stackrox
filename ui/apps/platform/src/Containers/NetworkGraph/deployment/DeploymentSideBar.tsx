import React from 'react';
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

type DeploymentSideBarProps = {
    deploymentId: string;
    nodes: CustomNodeModel[];
    edges: CustomEdgeModel[];
};

function DeploymentSideBar({ deploymentId, nodes, edges }: DeploymentSideBarProps) {
    // component state
    const { deployment, isLoading, error } = useFetchDeployment(deploymentId);
    const { activeKeyTab, onSelectTab } = useTabs({
        defaultTab: 'Details',
    });

    // derived values
    const deploymentNode = getNodeById(nodes, deploymentId);
    const numExternalFlows = getNumExternalFlows(nodes, edges, deploymentId);
    const numInternalFlows = getNumInternalFlows(nodes, edges, deploymentId);
    const listenPorts = getListenPorts(nodes, deploymentId);
    const deploymentPolicyIds =
        deploymentNode?.data.type === 'DEPLOYMENT' ? deploymentNode?.data?.policyIds : [];

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
            <StackItem>
                <Tabs activeKey={activeKeyTab} onSelect={onSelectTab}>
                    <Tab
                        eventKey="Details"
                        tabContentId="Details"
                        title={<TabTitleText>Details</TabTitleText>}
                    />
                    <Tab
                        eventKey="Flows"
                        tabContentId="Flows"
                        title={<TabTitleText>Flows</TabTitleText>}
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
                    />
                </Tabs>
            </StackItem>
            <StackItem isFilled style={{ overflow: 'auto' }}>
                <TabContent eventKey="Details" id="Details" hidden={activeKeyTab !== 'Details'}>
                    {deployment && (
                        <DeploymentDetails
                            deployment={deployment}
                            numExternalFlows={numExternalFlows}
                            numInternalFlows={numInternalFlows}
                            listenPorts={listenPorts}
                        />
                    )}
                </TabContent>
                <TabContent eventKey="Flows" id="Flows" hidden={activeKeyTab !== 'Flows'}>
                    <DeploymentFlows />
                </TabContent>
                <TabContent
                    eventKey="Baselines"
                    id="Baselines"
                    hidden={activeKeyTab !== 'Baselines'}
                    className="pf-u-h-100"
                >
                    <DeploymentBaselines />
                </TabContent>
                <TabContent
                    eventKey="Network policies"
                    id="Network policies"
                    hidden={activeKeyTab !== 'Network policies'}
                >
                    <NetworkPolicies policyIds={deploymentPolicyIds} />
                </TabContent>
            </StackItem>
        </Stack>
    );
}

export default DeploymentSideBar;
