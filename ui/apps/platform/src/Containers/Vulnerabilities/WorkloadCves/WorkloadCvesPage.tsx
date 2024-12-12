import React from 'react';
import { Route, Switch } from 'react-router-dom';
import { PageSection } from '@patternfly/react-core';

import PageNotFound from 'Components/PageNotFound';
import PageTitle from 'Components/PageTitle';

import { vulnerabilitiesWorkloadCvesPath, vulnerabilityNamespaceViewPath } from 'routePaths';
import ScannerV4IntegrationBanner from 'Components/ScannerV4IntegrationBanner';
import usePermissions from 'hooks/usePermissions';
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
    const { hasReadAccess } = usePermissions();
    const hasReadAccessForIntegration = hasReadAccess('Integration');
    const hasReadAccessForNamespaces = hasReadAccess('Namespace');

    return (
        <>
            {hasReadAccessForIntegration && <ScannerV4IntegrationBanner />}
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
