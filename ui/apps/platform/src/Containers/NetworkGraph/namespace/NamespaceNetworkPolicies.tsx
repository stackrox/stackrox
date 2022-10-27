import React from 'react';
import { Button, Stack, StackItem } from '@patternfly/react-core';
import { CodeEditor, CodeEditorControl, Language } from '@patternfly/react-code-editor';

import download from 'utils/download';
import { useTheme } from 'Containers/ThemeProvider';
import { MoonIcon, SunIcon } from '@patternfly/react-icons';

const yamlText = `apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: ''
  namespace: managed-service-registry
`;

const downloadYAMLHandler = (fileContent) => () => {
    download('network-policy.yml', fileContent, 'yml');
};

function NamespaceNetworkPolicies() {
    const { isDarkMode } = useTheme();
    const [customDarkMode, setCustomDarkMode] = React.useState(isDarkMode);

    const onToggleDarkMode = () => {
        setCustomDarkMode((prevValue) => !prevValue);
    };

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
        <div className="pf-u-h-100 pf-u-p-md">
            <Stack hasGutter>
                <StackItem>
                    <CodeEditor
                        isDarkTheme={customDarkMode}
                        customControls={customControl}
                        isCopyEnabled
                        isLineNumbersVisible
                        isReadOnly
                        isMinimapVisible
                        code={yamlText}
                        language={Language.yaml}
                        height="450px"
                    />
                </StackItem>
                <StackItem>
                    <Button onClick={downloadYAMLHandler(yamlText)}>Export YAML</Button>
                </StackItem>
            </Stack>
        </div>
    );
}

export default NamespaceNetworkPolicies;
