import React from 'react';

import {
    imageSearchFilterConfig,
    imageComponentSearchFilterConfig,
    deploymentSearchFilterConfig,
    namespaceSearchFilterConfig,
    clusterSearchFilterConfig,
} from 'Containers/Vulnerabilities/searchFilterConfig';
import ImageCvePage from './ImageCvePage';

function ImageCvePageRoute() {
    const searchFilterConfig = [
        imageSearchFilterConfig,
        imageComponentSearchFilterConfig,
        deploymentSearchFilterConfig,
        namespaceSearchFilterConfig,
        clusterSearchFilterConfig,
    ];

    return <ImageCvePage searchFilterConfig={searchFilterConfig} showVulnerabilityStateTabs />;
}

export default ImageCvePageRoute;
