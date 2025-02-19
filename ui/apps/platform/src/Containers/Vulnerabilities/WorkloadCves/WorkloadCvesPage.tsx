import React, { useMemo } from 'react';
import { Route, Switch } from 'react-router-dom';
import { PageSection } from '@patternfly/react-core';

import PageNotFound from 'Components/PageNotFound';
import PageTitle from 'Components/PageTitle';

import {
    vulnerabilitiesAllImagesPath,
    vulnerabilitiesImagesWithoutCvesPath,
    vulnerabilitiesInactiveImagesPath,
    vulnerabilitiesPlatformPath,
    vulnerabilitiesUserWorkloadsPath,
    vulnerabilitiesWorkloadCvesPath,
} from 'routePaths';
import ScannerV4IntegrationBanner from 'Components/ScannerV4IntegrationBanner';
import useFeatureFlags, { IsFeatureFlagEnabled } from 'hooks/useFeatureFlags';
import usePermissions from 'hooks/usePermissions';
import { NonEmptyArray } from 'utils/type.utils';
import DeploymentPage from './Deployment/DeploymentPage';
import ImagePage from './Image/ImagePage';
import WorkloadCvesOverviewPage from './Overview/WorkloadCvesOverviewPage';
import ImageCvePage from './ImageCve/ImageCvePage';
import NamespaceViewPage from './NamespaceView/NamespaceViewPage';
import { WorkloadCveViewContext } from './WorkloadCveViewContext';

import './WorkloadCvesPage.css';
import { QuerySearchFilter, WorkloadEntityTab } from '../types';

export const userWorkloadViewId = 'user-workloads';
export const platformViewId = 'platform';
export const allImagesViewId = 'all-images';
export const inactiveImagesViewId = 'inactive-images';
export const imagesWithoutCvesViewId = 'images-without-cves';

function getWorkloadCveContextFromView(viewId: string, isFeatureFlagEnabled: IsFeatureFlagEnabled) {
    let pageTitle: string = '';
    let pageTitleDescription: string | undefined;
    let baseSearchFilter: QuerySearchFilter = {};
    let getAbsoluteUrl: (subPath: string) => string = () => '';
    let overviewEntityTabs: NonEmptyArray<WorkloadEntityTab> = ['CVE', 'Image', 'Deployment'];

    switch (viewId) {
        case userWorkloadViewId:
            if (isFeatureFlagEnabled('ROX_PLATFORM_CVE_SPLIT')) {
                pageTitle = 'User workload vulnerabilities';
                pageTitleDescription =
                    'Vulnerabilities affecting user-managed workloads and images';
                baseSearchFilter = { 'Platform Component': ['false'] };
                getAbsoluteUrl = (subPath: string) =>
                    `${vulnerabilitiesUserWorkloadsPath}/${subPath}`;
            } else {
                pageTitle = 'Workload CVEs';
                baseSearchFilter = {};
                getAbsoluteUrl = (subPath: string) =>
                    `${vulnerabilitiesWorkloadCvesPath}/${subPath}`;
            }
            break;
        case platformViewId:
            pageTitle = 'Platform vulnerabilities';
            pageTitleDescription =
                'Vulnerabilities affecting images and workloads used by the OpenShift Platform and layered services';
            baseSearchFilter = { 'Platform Component': ['true'] };
            getAbsoluteUrl = (subPath: string) => `${vulnerabilitiesPlatformPath}/${subPath}`;
            break;
        case allImagesViewId:
            pageTitle = 'All vulnerable images';
            pageTitleDescription =
                'Findings for user, platform, and inactive images simultaneously';
            baseSearchFilter = { 'Platform Component': ['true', 'false', '-'] };
            getAbsoluteUrl = (subPath: string) => `${vulnerabilitiesAllImagesPath}/${subPath}`;
            break;
        case inactiveImagesViewId:
            pageTitle = 'Inactive images only';
            pageTitleDescription =
                'Findings for watched images and images not currently deployed as workloads based on your image retention settings';
            baseSearchFilter = { 'Platform Component': ['-'] };
            getAbsoluteUrl = (subPath: string) => `${vulnerabilitiesInactiveImagesPath}/${subPath}`;
            overviewEntityTabs = ['CVE', 'Image'];
            break;
        case imagesWithoutCvesViewId:
            pageTitle = 'Images without CVEs';
            pageTitleDescription =
                'Images and workloads without observed CVEs (results might include false negatives due to scanner limitations, such as unsupported operating systems)';
            baseSearchFilter = { 'Image CVE Count': ['0'] };
            getAbsoluteUrl = (subPath: string) =>
                `${vulnerabilitiesImagesWithoutCvesPath}/${subPath}`;
            overviewEntityTabs = ['Image', 'Deployment'];
            break;
        default:
        // TODO Handle user-defined views, or error
    }

    return {
        pageTitle,
        pageTitleDescription,
        baseSearchFilter,
        getAbsoluteUrl,
        overviewEntityTabs,
    };
}

export type WorkloadCvePageProps = {
    view: string;
};

function WorkloadCvesPage({ view }: WorkloadCvePageProps) {
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const { hasReadAccess } = usePermissions();
    const hasReadAccessForIntegration = hasReadAccess('Integration');
    const hasReadAccessForNamespaces = hasReadAccess('Namespace');

    const context = useMemo(
        () => getWorkloadCveContextFromView(view, isFeatureFlagEnabled),
        [view, isFeatureFlagEnabled]
    );

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
