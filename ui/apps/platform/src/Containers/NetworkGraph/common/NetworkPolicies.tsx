import React, { useEffect, useMemo } from 'react';
import {
    Alert,
    AlertVariant,
    Bullseye,
    Button,
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
    policyIds: string[];
};

type NetworkPolicyYAML = {
    name: string;
    yaml: string;
};

const downloadYAMLHandler = (fileName: string, fileContent: string) => () => {
    download(`${fileName}.yml`, fileContent, 'yml');
};

function NetworkPolicies({ policyIds }: NetworkPoliciesProps): React.ReactElement {
    const { networkPolicies, isLoading, error } = useFetchNetworkPolicies(policyIds);
    const { isDarkMode } = useTheme();
    const [customDarkMode, setCustomDarkMode] = React.useState(isDarkMode);

    const allNetworkPoliciesYAML = useMemo(
        () => ({
            name: 'All network policies',
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
        if (value !== 'All network policies') {
            const newlySelectedNetworkPolicy = networkPolicies.find(
                (networkPolicy) => networkPolicy.name === value
            );
            setSelectedNetworkPolicy(newlySelectedNetworkPolicy);
        } else {
            setSelectedNetworkPolicy(allNetworkPoliciesYAML);
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
                        <SelectOption key="All network policies" value="All network policies">
                            All network policies
                        </SelectOption>
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
                        <Button
                            onClick={downloadYAMLHandler(
                                selectedNetworkPolicy.name,
                                selectedNetworkPolicy.yaml
                            )}
                        >
                            Export YAML
                        </Button>
                    </StackItem>
                )}
            </Stack>
        </div>
    );
}

export default NetworkPolicies;
