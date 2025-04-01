import React from 'react';
import { Route, Routes } from 'react-router-dom';
import { PageSection } from '@patternfly/react-core';

import PageNotFound from 'Components/PageNotFound';
import PageTitle from 'Components/PageTitle';
import ScannerV4IntegrationBanner from 'Components/ScannerV4IntegrationBanner';
import usePermissions from 'hooks/usePermissions';

import PlatformCvesOverviewPage from './Overview/PlatformCvesOverviewPage';
import PlatformCvePage from './PlatformCve/PlatformCvePage';
import ClusterPage from './Cluster/ClusterPage';

function PlatformCvesPage() {
    const { hasReadAccess } = usePermissions();
    const hasReadAccessForIntegration = hasReadAccess('Integration');

    return (
        <>
            {hasReadAccessForIntegration && <ScannerV4IntegrationBanner />}
            <Routes>
                <Route index element={<PlatformCvesOverviewPage />} />
                <Route path="cves/:cveId" element={<PlatformCvePage />} />
                <Route path="clusters/:clusterId" element={<ClusterPage />} />
                <Route
                    path="*"
                    element={
                        <PageSection variant="light">
                            <PageTitle title="Platform CVEs - Not Found" />
                            <PageNotFound />
                        </PageSection>
                    }
                />
            </Routes>
        </>
    );
}

export default PlatformCvesPage;
