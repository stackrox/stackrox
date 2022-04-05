import React, { ReactElement } from 'react';
import { Redirect, Route, Switch } from 'react-router-dom';

import {
    mainPath,
    dashboardPath,
    networkPath,
    violationsPath,
    compliancePath,
    clustersPathWithParam,
    clustersListPath,
    integrationsPath,
    policiesPath,
    riskPath,
    apidocsPath,
    accessControlPathV2,
    userBasePath,
    systemConfigPath,
    systemHealthPath,
    systemHealthPathPF,
    vulnManagementPath,
    vulnManagementReportsPath,
    configManagementPath,
    vulnManagementRiskAcceptancePath,
} from 'routePaths';
import { useTheme } from 'Containers/ThemeProvider';

import asyncComponent from 'Components/AsyncComponent';
import ErrorBoundary from 'Containers/ErrorBoundary';
import useFeatureFlags from 'hooks/useFeatureFlags';
import usePermissions from 'hooks/usePermissions';
import { knownBackendFlags } from 'utils/featureFlags';

const AsyncApiDocsPage = asyncComponent(() => import('Containers/Docs/ApiPage'));
const AsyncDashboardPage = asyncComponent(() => import('Containers/Dashboard/DashboardPage'));
const AsyncNetworkPage = asyncComponent(() => import('Containers/Network/Page'));
const AsyncClustersPage = asyncComponent(() => import('Containers/Clusters/ClustersPage'));
const AsyncPFClustersPage = asyncComponent(() => import('Containers/Clusters/PF/ClustersPage'));
const AsyncIntegrationsPage = asyncComponent(
    () => import('Containers/Integrations/IntegrationsPage')
);
const AsyncViolationsPage = asyncComponent(() => import('Containers/Violations/ViolationsPage'));

// TODO: rename this to AsyncPoliciesPage after we remove the old deprecated policies code
// Jira issue to track: https://issues.redhat.com/browse/ROX-9450
const AsyncPoliciesPagePatternFly = asyncComponent(
    () => import('Containers/Policies/PatternFly/PoliciesPage')
);
const AsyncCompliancePage = asyncComponent(() => import('Containers/Compliance/Page'));
const AsyncRiskPage = asyncComponent(() => import('Containers/Risk/RiskPage'));
const AsyncAccessControlPageV2 = asyncComponent(
    () => import('Containers/AccessControl/AccessControl')
);
const AsyncUserPage = asyncComponent(() => import('Containers/User/UserPage'));
const AsyncSystemConfigPage = asyncComponent(() => import('Containers/SystemConfig/Page'));
const AsyncConfigManagementPage = asyncComponent(() => import('Containers/ConfigManagement/Page'));
const AsyncVulnMgmtReports = asyncComponent(
    () => import('Containers/VulnMgmt/Reports/VulnMgmtReports')
);
const AsyncVulnMgmtRiskAcceptancePage = asyncComponent(
    () => import('Containers/VulnMgmt/RiskAcceptance/RiskAcceptancePage')
);
const AsyncVulnMgmtPage = asyncComponent(() => import('Containers/Workflow/WorkflowLayout'));
const AsyncSystemHealthPage = asyncComponent(() => import('Containers/SystemHealth/DashboardPage'));
const AsyncSystemHealthPagePF = asyncComponent(
    () => import('Containers/SystemHealth/PatternFly/SystemHealthDashboard')
);

function Body(): ReactElement {
    const { isDarkMode } = useTheme();

    // MainPage renders Body only when feature flags and permissions are available.

    const { isFeatureFlagEnabled } = useFeatureFlags();
    const isSystemHealthPatternFlyEnabled = isFeatureFlagEnabled(
        knownBackendFlags.ROX_SYSTEM_HEALTH_PF
    );
    const isVulnReportingEnabled = isFeatureFlagEnabled(knownBackendFlags.ROX_VULN_REPORTING);

    const { hasReadAccess } = usePermissions();
    const hasVulnerabilityReportsPermission = hasReadAccess('VulnerabilityReports');

    return (
        <div
            className={`flex flex-col h-full w-full relative overflow-auto ${
                isDarkMode ? 'bg-base-0' : 'bg-base-100'
            }`}
        >
            <ErrorBoundary>
                <Switch>
                    <Route path={dashboardPath} component={AsyncDashboardPage} />
                    <Route path={networkPath} component={AsyncNetworkPage} />
                    <Route path={violationsPath} component={AsyncViolationsPage} />
                    <Route path={compliancePath} component={AsyncCompliancePage} />
                    <Route path={integrationsPath} component={AsyncIntegrationsPage} />
                    <Route path={policiesPath} component={AsyncPoliciesPagePatternFly} />
                    <Route path={riskPath} component={AsyncRiskPage} />
                    <Route path={accessControlPathV2} component={AsyncAccessControlPageV2} />
                    <Route path={apidocsPath} component={AsyncApiDocsPage} />
                    <Route path={userBasePath} component={AsyncUserPage} />
                    <Route path={systemConfigPath} component={AsyncSystemConfigPage} />
                    {isVulnReportingEnabled && hasVulnerabilityReportsPermission && (
                        <Route path={vulnManagementReportsPath} component={AsyncVulnMgmtReports} />
                    )}
                    <Route
                        path={vulnManagementRiskAcceptancePath}
                        component={AsyncVulnMgmtRiskAcceptancePage}
                    />
                    <Route path={vulnManagementPath} component={AsyncVulnMgmtPage} />
                    <Route path={configManagementPath} component={AsyncConfigManagementPage} />
                    <Route path={clustersPathWithParam} component={AsyncClustersPage} />
                    {process.env.NODE_ENV === 'development' && (
                        <Route path={clustersListPath} component={AsyncPFClustersPage} />
                    )}
                    <Route path={systemHealthPath} component={AsyncSystemHealthPage} />
                    {isSystemHealthPatternFlyEnabled && (
                        <Route path={systemHealthPathPF} component={AsyncSystemHealthPagePF} />
                    )}
                    <Redirect from={mainPath} to={dashboardPath} />
                </Switch>
            </ErrorBoundary>
        </div>
    );
}

export default Body;
