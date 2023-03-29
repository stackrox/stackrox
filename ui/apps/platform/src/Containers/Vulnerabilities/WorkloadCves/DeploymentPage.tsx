import React from 'react';
import { useParams } from 'react-router-dom';

import PageTitle from 'Components/PageTitle';

function DeploymentPage() {
    const { deploymentId } = useParams();
    return (
        <>
            <PageTitle title={`Workload CVEs - Deployment ${'TODO'}`} />
            Workload CVE Deployment Single Page: {deploymentId}
        </>
    );
}

export default DeploymentPage;
