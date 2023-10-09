import React, { ReactElement } from 'react';
import { matchPath, useLocation } from 'react-router-dom';
import { Nav, NavExpandable, NavList, PageSidebar } from '@patternfly/react-core';

import { IsFeatureFlagEnabled } from 'hooks/useFeatureFlags';
import { HasReadAccess } from 'hooks/usePermissions';

// Import path variables in alphabetical order to minimize merge conflicts when multiple people add routes.
import {
    RouteKey,
    accessControlBasePath,
    administrationEventsBasePath,
    clustersBasePath,
    collectionsBasePath,
    complianceBasePath,
    complianceEnhancedStatusPath,
    configManagementPath,
    dashboardPath,
    exceptionConfigurationPath,
    integrationsPath,
    isRouteEnabled, // predicate function
    listeningEndpointsBasePath,
    networkBasePath,
    policyManagementBasePath,
    riskBasePath,
    systemConfigPath,
    systemHealthPath,
    violationsBasePath,
    vulnManagementPath,
    vulnManagementReportsPath,
    vulnManagementRiskAcceptancePath,
    vulnerabilitiesWorkloadCvesPath,
    vulnerabilityReportsPath,
} from 'routePaths';

import NavigationContent from './NavigationContent';
import NavigationItem from './NavigationItem';

import './NavigationSidebar.css';

// Child example; Compliance (1.0) if Compliance (2.0) is rendered and Compliance otherwise.
// Parent example: Vulnerability Management (1.0) if Vulnerability Management (2.0) is rendered and so on.
type TitleCallback = (navDescriptionFiltered: NavDescription[]) => string;

// Child conditional title finds path to decide presence or absence of counterpart child.
// Parent conditional title finds key to decide presence or absence of counterpart parent.
const keyForNetwork = 'Network';
const keyForPlatformConfiguration = 'Platform Configuration';
const keyForVulnerabilityManagement1 = 'Vulnerability Management (1.0)';
const keyForVulnerabilityManagement2 = 'Vulnerability Management (2.0)';
const keyForCompliance2 = 'Compliance (2.0)';
type IsActiveCallback = (pathname: string) => boolean;

type ChildDescription = {
    type: 'child';
    content: string | TitleCallback | ReactElement;
    path: string;
    routeKey: RouteKey;
    isActive?: IsActiveCallback; // for example, exact match
};

// Encapsulate whether path match for child is specific or generic.
function isActiveChild(pathname: string, { isActive, path }: ChildDescription) {
    return typeof isActive === 'function' ? isActive(pathname) : Boolean(matchPath(pathname, path));
}

type ParentDescription = {
    type: 'parent';
    title: string | TitleCallback;
    key: string; // for key prop and especially for title callback
    children: ChildDescription[];
};

type NavDescription = ChildDescription | ParentDescription;

const navDescriptions: NavDescription[] = [
    {
        type: 'child',
        content: 'Dashboard',
        path: dashboardPath,
        routeKey: 'dashboard',
    },
    {
        type: 'parent',
        title: 'Network',
        key: keyForNetwork,
        children: [
            {
                type: 'child',
                content: 'Network Graph',
                path: networkBasePath,
                routeKey: 'network-graph',
            },
            {
                type: 'child',
                content: 'Listening Endpoints',
                path: listeningEndpointsBasePath,
                routeKey: 'listening-endpoints',
            },
        ],
    },
    {
        type: 'child',
        content: 'Violations',
        path: violationsBasePath,
        routeKey: 'violations',
    },
    {
        type: 'parent',
        title: (navDescriptionsFiltered) =>
            navDescriptionsFiltered.some(
                (navDescription) =>
                    navDescription.type === 'child' && navDescription.routeKey === 'compliance'
            )
                ? 'Compliance (2.0)'
                : 'Compliance',
        key: keyForCompliance2,
        children: [
            {
                type: 'child',
                content: 'Compliance Status',
                path: complianceEnhancedStatusPath,
                routeKey: 'compliance-enhanced',
            },
        ],
    },
    {
        type: 'child',
        content: (navDescriptionsFiltered) =>
            navDescriptionsFiltered.some(
                (navDescription) =>
                    navDescription.type === 'parent' && navDescription.key === keyForCompliance2
            )
                ? 'Compliance (1.0)'
                : 'Compliance',
        path: complianceBasePath,
        routeKey: 'compliance',
    },
    {
        type: 'parent',
        title: (navDescriptionsFiltered) =>
            navDescriptionsFiltered.some(
                (navDescription) =>
                    navDescription.type === 'parent' &&
                    navDescription.key === keyForVulnerabilityManagement1
            )
                ? 'Vulnerability Management (2.0)'
                : 'Vulnerability Management',
        key: keyForVulnerabilityManagement2,
        children: [
            {
                type: 'child',
                content: <NavigationContent variant="TechPreview">Workload CVEs</NavigationContent>,
                path: vulnerabilitiesWorkloadCvesPath,
                routeKey: 'workload-cves',
            },
            {
                type: 'child',
                content: 'Vulnerability Reporting',
                path: vulnerabilityReportsPath,
                routeKey: 'vulnerabilities/reports',
            },
        ],
    },
    {
        type: 'parent',
        key: keyForVulnerabilityManagement1,
        title: (navDescriptionsFiltered) =>
            navDescriptionsFiltered.some(
                (navDescription) =>
                    navDescription.type === 'parent' &&
                    navDescription.key === keyForVulnerabilityManagement2
            )
                ? 'Vulnerability Management (1.0)'
                : 'Vulnerability Management',
        children: [
            {
                type: 'child',
                content: 'Dashboard',
                path: vulnManagementPath,
                routeKey: 'vulnerability-management',
                isActive: (pathname) =>
                    Boolean(matchPath(pathname, { vulnManagementPath, exact: true })),
            },
            {
                type: 'child',
                content: 'Risk Acceptance',
                path: vulnManagementRiskAcceptancePath,
                routeKey: 'vulnerability-management/risk-acceptance',
            },
            {
                type: 'child',
                content: 'Reporting',
                path: vulnManagementReportsPath,
                routeKey: 'vulnerability-management/reports',
            },
        ],
    },
    {
        type: 'child',
        content: 'Configuration Management',
        path: configManagementPath,
        routeKey: 'configmanagement',
    },
    {
        type: 'child',
        content: 'Risk',
        path: riskBasePath,
        routeKey: 'risk',
    },
    {
        type: 'parent',
        key: 'Platform Configuration',
        title: keyForPlatformConfiguration,
        children: [
            {
                type: 'child',
                content: 'Clusters',
                path: clustersBasePath,
                routeKey: 'clusters',
            },
            {
                type: 'child',
                content: 'Policy Management',
                path: policyManagementBasePath,
                routeKey: 'policy-management',
            },
            {
                type: 'child',
                content: 'Collections',
                path: collectionsBasePath,
                routeKey: 'collections',
            },
            {
                type: 'child',
                content: 'Integrations',
                path: integrationsPath,
                routeKey: 'integrations',
            },
            {
                type: 'child',
                content: 'Exception Configuration',
                path: exceptionConfigurationPath,
                routeKey: 'exception-configuration',
            },
            {
                type: 'child',
                content: 'Access Control',
                path: accessControlBasePath,
                routeKey: 'access-control',
            },
            {
                type: 'child',
                content: 'System Configuration',
                path: systemConfigPath,
                routeKey: 'systemconfig',
            },
            {
                type: 'child',
                content: 'Administration Events',
                path: administrationEventsBasePath,
                routeKey: 'administration-events',
            },
            {
                type: 'child',
                content: 'System Health',
                path: systemHealthPath,
                routeKey: 'system-health',
            },
        ],
    },
];

