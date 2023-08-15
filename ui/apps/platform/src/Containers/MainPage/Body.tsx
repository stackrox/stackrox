import React, { ElementType, ReactElement, useEffect } from 'react';
import { Redirect, Route, Switch, useLocation } from 'react-router-dom';
import { PageSection } from '@patternfly/react-core';

// Import path variables in alphabetical order to minimize merge conflicts when multiple people add routes.
import {
    isRouteEnabled, // predicate function
    accessControlPath,
    apidocsPath,
    clustersDelegatedScanningPath,
    clustersPathWithParam,
    collectionsPath,
    complianceEnhancedBasePath,
    compliancePath,
    configManagementPath,
    dashboardPath,
    deprecatedPoliciesPath,
    integrationsPath,
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
} from 'routePaths';
import { useTheme } from 'Containers/ThemeProvider';

import PageNotFound from 'Components/PageNotFound';
import PageTitle from 'Components/PageTitle';
import ErrorBoundary from 'Containers/ErrorBoundary';
import { HasReadAccess } from 'hooks/usePermissions';
import { IsFeatureFlagEnabled } from 'hooks/useFeatureFlags';
import useAnalytics from 'hooks/useAnalytics';

import asyncComponent from './AsyncComponent';

function NotFoundPage(): ReactElement {
    return (
        <PageSection variant="light">
            <PageTitle title="Not Found" />
            <PageNotFound />
        </PageSection>
    );
}

// routeComponentMap corresponds to routeDescriptionMap in src/routePaths.ts file.
// Add path keys in alphabetical order to minimize merge conflicts when multiple people add routes.
const routeComponentMap: Record<string, ElementType> = {
    [accessControlPath]: asyncComponent(() => import('Containers/AccessControl/AccessControl')),
    [apidocsPath]: asyncComponent(() => import('Containers/Docs/ApiPage')),
    [clustersDelegatedScanningPath]: asyncComponent(
        () => import('Containers/Clusters/DelegateScanning/DelegateScanningPage')
    ),
    [clustersPathWithParam]: asyncComponent(() => import('Containers/Clusters/ClustersPage')),
    [collectionsPath]: asyncComponent(() => import('Containers/Collections/CollectionsPage')),
    [compliancePath]: asyncComponent(() => import('Containers/Compliance/Page')),
    [complianceEnhancedBasePath]: asyncComponent(
        () => import('Containers/ComplianceEnhanced/Dashboard/ComplianceDashboardPage')
    ),
    [configManagementPath]: asyncComponent(() => import('Containers/ConfigManagement/Page')),
    [dashboardPath]: asyncComponent(() => import('Containers/Dashboard/DashboardPage')),
    [integrationsPath]: asyncComponent(() => import('Containers/Integrations/IntegrationsPage')),
    [listeningEndpointsBasePath]: asyncComponent(
        () => import('Containers/Audit/ListeningEndpoints/ListeningEndpointsPage')
    ),
    [networkPath]: asyncComponent(() => import('Containers/NetworkGraph/NetworkGraphPage')),
    [policyManagementBasePath]: asyncComponent(
        () => import('Containers/PolicyManagement/PolicyManagementPage')
    ),
    [riskPath]: asyncComponent(() => import('Containers/Risk/RiskPage')),
    [searchPath]: asyncComponent(() => import('Containers/Search/SearchPage')),
    [systemConfigPath]: asyncComponent(() => import('Containers/SystemConfig/SystemConfigPage')),
    [systemHealthPath]: asyncComponent(() => import('Containers/SystemHealth/DashboardPage')),
    [userBasePath]: asyncComponent(() => import('Containers/User/UserPage')),
    [violationsPath]: asyncComponent(() => import('Containers/Violations/ViolationsPage')),
    // Reporting and Risk Acceptance must precede generic Vulnerability Management.
    [vulnManagementReportsPath]: asyncComponent(
        () => import('Containers/VulnMgmt/Reports/VulnMgmtReports')
    ),
    [vulnManagementRiskAcceptancePath]: asyncComponent(
        () => import('Containers/VulnMgmt/RiskAcceptance/RiskAcceptancePage')
    ),
    [vulnManagementPath]: asyncComponent(() => import('Containers/VulnMgmt/WorkflowLayout')),
    [vulnerabilitiesWorkloadCvesPath]: asyncComponent(
        () => import('Containers/Vulnerabilities/WorkloadCves/WorkloadCvesPage')
    ),
    [vulnerabilityReportsPath]: asyncComponent(
        () => import('Containers/Vulnerabilities/VulnerablityReporting/VulnReportingPage')
    ),
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
                    {Object.entries(routeComponentMap)
                        .filter(([path]) => isRouteEnabled(routePredicates, path))
                        .map(([path, component]) => {
                            return <Route key={path} path={path} component={component} />;
                        })}
                    <Route component={NotFoundPage} />
                </Switch>
            </ErrorBoundary>
        </div>
    );
}

export default Body;
