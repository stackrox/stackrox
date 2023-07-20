import React from 'react';
import {
    Alert,
    Bullseye,
    Button,
    Checkbox,
    Divider,
    Flex,
    FlexItem,
    Popover,
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
import { HelpIcon } from '@patternfly/react-icons';

import useTabs from 'hooks/patternfly/useTabs';
import ViewActiveYAMLs from './ViewActiveYAMLs';
import {
    NetworkPolicySimulator,
    SetNetworkPolicyModification,
} from '../hooks/useNetworkPolicySimulator';
import NetworkPoliciesYAML from './NetworkPoliciesYAML';
import { getDisplayYAMLFromNetworkPolicyModification } from '../utils/simulatorUtils';
import UploadYAMLButton from './UploadYAMLButton';
import NetworkSimulatorActions from './NetworkSimulatorActions';
import NotifyYAMLModal from './NotifyYAMLModal';
import { NetworkScopeHierarchy } from '../types/networkScopeHierarchy';

type NetworkPolicySimulatorSidePanelProps = {
    simulator: NetworkPolicySimulator;
    setNetworkPolicyModification: SetNetworkPolicyModification;
    scopeHierarchy: NetworkScopeHierarchy;
};

const tabs = {
    SIMULATE_NETWORK_POLICIES: 'Simulate network policies',
    VIEW_ACTIVE_YAMLS: 'View active YAMLS',
};

function NetworkPolicySimulatorSidePanel({
    simulator,
    setNetworkPolicyModification,
    scopeHierarchy,
}: NetworkPolicySimulatorSidePanelProps) {
    const { activeKeyTab, onSelectTab } = useTabs({
        defaultTab: tabs.SIMULATE_NETWORK_POLICIES,
    });
    const [isExcludingPortsAndProtocols, setIsExcludingPortsAndProtocols] =
        React.useState<boolean>(false);
    const [isNotifyModalOpen, setIsNotifyModalOpen] = React.useState(false);

    function handleFileInputChange(
        _event: React.ChangeEvent<HTMLInputElement> | React.DragEvent<HTMLElement>,
        file: File
    ) {
        if (file && !file.name.includes('.yaml')) {
            setNetworkPolicyModification({
                state: 'UPLOAD',
                options: {
                    modification: null,
                    error: 'File must be .yaml',
                },
            });
        } else {
            const reader = new FileReader();
            reader.onload = () => {
                const fileAsBinaryString = reader.result;
                setNetworkPolicyModification({
                    state: 'UPLOAD',
                    options: {
                        modification: {
                            applyYaml: fileAsBinaryString as string,
                            toDelete: [],
                        },
                        error: '',
                    },
                });
            };
            reader.onerror = () => {
                setNetworkPolicyModification({
                    state: 'UPLOAD',
                    options: {
                        modification: null,
                        error: 'Could not read file',
                    },
                });
            };
            reader.readAsBinaryString(file);
        }
    }

    function generateNetworkPolicies() {
        setNetworkPolicyModification({
            state: 'GENERATED',
            options: {
                scopeHierarchy,
                networkDataSince: '',
                excludePortsAndProtocols: isExcludingPortsAndProtocols,
            },
        });
    }

    function undoNetworkPolicies() {
        setNetworkPolicyModification({
            state: 'UNDO',
            options: {
                clusterId: scopeHierarchy.cluster.id,
            },
        });
    }

    function openNotifyYAMLModal() {
        setIsNotifyModalOpen(true);
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
                                    : `Policies generated from the baseline for cluster “${scopeHierarchy.cluster.name}”`
                            }
                        />
                    </StackItem>
                    <StackItem isFilled style={{ overflow: 'auto' }}>
                        <NetworkPoliciesYAML yaml={yaml} />
                    </StackItem>
                    <StackItem>
                        <NetworkSimulatorActions
                            generateNetworkPolicies={generateNetworkPolicies}
                            undoNetworkPolicies={undoNetworkPolicies}
                            onFileInputChange={handleFileInputChange}
                            openNotifyYAMLModal={openNotifyYAMLModal}
                        />
                    </StackItem>
                </Stack>
                <NotifyYAMLModal
                    isModalOpen={isNotifyModalOpen}
                    setIsModalOpen={setIsNotifyModalOpen}
                    clusterId={scopeHierarchy.cluster.id}
                    modification={simulator.modification}
                />
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
                        <NetworkPoliciesYAML yaml={yaml} />
                    </StackItem>
                    <StackItem>
                        <NetworkSimulatorActions
                            generateNetworkPolicies={generateNetworkPolicies}
                            undoNetworkPolicies={undoNetworkPolicies}
                            onFileInputChange={handleFileInputChange}
                            openNotifyYAMLModal={openNotifyYAMLModal}
                        />
                    </StackItem>
                </Stack>
                <NotifyYAMLModal
                    isModalOpen={isNotifyModalOpen}
                    setIsModalOpen={setIsNotifyModalOpen}
                    clusterId={scopeHierarchy.cluster.id}
                    modification={simulator.modification}
                />
            </div>
        );
    }

    if (simulator.state === 'UPLOAD') {
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
                                simulator.error ? simulator.error : 'Uploaded policies processed'
                            }
                        />
                    </StackItem>
                    <StackItem isFilled style={{ overflow: 'auto' }}>
                        <NetworkPoliciesYAML yaml={yaml} />
                    </StackItem>
                    <StackItem>
                        <NetworkSimulatorActions
                            generateNetworkPolicies={generateNetworkPolicies}
                            undoNetworkPolicies={undoNetworkPolicies}
                            onFileInputChange={handleFileInputChange}
                            openNotifyYAMLModal={openNotifyYAMLModal}
                        />
                    </StackItem>
                </Stack>
                <NotifyYAMLModal
                    isModalOpen={isNotifyModalOpen}
                    setIsModalOpen={setIsNotifyModalOpen}
                    clusterId={scopeHierarchy.cluster.id}
                    modification={simulator.modification}
                />
            </div>
        );
    }

    return (
        <Stack>
            <StackItem>
                <Flex direction={{ default: 'row' }} className="pf-u-p-lg pf-u-mb-0">
                    <FlexItem>
                        <TextContent>
                            <Text
                                component={TextVariants.h2}
                                className="pf-u-font-size-xl pf-u-mr-xl"
                            >
                                Simulate network policy for cluster “{scopeHierarchy.cluster.name}”
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
                                                Generate network policies based on the baseline
                                                <Popover
                                                    showClose={false}
                                                    bodyContent={
                                                        <div>
                                                            <p className="pf-u-mb-sm">
                                                                A baseline is considered the trusted
                                                                traffic (incoming and outgoing) for
                                                                a given entity, like a cluster,
                                                                namespace, or deployment.
                                                            </p>
                                                            <p className="pf-u-mb-sm">
                                                                It is automatically generated for
                                                                every deployment, by collecting
                                                                incoming and outgoing traffic during
                                                                its first hour of existence.
                                                            </p>
                                                            <p>
                                                                In addition, a user can modify the
                                                                baseline by adding or removing any
                                                                active flows that have been observed
                                                                over a period of time.
                                                            </p>
                                                        </div>
                                                    }
                                                >
                                                    <button
                                                        type="button"
                                                        aria-label="More info on network baselines"
                                                        onClick={(e) => e.preventDefault()}
                                                        className="pf-u-mx-sm pf-u-mt-xs"
                                                    >
                                                        <HelpIcon />
                                                    </button>
                                                </Popover>
                                            </Text>
                                        </TextContent>
                                    </StackItem>
                                    <StackItem>
                                        <TextContent>
                                            <Text component={TextVariants.p}>
                                                Generate a set of recommended network policies based
                                                on your cluster baseline. Cluster baseline is the
                                                aggregatation of the baselines of the deployments
                                                that belong to the cluster.
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
                                        <UploadYAMLButton
                                            onFileInputChange={handleFileInputChange}
                                        />
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
                        generateNetworkPolicies={generateNetworkPolicies}
                        undoNetworkPolicies={undoNetworkPolicies}
                        onFileInputChange={handleFileInputChange}
                    />
                </TabContent>
            </StackItem>
        </Stack>
    );
}

export default NetworkPolicySimulatorSidePanel;
