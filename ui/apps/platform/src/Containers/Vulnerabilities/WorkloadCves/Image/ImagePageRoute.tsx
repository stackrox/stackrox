import React from 'react';

import ImagePage from './ImagePage';
import useVulnerabilityState from '../hooks/useVulnerabilityState';

function ImagePageRoute() {
    const vulnerabilityState = useVulnerabilityState();
    return <ImagePage showVulnerabilityStateTabs vulnerabilityState={vulnerabilityState} />;
}

export default ImagePageRoute;
