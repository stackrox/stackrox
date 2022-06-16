import React, { ElementType, ReactElement } from 'react';
import { Redirect, Route, Switch } from 'react-router-dom';
import { PageSection } from '@patternfly/react-core';

import {
    IsRenderedRoutePath,
    mainPath,
    dashboardPath,
    dashboardPathPF,
    networkBasePath,
    networkPath,
    violationsBasePath,
    violationsPath,
    complianceBasePath,
    compliancePath,
    clustersBasePath,
    clustersPathWithParam,
    // clustersListPath,
    integrationsPath,
    policiesBasePath,
    policiesPath,
    deprecatedPoliciesPath,
    riskBasePath,
    riskPath,
    apidocsPath,
    accessControlBasePathV2,
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

import PageNotFound from 'Components/PageNotFound';
import PageTitle from 'Components/PageTitle';
import ErrorBoundary from 'Containers/ErrorBoundary';
import { IsFeatureFlagEnabled } from 'hooks/useFeatureFlags';
import { FeatureFlagEnvVar } from 'types/featureFlag';

import asyncComponent from './AsyncComponent';

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
// const AsyncPFClustersPage = asyncComponent(() => import('Containers/Clusters/PF/ClustersPage'));
const AsyncIntegrationsPage = asyncComponent(
    () => import('Containers/Integrations/IntegrationsPage')
);
const AsyncViolationsPage = asyncComponent(() => import('Containers/Violations/ViolationsPage'));

const AsyncPolicyManagementPage = asyncComponent(() => import('Containers/Policies/PoliciesPage'));

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

/*
 * basePath like violationsBasePath = '/main/violations' is key of routeDescriptorMap in routePaths.ts
 * propPath includes parameters like path="/main/violations/:alertId?"
 */
type BaseRouteComponent = {
    basePath: string;
    propPath?: string;
};

/*
 * Specify featureFlagDependency property of routeDescriptorMap in routePaths.ts
 * for a brand new route like /main/vulnerability-management/risk-acceptance
 * for a temporary route like /main/dashboard-pf for superseding conponent
 */
type IndependentRouteComponent = {
    component: ElementType;
} & BaseRouteComponent;

/*
 * At transition from development to release:
 * Delete temporary route like /main/dashboard-pf from routeDescriptorMap in routePaths.ts
 * Replace pair of independent route components for original and temporary route in routeComponents
 * with dependent route component with disabled superseded component and enabled superseding component.
 *
 * In case of emergency, patch release turns off feature flag which reverts to superseded component.
 * When feature flag is deleted, replace dependent route component with independent route component.
 */
type DependentRouteComponent = {
    featureFlagDependency: FeatureFlagEnvVar;
    componentDisabled: ElementType;
    componentEnabled: ElementType;
} & BaseRouteComponent;

type RouteComponent = IndependentRouteComponent | DependentRouteComponent;

const routeComponents: RouteComponent[] = [
    // Sidebar Unexpandable1
    {
        basePath: dashboardPath,
        component: AsyncDashboardPage,
    },
    {
        basePath: dashboardPathPF,
        component: AsyncDashboardPagePF,
    },
    {
        basePath: networkBasePath,
        propPath: networkPath,
        component: AsyncNetworkPage,
    },
    {
        basePath: violationsBasePath,
        propPath: violationsPath,
        component: AsyncViolationsPage,
    },
    {
        basePath: complianceBasePath,
        propPath: compliancePath,
        component: AsyncCompliancePage,
    },

    // Sidebar VulnerabilityManagement
    // More specific paths must precede more generic path in React Router 5.1 but not in 6
    {
        basePath: vulnManagementRiskAcceptancePath,
        component: AsyncVulnMgmtRiskAcceptancePage,
    },
    {
        basePath: vulnManagementReportsPath,
        component: AsyncVulnMgmtReports,
    },
    {
        basePath: vulnManagementPath,
        component: AsyncVulnMgmtPage,
    },

    // Sidebar Unexpandable2
    {
        basePath: configManagementPath,
        component: AsyncConfigManagementPage,
    },
    {
        basePath: riskBasePath,
        propPath: riskPath,
        component: AsyncRiskPage,
    },

    // Sidebar PlatformConfiguration
    {
        basePath: clustersBasePath,
        propPath: clustersPathWithParam,
        component: AsyncClustersPage,
    },
    /*
    {
        basePath: clustersListPath,
        component: AsyncPFClustersPage,
    },
    */
    {
        basePath: policiesBasePath,
        propPath: policiesPath,
        component: AsyncPolicyManagementPage,
    },
    {
        basePath: integrationsPath,
        component: AsyncIntegrationsPage,
    },
    {
        basePath: accessControlBasePathV2,
        propPath: accessControlPathV2,
        component: AsyncAccessControlPageV2,
    },
    {
        basePath: systemConfigPath,
        component: AsyncSystemConfigPage,
    },
    {
        basePath: systemHealthPath,
        component: AsyncSystemHealthPage,
    },
    {
        basePath: systemHealthPathPF,
        component: AsyncSystemHealthPagePF,
    },

    // Header
    {
        basePath: apidocsPath,
        component: AsyncApiDocsPage,
    },
    // Help Center is an external link to /docs/product
    {
        basePath: userBasePath,
        component: AsyncUserPage,
    },
];

type BodyProps = {
    isFeatureFlagEnabled: IsFeatureFlagEnabled;
    isRenderedRoutePath: IsRenderedRoutePath;
};

function Body({ isFeatureFlagEnabled, isRenderedRoutePath }: BodyProps): ReactElement {
    const { isDarkMode } = useTheme();

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
                        path={deprecatedPoliciesPath}
                        exact
                        render={() => <Redirect to={policiesPath} />}
                    />
                    {routeComponents
                        .filter(({ basePath }) => isRenderedRoutePath(basePath))
                        .map((routeComponent) => {
                            const { basePath, propPath } = routeComponent;
                            const path = propPath ?? basePath;

                            if ('featureFlagDependency' in routeComponent) {
                                const {
                                    componentDisabled,
                                    componentEnabled,
                                    featureFlagDependency,
                                } = routeComponent;
                                const component = isFeatureFlagEnabled(featureFlagDependency)
                                    ? componentEnabled
                                    : componentDisabled;
                                return <Route key={basePath} path={path} component={component} />;
                            }

                            const { component } = routeComponent;
                            return <Route key={basePath} path={path} component={component} />;
                        })}
                    <Route component={NotFoundPage} />
                </Switch>
            </ErrorBoundary>
        </div>
    );
}

export default Body;
