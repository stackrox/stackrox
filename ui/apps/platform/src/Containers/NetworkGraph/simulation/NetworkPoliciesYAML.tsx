import React from 'react';
import { CodeEditor, CodeEditorControl, Language } from '@patternfly/react-code-editor';
import { DownloadIcon, MoonIcon, SunIcon } from '@patternfly/react-icons';

import { useTheme } from 'Containers/ThemeProvider';
import download from 'utils/download';

type NetworkPoliciesYAMLProp = {
    yaml: string;
};

const labels = {
    downloadYAML: 'Download YAML',
};

const downloadYAMLHandler = (fileName: string, fileContent: string) => () => {
    download(`${fileName}.yml`, fileContent, 'yml');
};

function NetworkPoliciesYAML({ yaml }: NetworkPoliciesYAMLProp) {
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
            aria-label={labels.downloadYAML}
            toolTipText={labels.downloadYAML}
            onClick={downloadYAMLHandler('networkPolicy', yaml)}
            isVisible
        />
    );

    return (
        <div className="pf-u-h-100">
            <CodeEditor
                isDarkTheme={customDarkMode}
                customControls={[toggleDarkModeControl, downloadYAMLControl]}
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
