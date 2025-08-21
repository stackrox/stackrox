import React, { useMemo } from 'react';
import { Route, Routes } from 'react-router-dom-v5-compat';
import { PageSection } from '@patternfly/react-core';

import PageNotFound from 'Components/PageNotFound';
import PageTitle from 'Components/PageTitle';

import {
    vulnerabilitiesAllImagesPath,
    vulnerabilitiesImagesWithoutCvesPath,
    vulnerabilitiesInactiveImagesPath,
    vulnerabilitiesPlatformPath,
    vulnerabilitiesUserWorkloadsPath,
} from 'routePaths';
import ScannerV4IntegrationBanner from 'Components/ScannerV4IntegrationBanner';
import useFeatureFlags, { IsFeatureFlagEnabled } from 'hooks/useFeatureFlags';
import usePermissions from 'hooks/usePermissions';
import { NonEmptyArray } from 'utils/type.utils';
import type { VulnerabilityState } from 'types/cve.proto';

import DeploymentPage from './Deployment/DeploymentPage';
import ImagePage from './Image/ImagePage';
import WorkloadCvesOverviewPage from './Overview/WorkloadCvesOverviewPage';
import ImageCvePage from './ImageCve/ImageCvePage';
import NamespaceViewPage from './NamespaceView/NamespaceViewPage';
import { WorkloadCveView, WorkloadCveViewContext } from './WorkloadCveViewContext';

import './WorkloadCvesPage.css';
import { QuerySearchFilter, WorkloadEntityTab } from '../types';
import { getOverviewPagePath, getWorkloadEntityPagePath } from '../utils/searchUtils';

export const userWorkloadViewId = 'user-workloads';
export const platformViewId = 'platform';
export const allImagesViewId = 'all-images';
export const inactiveImagesViewId = 'inactive-images';
export const imagesWithoutCvesViewId = 'images-without-cves';

function getUrlBuilder(viewId: string): WorkloadCveView['urlBuilder'] {
    let urlRoot = '';
    let cveBase: 'Workload' | 'Node' | 'Platform' = 'Workload';

    switch (viewId) {
        case userWorkloadViewId:
            urlRoot = vulnerabilitiesUserWorkloadsPath;
            cveBase = 'Workload';
            break;
        case platformViewId:
            urlRoot = vulnerabilitiesPlatformPath;
            cveBase = 'Platform';
            break;
        case allImagesViewId:
            urlRoot = vulnerabilitiesAllImagesPath;
            cveBase = 'Workload';
            break;
        case inactiveImagesViewId:
            urlRoot = vulnerabilitiesInactiveImagesPath;
            cveBase = 'Workload';
            break;
        case imagesWithoutCvesViewId:
            urlRoot = vulnerabilitiesImagesWithoutCvesPath;
            cveBase = 'Workload';
            break;
        default:
            // TODO Handle user-defined views, or error
            break;
    }

    function getAbsoluteUrl(subPath: string) {
        return `${urlRoot}/${subPath}`;
    }

    return {
        vulnMgmtBase: getAbsoluteUrl,
        cveList: (vulnerabilityState: VulnerabilityState) =>
            getAbsoluteUrl(getOverviewPagePath(cveBase, { vulnerabilityState, entityTab: 'CVE' })),
        cveDetails: (cve: string, vulnerabilityState: VulnerabilityState) =>
            getAbsoluteUrl(getWorkloadEntityPagePath('CVE', cve, vulnerabilityState)),
        imageList: (vulnerabilityState: VulnerabilityState) =>
            getAbsoluteUrl(
                getOverviewPagePath(cveBase, { vulnerabilityState, entityTab: 'Image' })
            ),
        imageDetails: (id: string, vulnerabilityState: VulnerabilityState) =>
            getAbsoluteUrl(getWorkloadEntityPagePath('Image', id, vulnerabilityState)),
        workloadList: (vulnerabilityState: VulnerabilityState) =>
            getAbsoluteUrl(
                getOverviewPagePath(cveBase, { vulnerabilityState, entityTab: 'Deployment' })
            ),
        workloadDetails: (
            workload: {
                id: string;
                namespace: string;
                name: string;
                type: string;
            },
            vulnerabilityState: VulnerabilityState
        ) =>
            getAbsoluteUrl(
                getWorkloadEntityPagePath('Deployment', workload.id, vulnerabilityState)
            ),
    };
}

function getWorkloadCveContextFromView(
    viewId: string,
    isFeatureFlagEnabled: IsFeatureFlagEnabled
): WorkloadCveView {
    let pageTitle: string = '';
    let pageTitleDescription: string | undefined;
    let baseSearchFilter: QuerySearchFilter = {};
    let overviewEntityTabs: NonEmptyArray<WorkloadEntityTab> = ['CVE', 'Image', 'Deployment'];
    let viewContext: string = '';

    switch (viewId) {
        case userWorkloadViewId:
            if (isFeatureFlagEnabled('ROX_PLATFORM_CVE_SPLIT')) {
                pageTitle = 'User workload vulnerabilities';
                pageTitleDescription =
                    'Vulnerabilities affecting user-managed workloads and images';
                baseSearchFilter = { 'Platform Component': ['false'] };
                viewContext = 'User workloads';
            } else {
                pageTitle = 'Workload CVEs';
                baseSearchFilter = {};
                viewContext = 'Workload CVEs';
            }
            break;
        case platformViewId:
            pageTitle = 'Platform vulnerabilities';
            pageTitleDescription =
                'Vulnerabilities affecting images and workloads used by the OpenShift Platform and layered services';
            baseSearchFilter = { 'Platform Component': ['true'] };
            viewContext = 'Platform';
            break;
        case allImagesViewId:
            pageTitle = 'All vulnerable images';
            pageTitleDescription =
                'Findings for user, platform, and inactive images simultaneously';
            baseSearchFilter = { 'Platform Component': ['true', 'false', '-'] };
            viewContext = 'All vulnerable images';
            break;
        case inactiveImagesViewId:
            pageTitle = 'Inactive images only';
            pageTitleDescription =
                'Findings for watched images and images not currently deployed as workloads based on your image retention settings';
            baseSearchFilter = { 'Platform Component': ['-'] };
            overviewEntityTabs = ['CVE', 'Image'];
            viewContext = 'Inactive images';
            break;
        case imagesWithoutCvesViewId:
            pageTitle = 'Images without CVEs';
            pageTitleDescription =
                'Images and workloads without observed CVEs (results might include false negatives due to scanner limitations, such as unsupported operating systems)';
            baseSearchFilter = { 'Image CVE Count': ['0'] };
            overviewEntityTabs = ['Image', 'Deployment'];
            viewContext = 'Images without CVEs';
            break;
        default:
        // TODO Handle user-defined views, or error
    }

    return {
        pageTitle,
        pageTitleDescription,
        baseSearchFilter,
        urlBuilder: getUrlBuilder(viewId),
        overviewEntityTabs,
        viewContext,
    };
}

export type WorkloadCvesPageProps = {
    view: string;
};

function WorkloadCvesPage({ view }: WorkloadCvesPageProps) {
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
            <Routes>
                {hasReadAccessForNamespaces && (
                    <Route path={'namespace-view'} element={<NamespaceViewPage />} />
                )}
                <Route path={'cves/:cveId'} element={<ImageCvePage />} />
                <Route path={'images/:imageId'} element={<ImagePage />} />
                <Route path={'deployments/:deploymentId'} element={<DeploymentPage />} />
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
