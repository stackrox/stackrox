import { Suspense, lazy, useEffect } from 'react';
import type { ElementType, ReactElement } from 'react';
import { useSelector } from 'react-redux';
import { Navigate, Route, Routes, useLocation, useParams } from 'react-router-dom-v5-compat';
import { Bullseye, PageSection, Spinner } from '@patternfly/react-core';

// Import path variables in alphabetical order to minimize merge conflicts when multiple people add routes.
import {
    accessControlBasePath,
    administrationEventsPathWithParam,
    apidocsPath,
    apidocsPathV2,
    baseImagesPath,
    clustersClusterRegistrationSecretsPathWithParam,
    clustersDelegatedScanningPath,
    clustersDiscoveredClustersPath,
    clustersInitBundlesPathWithParam,
    clustersPathWithParam,
    clustersSecureClusterCrsPath,
    clustersSecureClusterPath,
    collectionsPath,
    complianceEnhancedCoveragePath,
    complianceEnhancedSchedulesPath,
    configManagementPath,
    dashboardPath,
    deprecatedPoliciesPath,
    exceptionConfigurationPath,
    exceptionManagementPath,
    integrationsPath,
    isRouteEnabled, // predicate function
    listeningEndpointsBasePath,
    mainPath,
    networkPath,
    policiesBasePath,
    policyManagementBasePath,
    riskBasePath,
    riskSecretsBasePath,
    riskWorkloadsBasePath,
    searchPath,
    systemConfigPath,
    systemHealthPath,
    userBasePath,
    violationsBasePath,
    vulnManagementPath,
    vulnerabilitiesAllImagesPath,
    vulnerabilitiesImagesWithoutCvesPath,
    vulnerabilitiesInactiveImagesPath,
    vulnerabilitiesNodeCvesPath,
    vulnerabilitiesPlatformPath,
    vulnerabilitiesUserWorkloadsPath,
    vulnerabilitiesVirtualMachineCvesPath,
    vulnerabilitiesWorkloadCvesPath,
    vulnerabilityImageReportsPath,
    vulnerabilityNodeReportsPath,
    vulnerabilityReportsPath,
} from 'routePaths';
import type { RouteKey } from 'routePaths';

import PageNotFound from 'Components/PageNotFound';
import PageTitle from 'Components/PageTitle';
import usePermissions from 'hooks/usePermissions';
import type { HasReadAccess } from 'hooks/usePermissions';
import type { IsFeatureFlagEnabled } from 'hooks/useFeatureFlags';
import useAnalytics from 'hooks/useAnalytics';
import { selectors } from 'reducers';

import InviteUsersModal from './InviteUsers/InviteUsersModal';

function NotFoundPage(): ReactElement {
    return (
        <PageSection hasBodyWrapper={false}>
            <PageTitle title="Not Found" />
            <PageNotFound />
        </PageSection>
    );
}

const WorkloadCvesPage = lazy(
    () => import('Containers/Vulnerabilities/WorkloadCves/WorkloadCvesPage')
);

type RouteComponent = {
    component: ElementType;
    path: string | string[];
    componentKey?: string; // key of the component in the routeComponentMap to prevent state preservation across duplicate components
    props?: Partial<Record<string, unknown>>;
};

