import React, { useMemo } from 'react';
import { Route, Switch } from 'react-router-dom';
import { PageSection } from '@patternfly/react-core';

import PageNotFound from 'Components/PageNotFound';
import PageTitle from 'Components/PageTitle';

import {
    vulnerabilitiesPlatformWorkloadCvesPath,
    vulnerabilitiesWorkloadCvesPath,
} from 'routePaths';
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
        const platformComponentFilters =
            view === 'platform-workload'
                ? ['true']
                : // The '-' filter is used to include inactive images in the "user-workload" view
                  ['false', '-'];
        const baseSearchFilter = isFeatureFlagEnabled('ROX_PLATFORM_CVE_SPLIT')
            ? { 'Platform Component': platformComponentFilters }
            : {};
        const getAbsoluteUrl = (subPath: string) =>
            view === 'platform-workload'
                ? `${vulnerabilitiesPlatformWorkloadCvesPath}/${subPath}`
                : `${vulnerabilitiesWorkloadCvesPath}/${subPath}`;

        return { pageTitle, baseSearchFilter, getAbsoluteUrl };
    }, [view, isFeatureFlagEnabled]);

    return (
        <WorkloadCveViewContext.Provider value={context}>
            {hasReadAccessForIntegration && <ScannerV4IntegrationBanner />}
            <Switch>
                {hasReadAccessForNamespaces && (
                    <Route path={context.getAbsoluteUrl('namespace-view')}>
                        <NamespaceViewPage />
                    </Route>
                )}
                <Route path={context.getAbsoluteUrl('cves/:cveId')}>
                    <ImageCvePage />
                </Route>
                <Route path={context.getAbsoluteUrl('images/:imageId')}>
                    <ImagePage />
                </Route>
                <Route path={context.getAbsoluteUrl('deployments/:deploymentId')}>
                    <DeploymentPage />
                </Route>
                <Route exact path={context.getAbsoluteUrl('')}>
                    <WorkloadCvesOverviewPage />
                </Route>
                <Route>
                    <PageSection variant="light">
                        <PageTitle title={`${context.pageTitle} - Not Found`} />
                        <PageNotFound />
                    </PageSection>
                </Route>
            </Switch>
        </WorkloadCveViewContext.Provider>
    );
}

export default WorkloadCvesPage;
