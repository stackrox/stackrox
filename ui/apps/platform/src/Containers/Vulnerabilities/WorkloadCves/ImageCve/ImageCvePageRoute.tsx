import React from 'react';

import { hideColumnIf } from 'hooks/useManagedColumns';
import useIsScannerV4Enabled from 'hooks/useIsScannerV4Enabled';

import {
    imageSearchFilterConfig,
    imageComponentSearchFilterConfig,
    deploymentSearchFilterConfig,
    namespaceSearchFilterConfig,
    clusterSearchFilterConfig,
} from '../../searchFilterConfig';
import ImageCvePage from './ImageCvePage';
import useVulnerabilityState from '../hooks/useVulnerabilityState';

function ImageCvePageRoute() {
    const searchFilterConfig = [
        imageSearchFilterConfig,
        imageComponentSearchFilterConfig,
        deploymentSearchFilterConfig,
        namespaceSearchFilterConfig,
        clusterSearchFilterConfig,
    ];

    const vulnerabilityState = useVulnerabilityState();
    const isScannerV4Enabled = useIsScannerV4Enabled();

    const imageTableColumnOverrides = {
        nvdCvss: hideColumnIf(!isScannerV4Enabled),
    };
    const deploymentTableColumnOverrides = {};

    return (
        <ImageCvePage
            searchFilterConfig={searchFilterConfig}
            showVulnerabilityStateTabs
            vulnerabilityState={vulnerabilityState}
            imageTableColumnOverrides={imageTableColumnOverrides}
            deploymentTableColumnOverrides={deploymentTableColumnOverrides}
        />
    );
}

export default ImageCvePageRoute;
