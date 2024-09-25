import React from 'react';
import { Route, Switch } from 'react-router-dom';
import { PageSection } from '@patternfly/react-core';

import PageNotFound from 'Components/PageNotFound';
import PageTitle from 'Components/PageTitle';

import {
    vulnManagementPath,
    vulnerabilitiesWorkloadCvesPath,
    vulnerabilityNamespaceViewPath,
} from 'routePaths';
import TechPreviewBanner from 'Components/TechPreviewBanner';
import ScannerV4IntegrationBanner from 'Components/ScannerV4IntegrationBanner';
import usePermissions from 'hooks/usePermissions';
import useFeatureFlags from 'hooks/useFeatureFlags';
import DeploymentPage from './Deployment/DeploymentPage';
import ImagePage from './Image/ImagePage';
import WorkloadCvesOverviewPage from './Overview/WorkloadCvesOverviewPage';
import ImageCvePage from './ImageCve/ImageCvePage';
import NamespaceViewPage from './NamespaceView/NamespaceViewPage';

import './WorkloadCvesPage.css';

const vulnerabilitiesWorkloadCveSinglePath = `${vulnerabilitiesWorkloadCvesPath}/cves/:cveId`;
const vulnerabilitiesWorkloadCveImageSinglePath = `${vulnerabilitiesWorkloadCvesPath}/images/:imageId`;
const vulnerabilitiesWorkloadCveDeploymentSinglePath = `${vulnerabilitiesWorkloadCvesPath}/deployments/:deploymentId`;

function WorkloadCvesPage() {
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const { hasReadAccess } = usePermissions();
    const hasReadAccessForIntegration = hasReadAccess('Integration');
    const hasReadAccessForNamespaces = hasReadAccess('Namespace');

    return (
        <>
            {hasReadAccessForIntegration && <ScannerV4IntegrationBanner />}
            {!isFeatureFlagEnabled('ROX_VULN_MGMT_2_GA') && (
                <TechPreviewBanner
                    featureURL={vulnManagementPath}
                    featureName="Vulnerability Management (1.0)"
                    routeKey="vulnerability-management"
                />
            )}
            <Switch>
                {hasReadAccessForNamespaces && (
                    <Route path={vulnerabilityNamespaceViewPath}>
                        <NamespaceViewPage />
                    </Route>
                )}
                <Route path={vulnerabilitiesWorkloadCveSinglePath}>
                    <ImageCvePage />
                </Route>
                <Route path={vulnerabilitiesWorkloadCveImageSinglePath}>
                    <ImagePage />
                </Route>
                <Route path={vulnerabilitiesWorkloadCveDeploymentSinglePath}>
                    <DeploymentPage />
                </Route>
                <Route exact path={vulnerabilitiesWorkloadCvesPath}>
                    <WorkloadCvesOverviewPage />
                </Route>
                <Route>
                    <PageSection variant="light">
                        <PageTitle title="Workload CVEs - Not Found" />
                        <PageNotFound />
                    </PageSection>
                </Route>
            </Switch>
        </>
    );
}

export default WorkloadCvesPage;
