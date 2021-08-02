import React, { ReactElement } from 'react';
import { Redirect, Switch } from 'react-router-dom';

import {
    mainPath,
    dashboardPath,
    networkPath,
    violationsPath,
    violationsPFBasePath,
    compliancePath,
    clustersPathWithParam,
    integrationsPath,
    policiesPath,
    riskPath,
    apidocsPath,
    accessControlPath,
    accessControlPathV2,
    userBasePath,
    systemConfigPath,
    systemHealthPath,
    vulnManagementPath,
    configManagementPath,
} from 'routePaths';
import { knownBackendFlags } from 'utils/featureFlags';
import useFeatureFlagEnabled from 'hooks/useFeatureFlagEnabled';
import { useTheme } from 'Containers/ThemeProvider';

import asyncComponent from 'Components/AsyncComponent';
import ProtectedRoute from 'Components/ProtectedRoute';
import ErrorBoundary from 'Containers/ErrorBoundary';

const AsyncApiDocsPage = asyncComponent(() => import('Containers/Docs/ApiPage'));
const AsyncDashboardPage = asyncComponent(() => import('Containers/Dashboard/DashboardPage'));
const AsyncNetworkPage = asyncComponent(() => import('Containers/Network/Page'));
const AsyncClustersPage = asyncComponent(() => import('Containers/Clusters/ClustersPage'));
const AsyncIntegrationsPage = asyncComponent(
    () => import('Containers/Integrations/IntegrationsPage')
);
const AsyncViolationsPage = asyncComponent(() => import('Containers/Violations/ViolationsPage'));
const AsyncViolationsPFPage = asyncComponent(
    () => import('Containers/Violations/PatternFly/ViolationsPage')
);

const AsyncPoliciesPage = asyncComponent(() => import('Containers/Policies/Page'));
const AsyncCompliancePage = asyncComponent(() => import('Containers/Compliance/Page'));
const AsyncRiskPage = asyncComponent(() => import('Containers/Risk/RiskPage'));
const AsyncAccessControlPage = asyncComponent(
    () => import('Containers/AccessControl/classic/Page')
);
const AsyncAccessControlPageV2 = asyncComponent(
    () => import('Containers/AccessControl/AccessControl')
);
const AsyncUserPage = asyncComponent(() => import('Containers/User/UserPage'));
const AsyncSystemConfigPage = asyncComponent(() => import('Containers/SystemConfig/Page'));
const AsyncConfigManagementPage = asyncComponent(() => import('Containers/ConfigManagement/Page'));
const AsyncVulnMgmtPage = asyncComponent(() => import('Containers/Workflow/WorkflowLayout'));
const AsyncSystemHealthPage = asyncComponent(() => import('Containers/SystemHealth/DashboardPage'));

function Body(): ReactElement {
    const { isDarkMode } = useTheme();
    const isScopedAccessControlEnabled = useFeatureFlagEnabled(
        knownBackendFlags.ROX_SCOPED_ACCESS_CONTROL
    );

    return (
        <div
            className={`flex flex-col h-full w-full relative overflow-auto ${
                isDarkMode ? 'bg-base-0' : 'bg-base-100'
            }`}
        >
            <ErrorBoundary>
                <Switch>
                    <ProtectedRoute path={dashboardPath} component={AsyncDashboardPage} />
                    <ProtectedRoute path={networkPath} component={AsyncNetworkPage} />
                    <ProtectedRoute
                        path={violationsPFBasePath}
                        component={AsyncViolationsPFPage}
                        devOnly
                    />
                    <ProtectedRoute path={violationsPath} component={AsyncViolationsPage} />
                    <ProtectedRoute path={compliancePath} component={AsyncCompliancePage} />
                    <ProtectedRoute path={integrationsPath} component={AsyncIntegrationsPage} />
                    <ProtectedRoute path={policiesPath} component={AsyncPoliciesPage} />
                    <ProtectedRoute path={riskPath} component={AsyncRiskPage} />
                    <ProtectedRoute path={accessControlPath} component={AsyncAccessControlPage} />
                    <ProtectedRoute
                        path={accessControlPathV2}
                        component={AsyncAccessControlPageV2}
                        featureFlagEnabled={isScopedAccessControlEnabled}
                    />
                    <ProtectedRoute path={apidocsPath} component={AsyncApiDocsPage} />
                    <ProtectedRoute path={userBasePath} component={AsyncUserPage} />
                    <ProtectedRoute path={systemConfigPath} component={AsyncSystemConfigPage} />
                    <ProtectedRoute path={vulnManagementPath} component={AsyncVulnMgmtPage} />
                    <ProtectedRoute
                        path={configManagementPath}
                        component={AsyncConfigManagementPage}
                    />
                    <ProtectedRoute path={clustersPathWithParam} component={AsyncClustersPage} />
                    <ProtectedRoute path={systemHealthPath} component={AsyncSystemHealthPage} />
                    <Redirect from={mainPath} to={dashboardPath} />
                </Switch>
            </ErrorBoundary>
        </div>
    );
}

export default Body;
