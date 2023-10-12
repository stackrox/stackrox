import React, { ElementType, ReactElement, useEffect } from 'react';
import { Redirect, Route, Switch, useLocation } from 'react-router-dom';
import { PageSection } from '@patternfly/react-core';

// Import path variables in alphabetical order to minimize merge conflicts when multiple people add routes.
import {
    RouteKey,
    accessControlPath,
    administrationEventsPathWithParam,
    apidocsPath,
    clustersDelegatedScanningPath,
    clustersInitBundlesPathWithParam,
    clustersPathWithParam,
    collectionsPath,
    complianceEnhancedBasePath,
    compliancePath,
    configManagementPath,
    dashboardPath,
    exceptionConfigurationPath,
    deprecatedPoliciesPath,
    integrationsPath,
    isRouteEnabled, // predicate function
    listeningEndpointsBasePath,
    mainPath,
    networkPath,
    policiesPath,
    policyManagementBasePath,
    riskPath,
    searchPath,
    systemConfigPath,
    systemHealthPath,
    userBasePath,
    violationsPath,
    vulnManagementPath,
    vulnManagementReportsPath,
    vulnManagementRiskAcceptancePath,
    vulnerabilitiesWorkloadCvesPath,
    vulnerabilityReportsPath,
    complianceEnhancedScanConfigsPath,
} from 'routePaths';
import { useTheme } from 'Containers/ThemeProvider';

import PageNotFound from 'Components/PageNotFound';
import PageTitle from 'Components/PageTitle';
import ErrorBoundary from 'Components/PatternFly/ErrorBoundary/ErrorBoundary';
import { HasReadAccess } from 'hooks/usePermissions';
import { IsFeatureFlagEnabled } from 'hooks/useFeatureFlags';
import useAnalytics from 'hooks/useAnalytics';

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

type RouteComponent = {
    component: ElementType;
    path: string;
};

// routeComponentMap corresponds to routeRequirementsMap in src/routePaths.ts file.
// Add route keys in alphabetical order to minimize merge conflicts when multiple people add routes.
const routeComponentMap: Record<RouteKey, RouteComponent> = {
    'access-control': {
        component: asyncComponent(() => import('Containers/AccessControl/AccessControl')),
        path: accessControlPath,
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
    // Delegated image scanning must precede generic Clusters.
    'clusters/delegated-image-scanning': {
        component: asyncComponent(
            () => import('Containers/Clusters/DelegateScanning/DelegateScanningPage')
        ),
        path: clustersDelegatedScanningPath,
    },
    // Cluster init bundles must precede generic Clusters.
    'clusters/init-bundles': {
        component: asyncComponent(() => import('Containers/Clusters/InitBundles/InitBundlesRoute')),
        path: clustersInitBundlesPathWithParam,
    },
    clusters: {
        component: asyncComponent(() => import('Containers/Clusters/ClustersPage')),
        path: clustersPathWithParam,
    },
    collections: {
        component: asyncComponent(() => import('Containers/Collections/CollectionsPage')),
        path: collectionsPath,
    },
    compliance: {
        component: asyncComponent(() => import('Containers/Compliance/Page')),
        path: compliancePath,
    },
    'compliance-enhanced/scan-configs': {
        component: asyncComponent(
            () => import('Containers/ComplianceEnhanced/Scheduling/SchedulingPage')
        ),
        path: complianceEnhancedScanConfigsPath,
    },
    'compliance-enhanced': {
        component: asyncComponent(
            () => import('Containers/ComplianceEnhanced/ComplianceEnhancedPage')
        ),
        path: complianceEnhancedBasePath,
    },
    configmanagement: {
        component: asyncComponent(() => import('Containers/ConfigManagement/Page')),
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
        path: violationsPath,
    },
    'vulnerabilities/reports': {
        component: asyncComponent(
            () => import('Containers/Vulnerabilities/VulnerablityReporting/VulnReportingPage')
        ),
        path: vulnerabilityReportsPath,
    },
    // Reports must precede generic Vulnerability Management.
    'vulnerability-management/reports': {
        component: asyncComponent(() => import('Containers/VulnMgmt/Reports/VulnMgmtReports')),
        path: vulnManagementReportsPath,
    },
    // Risk Acceptance must precede generic Vulnerability Management.
    'vulnerability-management/risk-acceptance': {
        component: asyncComponent(
            () => import('Containers/VulnMgmt/RiskAcceptance/RiskAcceptancePage')
        ),
        path: vulnManagementRiskAcceptancePath,
    },
    'vulnerability-management': {
        component: asyncComponent(() => import('Containers/VulnMgmt/WorkflowLayout')),
        path: vulnManagementPath,
    },
    'workload-cves': {
        component: asyncComponent(
            () => import('Containers/Vulnerabilities/WorkloadCves/WorkloadCvesPage')
        ),
        path: vulnerabilitiesWorkloadCvesPath,
    },
};

type BodyProps = {
    hasReadAccess: HasReadAccess;
    isFeatureFlagEnabled: IsFeatureFlagEnabled;
};

function Body({ hasReadAccess, isFeatureFlagEnabled }: BodyProps): ReactElement {
    const location = useLocation();
    const { analyticsPageVisit } = useAnalytics();
    useEffect(() => {
        analyticsPageVisit('Page Viewed', '', { path: location.pathname });
    }, [location, analyticsPageVisit]);

    const { isDarkMode } = useTheme();

    const routePredicates = { hasReadAccess, isFeatureFlagEnabled };

    return (
        <div
            className={`flex flex-col h-full w-full relative overflow-auto ${
                isDarkMode ? 'bg-base-0' : 'bg-base-100'
            }`}
        >
            <ErrorBoundary>
                <Switch>
                    <Route path="/" exact render={() => <Redirect to={dashboardPath} />} />
                    <Route path={mainPath} exact render={() => <Redirect to={dashboardPath} />} />
                    {/* Make sure the following Redirect element works after react-router-dom upgrade */}
                    <Redirect exact from={deprecatedPoliciesPath} to={policiesPath} />
                    {Object.keys(routeComponentMap)
                        .filter((routeKey) => isRouteEnabled(routePredicates, routeKey as RouteKey))
                        .map((routeKey) => {
                            const { component, path } = routeComponentMap[routeKey];
                            return <Route key={routeKey} path={path} component={component} />;
                        })}
                    <Route component={NotFoundPage} />
                </Switch>
                <InviteUsersModal />
            </ErrorBoundary>
        </div>
    );
}

export default Body;
