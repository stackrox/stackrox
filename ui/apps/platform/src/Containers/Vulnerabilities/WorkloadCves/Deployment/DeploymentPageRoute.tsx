import React from 'react';

import DeploymentPage from './DeploymentPage';
import useVulnerabilityState from '../hooks/useVulnerabilityState';

function DeploymentPageRoute() {
    const vulnerabilityState = useVulnerabilityState();
    return <DeploymentPage showVulnerabilityStateTabs vulnerabilityState={vulnerabilityState} />;
}

export default DeploymentPageRoute;
