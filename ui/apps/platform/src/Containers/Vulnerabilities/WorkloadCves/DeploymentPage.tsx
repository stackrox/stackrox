import React from 'react';
import { useParams } from 'react-router-dom';

function DeploymentPage() {
    const { deploymentId } = useParams();
    return <>Workload CVE Deployment Single Page: {deploymentId}</>;
}

export default DeploymentPage;