// routeComponentMap corresponds to routeRequirementsMap in src/routePaths.ts file.
// Add route keys in alphabetical order to minimize merge conflicts when multiple people add routes.
const routeComponentMap: Record<RouteKey, RouteComponent> = {
    'access-control': {
        component: lazy(() => import('Containers/AccessControl/AccessControl')),
        path: accessControlBasePath,
    },
    'administration-events': {
        component: lazy(() => import('Containers/Administration/Events/AdministrationEventsRoute')),
        path: administrationEventsPathWithParam,
    },
    apidocs: {
        component: lazy(() => import('Containers/Docs/ApiPage')),
        path: apidocsPath,
    },
    'apidocs-v2': {
        component: lazy(() => import('Containers/Docs/ApiPageV2')),
        path: apidocsPathV2,
    },
    // Delegated image scanning must precede generic Clusters.
    'clusters/delegated-image-scanning': {
        component: lazy(() => import('Containers/Clusters/DelegateScanning/DelegateScanningPage')),
        path: clustersDelegatedScanningPath,
    },
    // Discovered clusters must precede generic Clusters.
    'clusters/discovered-clusters': {
        component: lazy(
            () => import('Containers/Clusters/DiscoveredClusters/DiscoveredClustersPage')
        ),
        path: clustersDiscoveredClustersPath,
    },
    // Cluster init bundles must precede generic Clusters.
    'clusters/init-bundles': {
        component: lazy(() => import('Containers/Clusters/InitBundles/InitBundlesRoute')),
        path: clustersInitBundlesPathWithParam,
    },
    // Cluster registration secrets must precede generic Clusters.
    'clusters/cluster-registration-secrets': {
        component: lazy(
            () =>
                import('Containers/Clusters/ClusterRegistrationSecrets/ClusterRegistrationSecretsRoute')
        ),
        path: clustersClusterRegistrationSecretsPathWithParam,
    },
    // Cluster secure-a-cluster must precede generic Clusters.
    'clusters/secure-a-cluster': {
        component: lazy(() => import('Containers/Clusters/InitBundles/SecureClusterPage')),
        path: clustersSecureClusterPath,
    },
    // Cluster secure-a-cluster-crs must precede generic Clusters.
    'clusters/secure-a-cluster-crs': {
        component: lazy(
            () => import('Containers/Clusters/ClusterRegistrationSecrets/SecureClusterPage')
        ),
        path: clustersSecureClusterCrsPath,
    },
    clusters: {
        component: lazy(() => import('Containers/Clusters/ClustersPage')),
        path: clustersPathWithParam,
    },
    collections: {
        component: lazy(() => import('Containers/Collections/CollectionsPage')),
        path: collectionsPath,
    },
    'compliance-coverage': {
        component: lazy(() => import('Containers/ComplianceEnhanced/Coverage/CoveragePage')),
        path: complianceEnhancedCoveragePath,
    },
    'compliance-schedules': {
        component: lazy(() => import('Containers/ComplianceEnhanced/Schedules/ScanConfigsPage')),
        path: complianceEnhancedSchedulesPath,
    },
    configmanagement: {
        component: lazy(() => import('Containers/ConfigManagement/ConfigManagementRoutes')),
        path: configManagementPath,
    },
    dashboard: {
        component: lazy(() => import('Containers/Dashboard/DashboardPage')),
        path: dashboardPath,
    },
    'exception-configuration': {
        component: lazy(
            () => import('Containers/ExceptionConfiguration/ExceptionConfigurationPage')
        ),
        path: exceptionConfigurationPath,
    },
    integrations: {
        component: lazy(() => import('Containers/Integrations/IntegrationsPage')),
        path: integrationsPath,
    },
    'listening-endpoints': {
        component: lazy(() => import('Containers/Audit/ListeningEndpoints/ListeningEndpointsPage')),
        path: listeningEndpointsBasePath,
    },
    'network-graph': {
        component: lazy(() => import('Containers/NetworkGraph/NetworkGraphPage')),
        path: networkPath,
    },
    'policy-management': {
        component: lazy(() => import('Containers/PolicyManagement/PolicyManagementPage')),
        path: policyManagementBasePath,
    },
    'risk/secrets': {
        component: lazy(() => import('Containers/Risk/Secrets/SecretsPage')),
        path: riskSecretsBasePath,
    },
    'risk/workloads': {
        component: lazy(() => import('Containers/Risk/RiskRoutes')),
        path: riskWorkloadsBasePath,
    },
    risk: {
        component: RiskRedirect,
        path: riskBasePath,
    },
    search: {
        component: lazy(() => import('Containers/Search/SearchPage')),
        path: searchPath,
    },
    'system-health': {
        component: lazy(() => import('Containers/SystemHealth/SystemHealthPage')),
        path: systemHealthPath,
    },
    systemconfig: {
        component: lazy(() => import('Containers/SystemConfig/SystemConfigPage')),
        path: systemConfigPath,
    },
    user: {
        component: lazy(() => import('Containers/User/UserPage')),
        path: userBasePath,
    },
    violations: {
        component: lazy(() => import('Containers/Violations/ViolationsPage')),
        path: violationsBasePath,
    },
    'vulnerabilities/exception-management': {
        component: lazy(
            () => import('Containers/Vulnerabilities/ExceptionManagement/ExceptionManagementPage')
        ),
        path: exceptionManagementPath,
    },
    'vulnerabilities/node-cves': {
        component: lazy(() => import('Containers/Vulnerabilities/NodeCves/NodeCvesPage')),
        path: vulnerabilitiesNodeCvesPath,
    },
    'vulnerabilities/user-workloads': {
        component: WorkloadCvesPage,
        componentKey: 'vulnerabilities/user-workloads',
        props: { view: 'user-workloads' },
        path: vulnerabilitiesUserWorkloadsPath,
    },
    'vulnerabilities/virtual-machine-cves': {
        component: lazy(
            () => import('Containers/Vulnerabilities/VirtualMachineCves/VirtualMachineCvesPage')
        ),
        path: vulnerabilitiesVirtualMachineCvesPath,
    },
    // Note: currently 'platform' is an implementation of the user-workloads view and
    // it is expected that this will change in the future as these views diverge
    'vulnerabilities/platform': {
        component: WorkloadCvesPage,
        componentKey: 'vulnerabilities/platform',
        props: { view: 'platform' },
        path: vulnerabilitiesPlatformPath,
    },
    'vulnerabilities/all-images': {
        component: WorkloadCvesPage,
        componentKey: 'vulnerabilities/all-images',
        props: { view: 'all-images' },
        path: vulnerabilitiesAllImagesPath,
    },
    'base-images': {
        component: lazy(() => import('Containers/BaseImages/BaseImagesPage')),
        path: baseImagesPath,
    },
    'vulnerabilities/inactive-images': {
        component: WorkloadCvesPage,
        componentKey: 'vulnerabilities/inactive-images',
        props: { view: 'inactive-images' },
        path: vulnerabilitiesInactiveImagesPath,
    },
    'vulnerabilities/images-without-cves': {
        component: WorkloadCvesPage,
        componentKey: 'vulnerabilities/images-without-cves',
        props: { view: 'images-without-cves' },
        path: vulnerabilitiesImagesWithoutCvesPath,
    },
    'vulnerabilities/reports': {
        component: lazy(
            () => import('Containers/Vulnerabilities/Reports/VulnerabilityReportsPage')
        ),
        path: vulnerabilityReportsPath,
    },
    'vulnerabilities/reports/images': {
        component: lazy(
            () => import('Containers/Vulnerabilities/VulnerablityReporting/VulnReportingPage')
        ),
        path: vulnerabilityImageReportsPath,
    },
    'vulnerabilities/reports/nodes': {
        component: lazy(
            () =>
                import('Containers/Vulnerabilities/Reports/NodeVulnerabilityReports/NodeVulnerabilityReportsPage')
        ),
        path: vulnerabilityNodeReportsPath,
    },
    'vulnerability-management': {
        component: lazy(() => import('Containers/VulnMgmt/WorkflowLayout')),
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

function RiskRedirect() {
    const location = useLocation();
    const newPath = location.pathname.replace(riskBasePath, riskWorkloadsBasePath);
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
        <div id="BodyRoutes">
            <Routes>
                <Route path="/" element={<Navigate to={dashboardPath} replace />} />
                <Route path={mainPath} element={<Navigate to={dashboardPath} replace />} />
                {/* Make sure the following Redirect element works after react-router-dom upgrade */}
                <Route path={deprecatedPoliciesPath} element={<DeprecatedPoliciesRedirect />} />
                <Route
                    // all prior workload-cves routes must redirect to the new path.
                    path={`${vulnerabilitiesWorkloadCvesPath}/*`}
                    // Since all subpaths and query parameters must be retained, we need to do
                    // a search and replace of the subpath we are redirecting, which is accomplished
                    // by using the WorkloadCvesRedirect component.
                    element={<WorkloadCvesRedirect />}
                />
                {Object.keys(routeComponentMap)
                    .filter((routeKey) => isRouteEnabled(routePredicates, routeKey as RouteKey))
                    .map((routeKey) => {
                        const {
                            component: Component,
                            componentKey,
                            path,
                            props,
                        } = routeComponentMap[routeKey];
                        return (
                            <Route
                                key={routeKey}
                                path={`${path}/*`}
                                element={
                                    <Suspense
                                        fallback={
                                            <Bullseye>
                                                <Spinner />
                                            </Bullseye>
                                        }
                                    >
                                        <Component key={componentKey} {...props} />
                                    </Suspense>
                                }
                            />
                        );
                    })}
                <Route path="*" element={<NotFoundPage />} />
            </Routes>
            {hasWriteAccessForInviting && showInviteModal && <InviteUsersModal />}
        </div>
    );
}

export default Body;
