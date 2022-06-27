import React, { ReactElement } from 'react';
import { Redirect, Route, Switch } from 'react-router-dom';
import { PageSection } from '@patternfly/react-core';

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
    policyManagementBasePath,
    deprecatedPoliciesPath,
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
import PageNotFound from 'Components/PageNotFound';
import PageTitle from 'Components/PageTitle';
import ErrorBoundary from 'Containers/ErrorBoundary';
import { HasReadAccess } from 'hooks/usePermissions';
import { IsFeatureFlagEnabled } from 'hooks/useFeatureFlags';

function NotFoundPage(): ReactElement {
    return (
        <PageSection variant="light">
            <PageTitle title="Not Found" />
            <PageNotFound />
        </PageSection>
    );
}

const AsyncApiDocsPage = asyncComponent(() => import('Containers/Docs/ApiPage'));
const AsyncDashboardPage = asyncComponent(() => import('Containers/Dashboard/DashboardPage'));
// TODO Rename this and replace AsyncDashboardPage once Sec Metrics Phase One is complete
// Jira: https://issues.redhat.com/browse/ROX-10650
const AsyncDashboardPagePF = asyncComponent(
    () => import('Containers/Dashboard/PatternFly/DashboardPage')
);
const AsyncNetworkPage = asyncComponent(() => import('Containers/Network/Page'));
const AsyncClustersPage = asyncComponent(() => import('Containers/Clusters/ClustersPage'));
const AsyncPFClustersPage = asyncComponent(() => import('Containers/Clusters/PF/ClustersPage'));
const AsyncIntegrationsPage = asyncComponent(
    () => import('Containers/Integrations/IntegrationsPage')
);
const AsyncViolationsPage = asyncComponent(() => import('Containers/Violations/ViolationsPage'));

const AsyncPolicyManagementPage = asyncComponent(
    () => import('Containers/Policies/PolicyManagementPage')
);

const AsyncCompliancePage = asyncComponent(() => import('Containers/Compliance/Page'));
const AsyncRiskPage = asyncComponent(() => import('Containers/Risk/RiskPage'));
const AsyncAccessControlPageV2 = asyncComponent(
    () => import('Containers/AccessControl/AccessControl')
);
const AsyncUserPage = asyncComponent(() => import('Containers/User/UserPage'));
const AsyncSystemConfigPage = asyncComponent(
    () => import('Containers/SystemConfig/SystemConfigPage')
);
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

type BodyProps = {
    hasReadAccess: HasReadAccess;
    isFeatureFlagEnabled: IsFeatureFlagEnabled;
};

function Body({ hasReadAccess, isFeatureFlagEnabled }: BodyProps): ReactElement {
    const { isDarkMode } = useTheme();

    const isSystemHealthPatternFlyEnabled = isFeatureFlagEnabled('ROX_SYSTEM_HEALTH_PF');
    const isDashboardPatternFlyEnabled = isFeatureFlagEnabled('ROX_SECURITY_METRICS_PHASE_ONE');

    const hasVulnerabilityReportsPermission = hasReadAccess('VulnerabilityReports');

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
                    <Route
                        path={dashboardPath}
                        component={
                            isDashboardPatternFlyEnabled ? AsyncDashboardPagePF : AsyncDashboardPage
                        }
                    />
                    <Route path={networkPath} component={AsyncNetworkPage} />
                    <Route path={violationsPath} component={AsyncViolationsPage} />
                    <Route path={compliancePath} component={AsyncCompliancePage} />
                    <Route path={integrationsPath} component={AsyncIntegrationsPage} />
                    <Route path={policyManagementBasePath} component={AsyncPolicyManagementPage} />
                    <Redirect exact from={deprecatedPoliciesPath} to={policiesPath} />
                    <Route path={riskPath} component={AsyncRiskPage} />
                    <Route path={accessControlPathV2} component={AsyncAccessControlPageV2} />
                    <Route path={apidocsPath} component={AsyncApiDocsPage} />
                    <Route path={userBasePath} component={AsyncUserPage} />
                    <Route path={systemConfigPath} component={AsyncSystemConfigPage} />
                    {hasVulnerabilityReportsPermission && (
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
                    <Route component={NotFoundPage} />
                </Switch>
            </ErrorBoundary>
        </div>
    );
}

export default Body;
