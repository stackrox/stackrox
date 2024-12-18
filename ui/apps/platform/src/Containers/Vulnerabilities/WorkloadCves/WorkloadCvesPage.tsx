import React from 'react';
import { Route, Routes } from 'react-router-dom';
import { PageSection } from '@patternfly/react-core';

import PageNotFound from 'Components/PageNotFound';
import PageTitle from 'Components/PageTitle';
import { vulnerabilitiesWorkloadCvesPath } from 'routePaths';

import ScannerV4IntegrationBanner from 'Components/ScannerV4IntegrationBanner';
import usePermissions from 'hooks/usePermissions';
import DeploymentPage from './Deployment/DeploymentPage';
import ImagePage from './Image/ImagePage';
import WorkloadCvesOverviewPage from './Overview/WorkloadCvesOverviewPage';
import ImageCvePage from './ImageCve/ImageCvePage';
import NamespaceViewPage from './NamespaceView/NamespaceViewPage';
import { WorkloadCveViewContext } from './WorkloadCveViewContext';

import './WorkloadCvesPage.css';

const vulnerabilitiesWorkloadCveSinglePath = `cves/:cveId`;
const vulnerabilitiesWorkloadCveImageSinglePath = `images/:imageId`;
const vulnerabilitiesWorkloadCveDeploymentSinglePath = `deployments/:deploymentId`;

const userWorkloadContext = {
    pageTitle: 'Workload CVEs', // TODO Implement throughout in follow up
    baseSearchFilter: {}, // TODO Implement throughout in follow up
    createUrl: (path) => `${vulnerabilitiesWorkloadCvesPath}${path}`, // TODO Implement throughout in follow up
};

// TODO Update these values for Platform View
const platformWorkloadContext = {
    pageTitle: 'Workload CVEs', // TODO Implement throughout in follow up
    baseSearchFilter: {}, // TODO Implement throughout in follow up
    createUrl: (path) => `${vulnerabilitiesWorkloadCvesPath}${path}`, // TODO Implement throughout in follow up
};

export type WorkloadCvePageProps = {
    view: 'user-workload' | 'platform-workload';
};

function WorkloadCvesPage({ view }: WorkloadCvePageProps) {
    const { hasReadAccess } = usePermissions();
    const hasReadAccessForIntegration = hasReadAccess('Integration');
    const hasReadAccessForNamespaces = hasReadAccess('Namespace');

    const context = view === 'user-workload' ? userWorkloadContext : platformWorkloadContext;

    return (
        <WorkloadCveViewContext.Provider value={context}>
            {hasReadAccessForIntegration && <ScannerV4IntegrationBanner />}
            <Routes>
                <Route index element={<WorkloadCvesOverviewPage />} />
                {hasReadAccessForNamespaces && (
                    <Route path={'namespace-view'} element={<NamespaceViewPage />} />
                )}
                <Route path={vulnerabilitiesWorkloadCveSinglePath} element={<ImageCvePage />} />
                <Route path={vulnerabilitiesWorkloadCveImageSinglePath} element={<ImagePage />} />
                <Route
                    path={vulnerabilitiesWorkloadCveDeploymentSinglePath}
                    element={<DeploymentPage />}
                />
                <Route
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
