import { Route, Routes } from 'react-router-dom-v5-compat';
import { PageSection } from '@patternfly/react-core';

import PageNotFound from 'Components/PageNotFound';

import CveListPage from './CveList/CveListPage';
import CveDetailPage from './CveDetail/CveDetailPage';
import DeploymentListPage from './Deployments/DeploymentListPage';
import DeploymentDetailPage from './Deployments/DeploymentDetailPage';

/**
 * React Router routes for the CVE prototype pages.
 *
 * Expected to be mounted at the `vulnerabilitiesPrototypePath` base.
 */
function ProtoRoutes() {
    return (
        <Routes>
            <Route path="cves/:cveName" element={<CveDetailPage />} />
            <Route path="cves" element={<CveListPage />} />
            <Route
                path="deployments/:deploymentId"
                element={<DeploymentDetailPage />}
            />
            <Route path="deployments" element={<DeploymentListPage />} />
            <Route index element={<CveListPage />} />
            <Route
                path="*"
                element={
                    <PageSection hasBodyWrapper={false}>
                        <PageNotFound />
                    </PageSection>
                }
            />
        </Routes>
    );
}

export default ProtoRoutes;
