import React from 'react';
import { CodeEditor, CodeEditorControl, Language } from '@patternfly/react-code-editor';
import { DownloadIcon } from '@patternfly/react-icons';

import { useTheme } from 'Containers/ThemeProvider';
import useAnalytics, { DOWNLOAD_NETWORK_POLICIES } from 'hooks/useAnalytics';
import useURLSearch from 'hooks/useURLSearch';
import download from 'utils/download';
import CodeEditorDarkModeControl from 'Components/PatternFly/CodeEditorDarkModeControl';
import { getPropertiesForAnalytics } from '../utils/networkGraphURLUtils';

import './NetworkPoliciesYAML.css';

type NetworkPoliciesYAMLProp = {
    yaml: string;
    height?: string;
    additionalControls?: React.ReactNode[];
};

const labels = {
    downloadYAML: 'Download YAML',
};

function NetworkPoliciesYAML({
    yaml,
    height = '300px',
    additionalControls = [],
}: NetworkPoliciesYAMLProp) {
    const { analyticsTrack } = useAnalytics();
    const { searchFilter } = useURLSearch();

    const { isDarkMode } = useTheme();
    const [customDarkMode, setCustomDarkMode] = React.useState(isDarkMode);

    function onToggleDarkMode() {
        setCustomDarkMode((prevValue) => !prevValue);
    }

    const downloadYAMLHandler = (fileName: string, fileContent: string) => () => {
        const properties = getPropertiesForAnalytics(searchFilter);

        analyticsTrack({
            event: DOWNLOAD_NETWORK_POLICIES,
            properties,
        });

        download(`${fileName}.yml`, fileContent, 'yml');
    };

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
        <div className="network-policies-yaml pf-u-h-100">
            <CodeEditor
                isDarkTheme={customDarkMode}
                customControls={[
                    <CodeEditorDarkModeControl
                        isDarkMode={customDarkMode}
                        onToggleDarkMode={onToggleDarkMode}
                    />,
                    downloadYAMLControl,
                    ...additionalControls,
                ]}
                isCopyEnabled
                isLineNumbersVisible
                isReadOnly
                code={yaml}
                language={Language.yaml}
                height={height}
            />
        </div>
    );
}

export default NetworkPoliciesYAML;
