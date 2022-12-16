import React from 'react';
import {
    Alert,
    Bullseye,
    Button,
    Checkbox,
    Divider,
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
import { FileUploadIcon } from '@patternfly/react-icons';

import useTabs from 'hooks/patternfly/useTabs';
import ViewActiveYAMLs from './ViewActiveYAMLs';
import useNetworkPolicySimulator from '../hooks/useNetworkPolicySimulator';
import NetworkPoliciesYAML from './NetworkPoliciesYAML';
import { getDisplayYAMLFromNetworkPolicyModification } from '../utils/simulatorUtils';

type NetworkPolicySimulatorSidePanelProps = {
    selectedClusterId: string;
};

const tabs = {
    SIMULATE_NETWORK_POLICIES: 'Simulate network policies',
    VIEW_ACTIVE_YAMLS: 'View active YAMLS',
};

function NetworkPolicySimulatorSidePanel({
    selectedClusterId,
}: NetworkPolicySimulatorSidePanelProps) {
    const { activeKeyTab, onSelectTab } = useTabs({
        defaultTab: tabs.SIMULATE_NETWORK_POLICIES,
    });
    const [isExcludingPortsAndProtocols, setIsExcludingPortsAndProtocols] =
        React.useState<boolean>(false);
    const { simulator, setNetworkPolicyModification } = useNetworkPolicySimulator({
        clusterId: selectedClusterId,
    });

    function generateNetworkPolicies() {
        setNetworkPolicyModification({
            state: 'GENERATED',
            options: {
                clusterId: selectedClusterId,
                searchQuery: '',
                networkDataSince: '',
                excludePortsAndProtocols: isExcludingPortsAndProtocols,
            },
        });
    }

    function undoNetworkPolicies() {
        setNetworkPolicyModification({
            state: 'UNDO',
            options: {
                clusterId: selectedClusterId,
            },
        });
    }

    if (simulator.isLoading) {
        return (
            <Bullseye>
                <Spinner isSVG size="lg" />
            </Bullseye>
        );
    }

    if (simulator.state === 'GENERATED') {
        const yaml = getDisplayYAMLFromNetworkPolicyModification(simulator.modification);
        return (
            <div>
                <Flex
                    direction={{ default: 'row' }}
                    alignItems={{ default: 'alignItemsFlexEnd' }}
                    className="pf-u-p-lg pf-u-mb-0"
                >
                    <FlexItem>
                        <TextContent>
                            <Text component={TextVariants.h2} className="pf-u-font-size-xl">
                                Network Policy Simulator
                            </Text>
                        </TextContent>
                    </FlexItem>
                </Flex>
                <Divider component="div" />
                <Stack hasGutter>
                    <StackItem className="pf-u-p-md">
                        <Alert
                            variant={simulator.error ? 'danger' : 'success'}
                            isInline
                            isPlain
                            title={
                                simulator.error
                                    ? simulator.error
                                    : 'Policies generated from all network activity'
                            }
                        />
                    </StackItem>
                    <StackItem isFilled style={{ overflow: 'auto' }}>
                        <NetworkPoliciesYAML
                            yaml={yaml}
                            generateNetworkPolicies={generateNetworkPolicies}
                            undoNetworkPolicies={undoNetworkPolicies}
                        />
                    </StackItem>
                </Stack>
            </div>
        );
    }

    // @TODO: Consider how to reuse parts of this that are similiar between states
    if (simulator.state === 'UNDO') {
        const yaml = getDisplayYAMLFromNetworkPolicyModification(simulator.modification);
        return (
            <div>
                <Flex
                    direction={{ default: 'row' }}
                    alignItems={{ default: 'alignItemsFlexEnd' }}
                    className="pf-u-p-lg pf-u-mb-0"
                >
                    <FlexItem>
                        <TextContent>
                            <Text component={TextVariants.h2} className="pf-u-font-size-xl">
                                Network Policy Simulator
                            </Text>
                        </TextContent>
                    </FlexItem>
                </Flex>
                <Divider component="div" />
                <Stack hasGutter>
                    <StackItem className="pf-u-p-md">
                        <Alert
                            variant={simulator.error ? 'danger' : 'success'}
                            isInline
                            isPlain
                            title={
                                simulator.error
                                    ? simulator.error
                                    : 'Viewing modification that will undo last applied change'
                            }
                        />
                    </StackItem>
                    <StackItem isFilled style={{ overflow: 'auto' }}>
                        <NetworkPoliciesYAML
                            yaml={yaml}
                            generateNetworkPolicies={generateNetworkPolicies}
                            undoNetworkPolicies={undoNetworkPolicies}
                        />
                    </StackItem>
                </Stack>
            </div>
        );
    }

    return (
        <Stack>
            <StackItem>
                <Flex direction={{ default: 'row' }} className="pf-u-p-lg pf-u-mb-0">
                    <FlexItem>
                        <TextContent>
                            <Text component={TextVariants.h2} className="pf-u-font-size-xl">
                                Simulate network policy
                            </Text>
                        </TextContent>
                    </FlexItem>
                </Flex>
            </StackItem>
            <StackItem>
                <Tabs activeKey={activeKeyTab} onSelect={onSelectTab}>
                    <Tab
                        eventKey={tabs.SIMULATE_NETWORK_POLICIES}
                        tabContentId={tabs.SIMULATE_NETWORK_POLICIES}
                        title={<TabTitleText>{tabs.SIMULATE_NETWORK_POLICIES}</TabTitleText>}
                    />
                    <Tab
                        eventKey={tabs.VIEW_ACTIVE_YAMLS}
                        tabContentId={tabs.VIEW_ACTIVE_YAMLS}
                        title={<TabTitleText>{tabs.VIEW_ACTIVE_YAMLS}</TabTitleText>}
                    />
                </Tabs>
            </StackItem>
            <StackItem isFilled style={{ overflow: 'auto' }}>
                <TabContent
                    eventKey={tabs.SIMULATE_NETWORK_POLICIES}
                    id={tabs.SIMULATE_NETWORK_POLICIES}
                    hidden={activeKeyTab !== tabs.SIMULATE_NETWORK_POLICIES}
                >
                    <div className="pf-u-p-lg pf-u-h-100">
                        <Stack hasGutter>
                            <StackItem>
                                <Stack hasGutter>
                                    <StackItem>
                                        <TextContent>
                                            <Text
                                                component={TextVariants.h2}
                                                className="pf-u-font-size-lg"
                                            >
                                                Generate network policies
                                            </Text>
                                        </TextContent>
                                    </StackItem>
                                    <StackItem>
                                        <TextContent>
                                            <Text component={TextVariants.p}>
                                                Generate a set of recommended network policies based
                                                on your environment&apos;s configuration. Select a
                                                time window for the network connections you would
                                                like to capture and generate policies on, and then
                                                apply them directly or share them with your team.
                                            </Text>
                                        </TextContent>
                                    </StackItem>
                                    <StackItem>
                                        <Checkbox
                                            label="Exclude ports & protocols"
                                            isChecked={isExcludingPortsAndProtocols}
                                            onChange={setIsExcludingPortsAndProtocols}
                                            id="controlled-check-1"
                                            name="check1"
                                        />
                                    </StackItem>
                                    <StackItem>
                                        <Button
                                            variant="secondary"
                                            onClick={generateNetworkPolicies}
                                        >
                                            Generate and simulate network policies
                                        </Button>
                                    </StackItem>
                                </Stack>
                            </StackItem>
                            <StackItem>
                                <Divider component="div" />
                            </StackItem>
                            <StackItem>
                                <Stack hasGutter>
                                    <StackItem>
                                        <TextContent>
                                            <Text
                                                component={TextVariants.h2}
                                                className="pf-u-font-size-lg"
                                            >
                                                Upload a network policy YAML
                                            </Text>
                                        </TextContent>
                                    </StackItem>
                                    <StackItem>
                                        <TextContent>
                                            <Text component={TextVariants.p}>
                                                Upload your network policies to quickly preview your
                                                environment under different policy configurations
                                                and time windows. When ready, apply the network
                                                policies directly or share them with your team.
                                            </Text>
                                        </TextContent>
                                    </StackItem>
                                    <StackItem>
                                        <Button variant="secondary" icon={<FileUploadIcon />}>
                                            Upload YAML
                                        </Button>
                                    </StackItem>
                                </Stack>
                            </StackItem>
                        </Stack>
                    </div>
                </TabContent>
                <TabContent
                    eventKey={tabs.VIEW_ACTIVE_YAMLS}
                    id={tabs.VIEW_ACTIVE_YAMLS}
                    hidden={activeKeyTab !== tabs.VIEW_ACTIVE_YAMLS}
                >
                    <ViewActiveYAMLs
                        networkPolicies={
                            simulator.state === 'ACTIVE' ? simulator.networkPolicies : []
                        }
                    />
                </TabContent>
            </StackItem>
        </Stack>
    );
}

export default NetworkPolicySimulatorSidePanel;
