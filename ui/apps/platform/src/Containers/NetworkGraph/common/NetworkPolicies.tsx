import React from 'react';
import { Button, SelectOption, Stack, StackItem } from '@patternfly/react-core';
import { CodeEditor, CodeEditorControl, Language } from '@patternfly/react-code-editor';

import download from 'utils/download';
import SelectSingle from 'Components/SelectSingle';
import { useTheme } from 'Containers/ThemeProvider';
import { MoonIcon, SunIcon } from '@patternfly/react-icons';

type NetworkPolicy = {
    name: string;
    yaml: string;
};

type NetworkPoliciesProps = {
    networkPolicies: NetworkPolicy[];
};

const downloadYAMLHandler = (fileContent: string) => () => {
    download('network-policy.yml', fileContent, 'yml');
};

function NetworkPolicies({ networkPolicies }: NetworkPoliciesProps): React.ReactElement {
    const { isDarkMode } = useTheme();
    const [customDarkMode, setCustomDarkMode] = React.useState(isDarkMode);
    const [selectedNetworkPolicy, setSelectedNetworkPolicy] = React.useState<
        NetworkPolicy | undefined
    >(networkPolicies[0]);

    function onToggleDarkMode() {
        setCustomDarkMode((prevValue) => !prevValue);
    }

    function handleSelectedNetworkPolicy(_, value: string) {
        const newlySelectedNetworkPolicy = networkPolicies.find(
            (networkPolicy) => networkPolicy.name === value
        );
        setSelectedNetworkPolicy(newlySelectedNetworkPolicy);
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

    return (
        <Stack hasGutter>
            <StackItem>
                <SelectSingle
                    id="search-filter-attributes-select"
                    value={selectedNetworkPolicy?.name || ''}
                    handleSelect={handleSelectedNetworkPolicy}
                    placeholderText="Select a network policy"
                >
                    {networkPolicies.map((networkPolicy) => {
                        return (
                            <SelectOption key={networkPolicy.name} value={networkPolicy.name}>
                                {networkPolicy.name}
                            </SelectOption>
                        );
                    })}
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
                    <Button onClick={downloadYAMLHandler(selectedNetworkPolicy.yaml)}>
                        Export YAML
                    </Button>
                </StackItem>
            )}
        </Stack>
    );
}

export default NetworkPolicies;
