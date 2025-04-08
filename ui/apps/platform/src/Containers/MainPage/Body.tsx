import React, { ElementType, ReactElement, useEffect } from 'react';
import { useSelector } from 'react-redux';
import { Navigate, Route, Routes, useLocation, useParams } from 'react-router-dom';
import { PageSection } from '@patternfly/react-core';

// Import path variables in alphabetical order to minimize merge conflicts when multiple people add routes.
import {
    RouteKey,
    accessControlBasePath,
    administrationEventsPathWithParam,
    apidocsPath,
    apidocsPathV2,
    clustersDelegatedScanningPath,
    clustersDiscoveredClustersPath,
    clustersInitBundlesPathWithParam,
    clustersClusterRegistrationSecretsPathWithParam,
    clustersPathWithParam,
    clustersSecureClusterPath,
    clustersSecureClusterCrsPath,
    collectionsPath,
    complianceEnhancedCoveragePath,
    complianceEnhancedSchedulesPath,
    complianceBasePath,
    configManagementPath,
    dashboardPath,
    exceptionConfigurationPath,
    deprecatedPoliciesPath,
    integrationsPath,
    isRouteEnabled, // predicate function
    listeningEndpointsBasePath,
    mainPath,
    networkPath,
    policyManagementBasePath,
    riskPath,
    searchPath,
    systemConfigPath,
    systemHealthPath,
    userBasePath,
    violationsBasePath,
    vulnManagementPath,
    vulnerabilitiesWorkloadCvesPath,
    vulnerabilityReportsPath,
    exceptionManagementPath,
    vulnerabilitiesNodeCvesPath,
    vulnerabilitiesPlatformCvesPath,
    policiesBasePath,
    vulnerabilitiesUserWorkloadsPath,
    vulnerabilitiesPlatformPath,
    vulnerabilitiesAllImagesPath,
    vulnerabilitiesInactiveImagesPath,
    vulnerabilitiesImagesWithoutCvesPath,
} from 'routePaths';

import PageNotFound from 'Components/PageNotFound';
import PageTitle from 'Components/PageTitle';
import ErrorBoundary from 'Components/PatternFly/ErrorBoundary/ErrorBoundary';
import usePermissions, { HasReadAccess } from 'hooks/usePermissions';
import { IsFeatureFlagEnabled } from 'hooks/useFeatureFlags';
import useAnalytics from 'hooks/useAnalytics';
import { selectors } from 'reducers';

import asyncComponent from './AsyncComponent';
import InviteUsersModal from './InviteUsers/InviteUsersModal';

function NotFoundPage(): ReactElement {
    return (
        <PageSection variant="light">
            <PageTitle title="Not Found" />
            <PageNotFound />
        </PageSection>
    );
}

function makeVulnMgmtUserWorkloadView(view: string) {
    const AsyncWorkloadCvesComponent = asyncComponent(
        () => import('Containers/Vulnerabilities/WorkloadCves/WorkloadCvesPage')
    );
    return function WorkloadCvesPage() {
        return <AsyncWorkloadCvesComponent view={view} />;
    };
}

type RouteComponent = {
    component: ElementType;
    path: string | string[];
};

