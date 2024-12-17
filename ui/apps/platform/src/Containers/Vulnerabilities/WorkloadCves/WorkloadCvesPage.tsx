import React, { useMemo } from 'react';
import { Route, Routes } from 'react-router-dom';
import { PageSection } from '@patternfly/react-core';

import PageNotFound from 'Components/PageNotFound';
import PageTitle from 'Components/PageTitle';

import ScannerV4IntegrationBanner from 'Components/ScannerV4IntegrationBanner';
import useFeatureFlags from 'hooks/useFeatureFlags';
import usePermissions from 'hooks/usePermissions';
import DeploymentPage from './Deployment/DeploymentPage';
import ImagePage from './Image/ImagePage';
import WorkloadCvesOverviewPage from './Overview/WorkloadCvesOverviewPage';
import ImageCvePage from './ImageCve/ImageCvePage';
import NamespaceViewPage from './NamespaceView/NamespaceViewPage';
import { WorkloadCveViewContext } from './WorkloadCveViewContext';

import './WorkloadCvesPage.css';

export type WorkloadCvePageProps = {
    view: 'user-workload' | 'platform-workload';
};

function WorkloadCvesPage({ view }: WorkloadCvePageProps) {
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const { hasReadAccess } = usePermissions();
    const hasReadAccessForIntegration = hasReadAccess('Integration');
    const hasReadAccessForNamespaces = hasReadAccess('Namespace');

    const context = useMemo(() => {
        const pageTitle = 'Workload CVEs'; // TODO Implement throughout in follow up
        const baseSearchFilter = isFeatureFlagEnabled('ROX_PLATFORM_CVE_SPLIT')
            ? { 'Platform Component': [String(view === 'platform-workload')] }
            : {};
        const getAbsoluteUrl = (subPath: string) =>
            view === 'platform-workload'
                ? `platform-workload-cves/${subPath}/*`
                : `workload-cves/${subPath}/*`;

        return { pageTitle, baseSearchFilter, getAbsoluteUrl };
    }, [view, isFeatureFlagEnabled]);

    return (
        <WorkloadCveViewContext.Provider value={context}>
            {hasReadAccessForIntegration && <ScannerV4IntegrationBanner />}
            <Routes>
                {hasReadAccessForNamespaces && (
                    <Route
                        path={context.getAbsoluteUrl('namespace-view')}
                        element={<NamespaceViewPage />}
                    />
                )}
                <Route path={context.getAbsoluteUrl('cves/:cveId')} element={<ImageCvePage />} />
                <Route path={context.getAbsoluteUrl('images/:imageId')} element={<ImagePage />} />
                <Route
                    path={context.getAbsoluteUrl('deployments/:deploymentId')}
                    element={<DeploymentPage />}
                />
                <Route index element={<WorkloadCvesOverviewPage />} />
                <Route
                    path="*"
                    element={
                        <PageSection variant="light">
                            <PageTitle title={`${context.pageTitle} - Not Found`} />
                            <PageNotFound />
                        </PageSection>
                    }
                />
            </Routes>
        </WorkloadCveViewContext.Provider>
    );
}

export default WorkloadCvesPage;
