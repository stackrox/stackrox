import React, { ReactElement, useEffect } from 'react';
import { Redirect, Route, Switch, useLocation } from 'react-router-dom';
import { PageSection } from '@patternfly/react-core';

import {
    mainPath,
    dashboardPath,
    networkPath,
    violationsPath,
    compliancePath,
    complianceEnhancedBasePath,
    clustersListPath,
    clustersDelegateScanningPath,
    clustersPathWithParam,
    integrationsPath,
    policiesPath,
    policyManagementBasePath,
    deprecatedPoliciesPath,
    riskPath,
    searchPath,
    apidocsPath,
    accessControlPath,
    userBasePath,
    systemConfigPath,
    systemHealthPath,
    vulnManagementPath,
    vulnManagementReportsPath,
    configManagementPath,
    vulnManagementRiskAcceptancePath,
    collectionsPath,
    vulnerabilitiesWorkloadCvesPath,
    vulnerabilityReportsPath,
    listeningEndpointsBasePath,
} from 'routePaths';
import { useTheme } from 'Containers/ThemeProvider';

import asyncComponent from 'Components/AsyncComponent';
import PageNotFound from 'Components/PageNotFound';
import PageTitle from 'Components/PageTitle';
import ErrorBoundary from 'Containers/ErrorBoundary';
import { HasReadAccess } from 'hooks/usePermissions';
import { IsFeatureFlagEnabled } from 'hooks/useFeatureFlags';
import useAnalytics from 'hooks/useAnalytics';

function NotFoundPage(): ReactElement {
    return (
        <PageSection variant="light">
            <PageTitle title="Not Found" />
            <PageNotFound />
        </PageSection>
    );
}

const AsyncSearchPage = asyncComponent(() => import('Containers/Search/SearchPage'));
const AsyncApiDocsPage = asyncComponent(() => import('Containers/Docs/ApiPage'));
const AsyncDashboardPage = asyncComponent(() => import('Containers/Dashboard/DashboardPage'));
const AsyncNetworkGraphPage = asyncComponent(
    () => import('Containers/NetworkGraph/NetworkGraphPage')
);
const AsyncDelegateScanningPage = asyncComponent(
    () => import('Containers/Clusters/DelegateScanning/DelegateScanningPage')
);
const AsyncClustersPage = asyncComponent(() => import('Containers/Clusters/ClustersPage'));
const AsyncPFClustersPage = asyncComponent(() => import('Containers/Clusters/PF/ClustersPage'));
const AsyncIntegrationsPage = asyncComponent(
    () => import('Containers/Integrations/IntegrationsPage')
);
const AsyncViolationsPage = asyncComponent(() => import('Containers/Violations/ViolationsPage'));

const AsyncPolicyManagementPage = asyncComponent(
    () => import('Containers/PolicyManagement/PolicyManagementPage')
);

const AsyncCollectionsPage = asyncComponent(() => import('Containers/Collections/CollectionsPage'));

const AsyncCompliancePage = asyncComponent(() => import('Containers/Compliance/Page'));
const AsyncComplianceEnhancedPage = asyncComponent(
    () => import('Containers/ComplianceEnhanced/CompliancePage')
);
const AsyncRiskPage = asyncComponent(() => import('Containers/Risk/RiskPage'));
const AsyncAccessControlPageV2 = asyncComponent(
    () => import('Containers/AccessControl/AccessControl')
);
const AsyncUserPage = asyncComponent(() => import('Containers/User/UserPage'));
const AsyncSystemConfigPage = asyncComponent(
    () => import('Containers/SystemConfig/SystemConfigPage')
);
const AsyncConfigManagementPage = asyncComponent(() => import('Containers/ConfigManagement/Page'));
const AsyncWorkloadCvesPage = asyncComponent(
    () => import('Containers/Vulnerabilities/WorkloadCves/WorkloadCvesPage')
);
const AsyncVulnerabilityReportingPage = asyncComponent(
    () => import('Containers/Vulnerabilities/VulnerablityReporting/VulnReportingPage')
);
const AsyncVulnMgmtReports = asyncComponent(
    () => import('Containers/VulnMgmt/Reports/VulnMgmtReports')
);
const AsyncVulnMgmtRiskAcceptancePage = asyncComponent(
    () => import('Containers/VulnMgmt/RiskAcceptance/RiskAcceptancePage')
);
const AsyncVulnMgmtPage = asyncComponent(() => import('Containers/VulnMgmt/WorkflowLayout'));
const AsyncSystemHealthPage = asyncComponent(() => import('Containers/SystemHealth/DashboardPage'));