// routeComponentMap corresponds to routeRequirementsMap in src/routePaths.ts file.
// Add route keys in alphabetical order to minimize merge conflicts when multiple people add routes.
const routeComponentMap: Record<RouteKey, RouteComponent> = {
    'access-control': {
        component: asyncComponent(() => import('Containers/AccessControl/AccessControl')),
        path: accessControlBasePath,
    },
    'administration-events': {
        component: asyncComponent(
            () => import('Containers/Administration/Events/AdministrationEventsRoute')
        ),
        path: administrationEventsPathWithParam,
    },
    apidocs: {
        component: asyncComponent(() => import('Containers/Docs/ApiPage')),
        path: apidocsPath,
    },
    'apidocs-v2': {
        component: asyncComponent(() => import('Containers/Docs/ApiPageV2')),
        path: apidocsPathV2,
    },
    // Delegated image scanning must precede generic Clusters.
    'clusters/delegated-image-scanning': {
        component: asyncComponent(
            () => import('Containers/Clusters/DelegateScanning/DelegateScanningPage')
        ),
        path: clustersDelegatedScanningPath,
    },
    // Discovered clusters must precede generic Clusters.
    'clusters/discovered-clusters': {
        component: asyncComponent(
            () => import('Containers/Clusters/DiscoveredClusters/DiscoveredClustersPage')
        ),
        path: clustersDiscoveredClustersPath,
    },
    // Cluster init bundles must precede generic Clusters.
    'clusters/init-bundles': {
        component: asyncComponent(() => import('Containers/Clusters/InitBundles/InitBundlesRoute')),
        path: clustersInitBundlesPathWithParam,
    },
    // Cluster registration secrets must precede generic Clusters.
    'clusters/cluster-registration-secrets': {
        component: asyncComponent(
            () =>
                import(
                    'Containers/Clusters/ClusterRegistrationSecrets/ClusterRegistrationSecretsRoute'
                )
        ),
        path: clustersClusterRegistrationSecretsPathWithParam,
    },
    // Cluster secure-a-cluster must precede generic Clusters.
    'clusters/secure-a-cluster': {
        component: asyncComponent(
            () => import('Containers/Clusters/InitBundles/SecureClusterPage')
        ),
        path: clustersSecureClusterPath,
    },
    // Cluster secure-a-cluster-crs must precede generic Clusters.
    'clusters/secure-a-cluster-crs': {
        component: asyncComponent(
            () => import('Containers/Clusters/ClusterRegistrationSecrets/SecureClusterPage')
        ),
        path: clustersSecureClusterCrsPath,
    },
    clusters: {
        component: asyncComponent(() => import('Containers/Clusters/ClustersPage')),
        path: clustersPathWithParam,
    },
    collections: {
        component: asyncComponent(() => import('Containers/Collections/CollectionsPage')),
        path: collectionsPath,
    },
    'compliance-coverage': {
        component: asyncComponent(
            () => import('Containers/ComplianceEnhanced/Coverage/CoveragePage')
        ),
        path: complianceEnhancedCoveragePath,
    },
    'compliance-schedules': {
        component: asyncComponent(
            () => import('Containers/ComplianceEnhanced/Schedules/ScanConfigsPage')
        ),
        path: complianceEnhancedSchedulesPath,
    },
    compliance: {
        component: asyncComponent(() => import('Containers/Compliance/Page')),
        path: complianceBasePath,
    },
    configmanagement: {
        component: asyncComponent(() => import('Containers/ConfigManagement/ConfigManagementRoutes')),
        path: configManagementPath,
    },
    dashboard: {
        component: asyncComponent(() => import('Containers/Dashboard/DashboardPage')),
        path: dashboardPath,
    },
    'exception-configuration': {
        component: asyncComponent(
            () => import('Containers/ExceptionConfiguration/ExceptionConfigurationPage')
        ),
        path: exceptionConfigurationPath,
    },
    integrations: {
        component: asyncComponent(() => import('Containers/Integrations/IntegrationsPage')),
        path: integrationsPath,
    },
    'listening-endpoints': {
        component: asyncComponent(
            () => import('Containers/Audit/ListeningEndpoints/ListeningEndpointsPage')
        ),
        path: listeningEndpointsBasePath,
    },
    'network-graph': {
        component: asyncComponent(() => import('Containers/NetworkGraph/NetworkGraphPage')),
        path: networkPath,
    },
    'policy-management': {
        component: asyncComponent(() => import('Containers/PolicyManagement/PolicyManagementPage')),
        path: policyManagementBasePath,
    },
    risk: {
        component: asyncComponent(() => import('Containers/Risk/RiskPage')),
        path: riskPath,
    },
    search: {
        component: asyncComponent(() => import('Containers/Search/SearchPage')),
        path: searchPath,
    },
    'system-health': {
        component: asyncComponent(() => import('Containers/SystemHealth/DashboardPage')),
        path: systemHealthPath,
    },
    systemconfig: {
        component: asyncComponent(() => import('Containers/SystemConfig/SystemConfigPage')),
        path: systemConfigPath,
    },
    user: {
        component: asyncComponent(() => import('Containers/User/UserPage')),
        path: userBasePath,
    },
    violations: {
        component: asyncComponent(() => import('Containers/Violations/ViolationsPage')),
        path: violationsBasePath,
    },
    'vulnerabilities/exception-management': {
        component: asyncComponent(
            () => import('Containers/Vulnerabilities/ExceptionManagement/ExceptionManagementPage')
        ),
        path: exceptionManagementPath,
    },
    'vulnerabilities/node-cves': {
        component: asyncComponent(() => import('Containers/Vulnerabilities/NodeCves/NodeCvesPage')),
        path: vulnerabilitiesNodeCvesPath,
    },
    'vulnerabilities/platform-cves': {
        component: asyncComponent(
            () => import('Containers/Vulnerabilities/PlatformCves/PlatformCvesPage')
        ),
        path: vulnerabilitiesPlatformCvesPath,
    },
    'vulnerabilities/user-workloads': {
        component: makeVulnMgmtUserWorkloadView('user-workloads'),
        path: vulnerabilitiesUserWorkloadsPath,
    },
    // Note: currently 'platform' is an implementation of the user-workloads view and
    // it is expected that this will change in the future as these views diverge
    'vulnerabilities/platform': {
        component: makeVulnMgmtUserWorkloadView('platform'),
        path: vulnerabilitiesPlatformPath,
    },
    'vulnerabilities/all-images': {
        component: makeVulnMgmtUserWorkloadView('all-images'),
        path: vulnerabilitiesAllImagesPath,
    },
    'vulnerabilities/inactive-images': {
        component: makeVulnMgmtUserWorkloadView('inactive-images'),
        path: vulnerabilitiesInactiveImagesPath,
    },
    'vulnerabilities/images-without-cves': {
        component: makeVulnMgmtUserWorkloadView('images-without-cves'),
        path: vulnerabilitiesImagesWithoutCvesPath,
    },
    'vulnerabilities/reports': {
        component: asyncComponent(
            () => import('Containers/Vulnerabilities/VulnerablityReporting/VulnReportingPage')
        ),
        path: vulnerabilityReportsPath,
    },
    'vulnerabilities/workload-cves': {
        component: makeVulnMgmtUserWorkloadView('user-workloads'),
        path: vulnerabilitiesWorkloadCvesPath,
    },
    'vulnerability-management': {
        component: asyncComponent(() => import('Containers/VulnMgmt/WorkflowLayout')),
        path: vulnManagementPath,
    },
};

