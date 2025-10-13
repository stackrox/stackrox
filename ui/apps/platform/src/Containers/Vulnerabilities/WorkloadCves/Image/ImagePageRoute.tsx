import React from 'react';

import ImagePage from './ImagePage';
import useVulnerabilityState from '../hooks/useVulnerabilityState';

function ImagePageRoute() {
    const vulnerabilityState = useVulnerabilityState();
    return (
        <ImagePage
            showVulnerabilityStateTabs
            vulnerabilityState={vulnerabilityState}
            deploymentResourceColumnOverrides={{}}
        />
    );
}

export default ImagePageRoute;