const AsyncListeningEndpointsPage = asyncComponent(
    () => import('Containers/Audit/ListeningEndpoints/ListeningEndpointsPage')
);

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

    const isVulnMgmtWorkloadCvesEnabled = isFeatureFlagEnabled('ROX_VULN_MGMT_WORKLOAD_CVES');
    const isVulnerabilityReportingEnhancementsEnabled = isFeatureFlagEnabled(
        'ROX_VULN_MGMT_REPORTING_ENHANCEMENTS'
    );

    const hasVulnerabilityReportsPermission = hasReadAccess('WorkflowAdministration');
    const hasCollectionsPermission = hasReadAccess('WorkflowAdministration');

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
                    <Route path={dashboardPath} component={AsyncDashboardPage} />
                    <Route path={networkPath} component={AsyncNetworkGraphPage} />
                    <Route path={violationsPath} component={AsyncViolationsPage} />
                    <Route path={compliancePath} component={AsyncCompliancePage} />
                    <Route
                        path={complianceEnhancedBasePath}
                        component={AsyncComplianceEnhancedPage}
                    />
                    <Route path={integrationsPath} component={AsyncIntegrationsPage} />
                    <Route path={policyManagementBasePath} component={AsyncPolicyManagementPage} />
                    {/* Make sure the following Redirect element works after react-router-dom upgrade */}
                    <Redirect exact from={deprecatedPoliciesPath} to={policiesPath} />
                    {hasCollectionsPermission && (
                        <Route path={collectionsPath} component={AsyncCollectionsPage} />
                    )}
                    <Route path={riskPath} component={AsyncRiskPage} />
                    <Route path={accessControlPath} component={AsyncAccessControlPageV2} />
                    <Route path={searchPath} component={AsyncSearchPage} />
                    <Route path={apidocsPath} component={AsyncApiDocsPage} />
                    <Route path={userBasePath} component={AsyncUserPage} />
                    <Route path={systemConfigPath} component={AsyncSystemConfigPage} />
                    {isVulnMgmtWorkloadCvesEnabled && (
                        <Route
                            path={vulnerabilitiesWorkloadCvesPath}
                            component={AsyncWorkloadCvesPage}
                        />
                    )}
                    {hasVulnerabilityReportsPermission &&
                        isVulnerabilityReportingEnhancementsEnabled && (
                            <Route
                                path={vulnerabilityReportsPath}
                                component={AsyncVulnerabilityReportingPage}
                            />
                        )}
                    {hasVulnerabilityReportsPermission && (
                        <Route path={vulnManagementReportsPath} component={AsyncVulnMgmtReports} />
                    )}
                    <Route
                        path={vulnManagementRiskAcceptancePath}
                        component={AsyncVulnMgmtRiskAcceptancePage}
                    />
                    <Route path={vulnManagementPath} component={AsyncVulnMgmtPage} />
                    <Route path={configManagementPath} component={AsyncConfigManagementPage} />
                    <Route
                        path={clustersDelegateScanningPath}
                        component={AsyncDelegateScanningPage}
                    />
                    <Route path={clustersPathWithParam} component={AsyncClustersPage} />
                    {process.env.NODE_ENV === 'development' && (
                        <Route path={clustersListPath} component={AsyncPFClustersPage} />
                    )}
                    <Route path={systemHealthPath} component={AsyncSystemHealthPage} />
                    {/* 
                    TODO - Add any necessary permissions to the following route. The user will need read access to
                          'Cluster' and 'Deployment' at the very least.
                     */}
                    <Route
                        path={listeningEndpointsBasePath}
                        component={AsyncListeningEndpointsPage}
                    />
                    <Route component={NotFoundPage} />
                </Switch>
            </ErrorBoundary>
        </div>
    );
}

export default Body;
