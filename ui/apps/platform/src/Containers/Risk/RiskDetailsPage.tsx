import type { ReactElement } from 'react';
import { useParams } from 'react-router-dom-v5-compat';

import RiskSidePanel from './RiskSidePanel';

function RiskDetailsPage(): ReactElement {
    const params = useParams();
    const { deploymentId } = params;

    return <RiskSidePanel selectedDeploymentId={deploymentId as string} />;
}

export default RiskDetailsPage;
