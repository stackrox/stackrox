import React from 'react';

import {
    imageSearchFilterConfig,
    imageComponentSearchFilterConfig,
    deploymentSearchFilterConfig,
    namespaceSearchFilterConfig,
    clusterSearchFilterConfig,
} from 'Containers/Vulnerabilities/searchFilterConfig';
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
    return (
        <ImageCvePage
            searchFilterConfig={searchFilterConfig}
            showVulnerabilityStateTabs
            vulnerabilityState={vulnerabilityState}
        />
    );
}

export default ImageCvePageRoute;
