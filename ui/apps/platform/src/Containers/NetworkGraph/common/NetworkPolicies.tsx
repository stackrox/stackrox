import React, { useEffect, useMemo } from 'react';
import {
    Alert,
    AlertVariant,
    Bullseye,
    Button,
    Divider,
    EmptyState,
    EmptyStateVariant,
    SelectOption,
    Spinner,
    Stack,
    StackItem,
    Title,
} from '@patternfly/react-core';
import { CodeEditor, CodeEditorControl, Language } from '@patternfly/react-code-editor';
import { MoonIcon, SunIcon } from '@patternfly/react-icons';

import download from 'utils/download';
import SelectSingle from 'Components/SelectSingle';
import { useTheme } from 'Containers/ThemeProvider';
import useFetchNetworkPolicies from 'hooks/useFetchNetworkPolicies';

type NetworkPoliciesProps = {
    entityName: string;
    policyIds: string[];
};

type NetworkPolicyYAML = {
    name: string;
    yaml: string;
};

const allNetworkPoliciesId = 'network-policy-combined-yaml-pf-key';

function NetworkPolicies({ entityName, policyIds }: NetworkPoliciesProps): React.ReactElement {
    const { networkPolicies, isLoading, error } = useFetchNetworkPolicies(policyIds);
    const { isDarkMode } = useTheme();
    const [customDarkMode, setCustomDarkMode] = React.useState(isDarkMode);

    const allNetworkPoliciesYAML = useMemo(
        () => ({
            name: allNetworkPoliciesId,
            yaml: networkPolicies.map((networkPolicy) => networkPolicy.yaml).join('---\n'),
        }),
        [networkPolicies]
    );

    const [selectedNetworkPolicy, setSelectedNetworkPolicy] = React.useState<
        NetworkPolicyYAML | undefined
    >(allNetworkPoliciesYAML);

    useEffect(() => {
        setSelectedNetworkPolicy(allNetworkPoliciesYAML);
    }, [allNetworkPoliciesYAML]);

    function onToggleDarkMode() {
        setCustomDarkMode((prevValue) => !prevValue);
    }

    function handleSelectedNetworkPolicy(_, value: string) {
        if (value !== allNetworkPoliciesId) {
            const newlySelectedNetworkPolicy = networkPolicies.find(
                (networkPolicy) => networkPolicy.name === value
            );
            setSelectedNetworkPolicy(newlySelectedNetworkPolicy);
        } else {
            setSelectedNetworkPolicy(allNetworkPoliciesYAML);
        }
    }

    function exportYAMLHandler() {
        if (selectedNetworkPolicy) {
            const fileName =
                selectedNetworkPolicy.name === allNetworkPoliciesId
                    ? entityName
                    : selectedNetworkPolicy.name;
            const fileContent = selectedNetworkPolicy.yaml;
            download(`${fileName}.yml`, fileContent, 'yml');
        }
    }

    const customControl = (
        <CodeEditorControl
            icon={customDarkMode ? <SunIcon /> : <MoonIcon />}
            aria-label="Toggle dark mode"
            toolTipText={customDarkMode ? 'Toggle to light mode' : 'Toggle to dark mode'}
            onClick={onToggleDarkMode}
            isVisible
        />
    );

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

    if (networkPolicies.length === 0) {
        return (
            <Bullseye>
                <EmptyState variant={EmptyStateVariant.xs}>
                    <Title headingLevel="h4" size="md">
                        No network policies
                    </Title>
                </EmptyState>
            </Bullseye>
        );
    }

    return (
        <div className="pf-u-h-100 pf-u-p-md">
            <Stack hasGutter>
                <StackItem>
                    <SelectSingle
                        id="search-filter-attributes-select"
                        value={selectedNetworkPolicy?.name || ''}
                        handleSelect={handleSelectedNetworkPolicy}
                        placeholderText="Select a network policy"
                    >
                        <SelectOption value="all">All network policies</SelectOption>
                        <Divider component="li" />
                        <>
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
                        </>
                    </SelectSingle>
                </StackItem>
                {selectedNetworkPolicy && (
                    <StackItem>
                        <div className="pf-u-h-100">
                            <CodeEditor
                                isDarkTheme={customDarkMode}
                                customControls={customControl}
                                isCopyEnabled
                                isLineNumbersVisible
                                isReadOnly
                                code={selectedNetworkPolicy.yaml}
                                language={Language.yaml}
                                height="300px"
                            />
                        </div>
                    </StackItem>
                )}
                {selectedNetworkPolicy && (
                    <StackItem>
                        <Button onClick={exportYAMLHandler}>Export YAML</Button>
                    </StackItem>
                )}
            </Stack>
        </div>
    );
}

export default NetworkPolicies;