type NavigationSidebarProps = {
    hasReadAccess: HasReadAccess;
    isFeatureFlagEnabled: IsFeatureFlagEnabled;
};

function NavigationSidebar({
    hasReadAccess,
    isFeatureFlagEnabled,
}: NavigationSidebarProps): ReactElement {
    const { pathname } = useLocation();
    const routePredicates = { hasReadAccess, isFeatureFlagEnabled };
    const navDescriptionsFiltered = navDescriptions
        .map((navDescription) => {
            switch (navDescription.type) {
                case 'parent': {
                    // Filter second-level children.
                    return {
                        ...navDescription,
                        children: navDescription.children.filter(({ routeKey }) =>
                            isRouteEnabled(routePredicates, routeKey)
                        ),
                    };
                }
                default: {
                    return navDescription;
                }
            }
        })
        .filter((navDescription) => {
            // Filter first-level parents and children.
            switch (navDescription.type) {
                case 'parent': {
                    return navDescription.children.length !== 0;
                }
                default: {
                    return isRouteEnabled(routePredicates, navDescription.routeKey);
                }
            }
        });

    const Navigation = (
        <Nav>
            <NavList>
                {navDescriptionsFiltered.map((navDescription) => {
                    switch (navDescription.type) {
                        case 'parent': {
                            const { children, key, title } = navDescription;
                            // NavExpandable needs both isActive and isExpanded props to close
                            // when another child elsewhere becomes active.
                            // This depends on generic matchPath instead of specific isActive callback,
                            // otherwise Vulnerability Management closes for classic pages other than Dashboard.
                            const hasChildMatchPath = children.some(({ path }) =>
                                Boolean(matchPath(pathname, path))
                            );
                            return (
                                <NavExpandable
                                    key={key}
                                    isActive={hasChildMatchPath}
                                    isExpanded={hasChildMatchPath}
                                    title={
                                        typeof title === 'function'
                                            ? title(navDescriptionsFiltered)
                                            : title
                                    }
                                >
                                    {navDescription.children.map((childDescription) => {
                                        const { content, path } = childDescription;
                                        return (
                                            <NavigationItem
                                                key={path}
                                                isActive={isActiveChild(pathname, childDescription)}
                                                path={path}
                                                content={
                                                    typeof content === 'function'
                                                        ? content(navDescriptionsFiltered)
                                                        : content
                                                }
                                            />
                                        );
                                    })}
                                </NavExpandable>
                            );
                        }
                        default: {
                            const { content, path } = navDescription;
                            return (
                                <NavigationItem
                                    key={path}
                                    isActive={isActiveChild(pathname, navDescription)}
                                    path={path}
                                    content={
                                        typeof content === 'function'
                                            ? content(navDescriptionsFiltered)
                                            : content
                                    }
                                />
                            );
                        }
                    }
                })}
            </NavList>
        </Nav>
    );

    return <PageSidebar nav={Navigation} />;
}

export default NavigationSidebar;
