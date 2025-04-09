import React, { CSSProperties, ReactNode } from 'react';

import { CodeBlockAction, Button } from '@patternfly/react-core';
import { DownloadIcon } from '@patternfly/react-icons';

import CodeViewer from 'Components/CodeViewer';
import useAnalytics, { DOWNLOAD_NETWORK_POLICIES } from 'hooks/useAnalytics';
import useURLSearch from 'hooks/useURLSearch';
import download from 'utils/download';
import { getPropertiesForAnalytics } from '../utils/networkGraphURLUtils';

type NetworkPoliciesYAMLProp = {
    yaml: string;
    style?: CSSProperties;
    additionalControls?: ReactNode;
};

function NetworkPoliciesYAML({ yaml, style, additionalControls }: NetworkPoliciesYAMLProp) {
    const { analyticsTrack } = useAnalytics();
    const { searchFilter } = useURLSearch();

    const downloadYAMLHandler = (fileName: string, fileContent: string) => () => {
        const properties = getPropertiesForAnalytics(searchFilter);

        analyticsTrack({
            event: DOWNLOAD_NETWORK_POLICIES,
            properties,
        });

        download(`${fileName}.yml`, fileContent, 'yml');
    };
    return (
        <div className="network-policies-yaml pf-v5-u-h-100">
            <CodeViewer
                code={yaml}
                style={style}
                additionalControls={
                    <>
                        {
                            <CodeBlockAction>
                                <Button
                                    variant="plain"
                                    aria-label="Download YAML"
                                    onClick={downloadYAMLHandler('networkPolicy', yaml)}
                                    icon={<DownloadIcon />}
                                />
                            </CodeBlockAction>
                        }
                        {additionalControls}
                    </>
                }
            />
        </div>
    );
}

export default NetworkPoliciesYAML;
