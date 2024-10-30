import React, { useEffect } from 'react';
import {
    Bullseye,
    EmptyState,
    Stack,
    StackItem,
    EmptyStateHeader,
    DropEvent,
} from '@patternfly/react-core';
import { SelectOption } from '@patternfly/react-core/deprecated';
import { NetworkPolicy } from 'types/networkPolicy.proto';
import SelectSingle from 'Components/SelectSingle';
import NetworkPoliciesYAML from './NetworkPoliciesYAML';
import NetworkSimulatorActions from './NetworkSimulatorActions';

type ViewActiveYamlsProps = {
    networkPolicies: NetworkPolicy[];
    generateNetworkPolicies: () => void;
    undoNetworkPolicies: () => void;
    onFileInputChange: (_event: DropEvent, file: File) => void;
};

function ViewActiveYamls({
    networkPolicies,
    generateNetworkPolicies,
    undoNetworkPolicies,
    onFileInputChange,
}: ViewActiveYamlsProps) {
    const [selectedNetworkPolicy, setSelectedNetworkPolicy] = React.useState<
        NetworkPolicy | undefined
    >(networkPolicies?.[0]);

    useEffect(() => {
        if (networkPolicies?.length && !selectedNetworkPolicy) {
            setSelectedNetworkPolicy(networkPolicies[0]);
        }
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [networkPolicies]);

    function handleSelectedNetworkPolicy(_, value: string) {
        const newlySelectedNetworkPolicy = networkPolicies?.find(
            (networkPolicy) => networkPolicy.name === value
        );
        setSelectedNetworkPolicy(newlySelectedNetworkPolicy);
    }

    if (networkPolicies.length === 0) {
        return (
            <Bullseye>
                <EmptyState variant="xs">
                    <EmptyStateHeader titleText="No network policies" headingLevel="h4" />
                </EmptyState>
            </Bullseye>
        );
    }

    return (
        <div className="pf-v5-u-h-100">
            <Stack>
                <StackItem>
                    <div className="pf-v5-u-p-md">
                        <SelectSingle
                            id="search-filter-attributes-select"
                            value={selectedNetworkPolicy?.name || ''}
                            handleSelect={handleSelectedNetworkPolicy}
                            placeholderText="Select a network policy"
                        >
                            {networkPolicies.map((networkPolicy) => {
                                return (
                                    <SelectOption
                                        key={networkPolicy.name}
                                        value={networkPolicy.name}
                                    >
                                        {networkPolicy.name}
                                    </SelectOption>
                                );
                            })}
                        </SelectSingle>
                    </div>
                </StackItem>
                {selectedNetworkPolicy && (
                    <StackItem>
                        <NetworkPoliciesYAML yaml={selectedNetworkPolicy.yaml} />
                    </StackItem>
                )}
                {selectedNetworkPolicy && (
                    <StackItem>
                        <NetworkSimulatorActions
                            generateNetworkPolicies={generateNetworkPolicies}
                            undoNetworkPolicies={undoNetworkPolicies}
                            onFileInputChange={onFileInputChange}
                        />
                    </StackItem>
                )}
            </Stack>
        </div>
    );
}

export default ViewActiveYamls;
