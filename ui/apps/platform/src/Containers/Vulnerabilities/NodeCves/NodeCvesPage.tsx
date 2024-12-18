import React from 'react';
import { Route, Routes } from 'react-router-dom';
import { PageSection } from '@patternfly/react-core';

import PageNotFound from 'Components/PageNotFound';
import PageTitle from 'Components/PageTitle';
import ScannerV4IntegrationBanner from 'Components/ScannerV4IntegrationBanner';
import usePermissions from 'hooks/usePermissions';
import NodeCvesOverviewPage from './Overview/NodeCvesOverviewPage';
import NodeCvePage from './NodeCve/NodeCvePage';
import NodePage from './Node/NodePage';

function NodeCvesPage() {
    const { hasReadAccess } = usePermissions();
    const hasReadAccessForIntegration = hasReadAccess('Integration');

    return (
        <>
            {hasReadAccessForIntegration && <ScannerV4IntegrationBanner />}
            <Routes>
                <Route path="cves/:cveId" element={<NodeCvePage />} />
                <Route path="nodes/:nodeId" element={<NodePage />} />
                <Route index element={<NodeCvesOverviewPage />} />
                <Route
                    path="*"
                    element={
                        <PageSection variant="light">
                            <PageTitle title="Node CVEs - Not Found" />
                            <PageNotFound />
                        </PageSection>
                    }
                />
            </Routes>
        </>
    );
}

export default NodeCvesPage;
