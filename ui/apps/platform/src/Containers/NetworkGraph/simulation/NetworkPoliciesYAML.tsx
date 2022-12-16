import React from 'react';
import { CodeEditor, CodeEditorControl, Language } from '@patternfly/react-code-editor';
import {
    DownloadIcon,
    MoonIcon,
    ProcessAutomationIcon,
    SunIcon,
    UndoIcon,
} from '@patternfly/react-icons';

import { useTheme } from 'Containers/ThemeProvider';
import download from 'utils/download';

type NetworkPoliciesYAMLProp = {
    yaml: string;
    generateNetworkPolicies: () => void;
    undoNetworkPolicies: () => void;
};

const downloadYAMLHandler = (fileName: string, fileContent: string) => () => {
    download(`${fileName}.yml`, fileContent, 'yml');
};

function NetworkPoliciesYAML({
    yaml,
    generateNetworkPolicies,
    undoNetworkPolicies,
}: NetworkPoliciesYAMLProp) {
    const { isDarkMode } = useTheme();
    const [customDarkMode, setCustomDarkMode] = React.useState(isDarkMode);

    function onToggleDarkMode() {
        setCustomDarkMode((prevValue) => !prevValue);
    }

    const toggleDarkModeControl = (
        <CodeEditorControl
            icon={customDarkMode ? <SunIcon /> : <MoonIcon />}
            aria-label="Toggle dark mode"
            toolTipText={customDarkMode ? 'Toggle to light mode' : 'Toggle to dark mode'}
            onClick={onToggleDarkMode}
            isVisible
        />
    );

    const downloadYAMLControl = (
        <CodeEditorControl
            icon={<DownloadIcon />}
            aria-label="Download YAML"
            toolTipText="Download YAML"
            onClick={downloadYAMLHandler('networkPolicy', yaml)}
            isVisible
        />
    );

    const generateNewYAMLControl = (
        <CodeEditorControl
            icon={<ProcessAutomationIcon />}
            aria-label="Generate a new YAML"
            toolTipText="Generate a new YAML"
            onClick={generateNetworkPolicies}
            isVisible
        />
    );

    const revertRecentYAML = (
        <CodeEditorControl
            icon={<UndoIcon />}
            aria-label="Revert most recently applied YAML"
            toolTipText="Revert most recently applied YAML"
            onClick={undoNetworkPolicies}
            isVisible
        />
    );

    return (
        <div className="pf-u-h-100">
            <CodeEditor
                isDarkTheme={customDarkMode}
                customControls={[
                    toggleDarkModeControl,
                    downloadYAMLControl,
                    generateNewYAMLControl,
                    revertRecentYAML,
                ]}
                isCopyEnabled
                isLineNumbersVisible
                isReadOnly
                code={yaml}
                language={Language.yaml}
                height="300px"
            />
        </div>
    );
}

export default NetworkPoliciesYAML;