type BodyProps = {
    hasReadAccess: HasReadAccess;
    isFeatureFlagEnabled: IsFeatureFlagEnabled;
};

function WorkloadCvesRedirect() {
    const location = useLocation();

    const newPath = location.pathname.replace(
        vulnerabilitiesWorkloadCvesPath,
        vulnerabilitiesAllImagesPath
    );

    return <Navigate to={`${newPath}${location.search}`} replace />;
}

function DeprecatedPoliciesRedirect() {
    const { policyId, command } = useParams();

    const newPath = `${policiesBasePath}${policyId ? `/${policyId}` : ''}${
        command ? `/${command}` : ''
    }`;

    return <Navigate to={newPath} replace />;
}

function Body({ hasReadAccess, isFeatureFlagEnabled }: BodyProps): ReactElement {
    const location = useLocation();
    const { analyticsPageVisit } = useAnalytics();
    useEffect(() => {
        analyticsPageVisit('Page Viewed', '', { path: location.pathname });
    }, [location.pathname, analyticsPageVisit]);
    const { hasReadWriteAccess } = usePermissions();
    const hasWriteAccessForInviting = hasReadWriteAccess('Access');
    const showInviteModal = useSelector(selectors.inviteSelector);

    const routePredicates = { hasReadAccess, isFeatureFlagEnabled };

    return (
        <div className="flex flex-col h-full w-full relative overflow-auto bg-base-100">
            <ErrorBoundary>
                <Routes>
                    <Route path="/" element={<Navigate to={dashboardPath} replace />} />
                    <Route path={mainPath} element={<Navigate to={dashboardPath} replace />} />
                    {/* Make sure the following Redirect element works after react-router-dom upgrade */}
                    <Route path={deprecatedPoliciesPath} element={<DeprecatedPoliciesRedirect />} />
                    {isFeatureFlagEnabled('ROX_PLATFORM_CVE_SPLIT') && (
                        <Route
                            // all prior workload-cves routes must redirect to the new path.
                            path={`${vulnerabilitiesWorkloadCvesPath}/*`}
                            // Since all subpaths and query parameters must be retained, we need to do
                            // a search and replace of the subpath we are redirecting, which is accomplished
                            // by using the WorkloadCvesRedirect component.
                            element={<WorkloadCvesRedirect />}
                        />
                    )}
                    {Object.keys(routeComponentMap)
                        .filter((routeKey) => isRouteEnabled(routePredicates, routeKey as RouteKey))
                        .map((routeKey) => {
                            const { component: Component, path } = routeComponentMap[routeKey];
                            return (
                                <Route key={routeKey} path={`${path}/*`} element={<Component />} />
                            );
                        })}
                    <Route path="*" element={<NotFoundPage />} />
                </Routes>
                {hasWriteAccessForInviting && showInviteModal && <InviteUsersModal />}
            </ErrorBoundary>
        </div>
    );
}

export default Body;
