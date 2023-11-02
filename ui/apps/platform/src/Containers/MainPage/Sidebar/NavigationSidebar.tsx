import React, { ReactElement } from 'react';
import { matchPath, useLocation } from 'react-router-dom';
import { Nav, NavExpandable, NavItemSeparator, NavList, PageSidebar } from '@patternfly/react-core';

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
    complianceEnhancedScanConfigsBasePath,
    configManagementPath,
    dashboardPath,
    exceptionConfigurationPath,
    exceptionManagementPath,
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

type LinkDescription = {
    type: 'link';
    content: string | TitleCallback | ReactElement;
    path: string;
    routeKey: RouteKey;
    isActive?: IsActiveCallback; // for example, exact match
};

// Encapsulate whether path match for child is specific or generic.
function isActiveLink(pathname: string, { isActive, path }: LinkDescription) {
    return typeof isActive === 'function' ? isActive(pathname) : Boolean(matchPath(pathname, path));
}

type SeparatorDescription = {
    type: 'separator';
    key: string; // corresponds to React key prop
};

type ChildDescription = LinkDescription | SeparatorDescription;

type ParentDescription = {
    type: 'parent';
    title: string | TitleCallback;
    key: string; // for key prop and especially for title callback
    children: ChildDescription[];
};

type NavDescription = LinkDescription | ParentDescription;

const navDescriptions: NavDescription[] = [
    {
        type: 'link',
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
                type: 'link',
                content: 'Network Graph',
                path: networkBasePath,
                routeKey: 'network-graph',
            },
            {
                type: 'link',
                content: 'Listening Endpoints',
                path: listeningEndpointsBasePath,
                routeKey: 'listening-endpoints',
            },
        ],
    },
    {
        type: 'link',
        content: 'Violations',
        path: violationsBasePath,
        routeKey: 'violations',
    },
    {
        type: 'parent',
        title: (navDescriptionsFiltered) =>
            navDescriptionsFiltered.some(
                (navDescription) =>
                    navDescription.type === 'link' && navDescription.routeKey === 'compliance'
            )
                ? 'Compliance (2.0)'
                : 'Compliance',
        key: keyForCompliance2,
        children: [
            {
                type: 'link',
                content: 'Compliance Status',
                path: complianceEnhancedStatusPath,
                routeKey: 'compliance-enhanced',
            },
            {
                type: 'link',
                content: 'Scheduling',
                path: complianceEnhancedScanConfigsBasePath,
                routeKey: 'compliance-enhanced',
            },
        ],
    },
    {
        type: 'link',
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
                type: 'link',
                content: <NavigationContent variant="TechPreview">Workload CVEs</NavigationContent>,
                path: vulnerabilitiesWorkloadCvesPath,
                routeKey: 'workload-cves',
            },
            {
                type: 'separator',
                key: 'following-cves',
            },
            {
                type: 'link',
                content: 'Exception Management',
                path: exceptionManagementPath,
                routeKey: 'exception-management',
            },
            {
                type: 'link',
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
                type: 'link',
                content: 'Dashboard',
                path: vulnManagementPath,
                routeKey: 'vulnerability-management',
                isActive: (pathname) =>
                    Boolean(matchPath(pathname, { vulnManagementPath, exact: true })),
            },
            {
                type: 'link',
                content: 'Risk Acceptance',
                path: vulnManagementRiskAcceptancePath,
                routeKey: 'vulnerability-management/risk-acceptance',
            },
            {
                type: 'link',
                content: 'Reporting',
                path: vulnManagementReportsPath,
                routeKey: 'vulnerability-management/reports',
            },
        ],
    },
    {
        type: 'link',
        content: 'Configuration Management',
        path: configManagementPath,
        routeKey: 'configmanagement',
    },
    {
        type: 'link',
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
                type: 'link',
                content: 'Clusters',
                path: clustersBasePath,
                routeKey: 'clusters',
            },
            {
                type: 'link',
                content: 'Policy Management',
                path: policyManagementBasePath,
                routeKey: 'policy-management',
            },
            {
                type: 'link',
                content: 'Collections',
                path: collectionsBasePath,
                routeKey: 'collections',
            },
            {
                type: 'link',
                content: 'Integrations',
                path: integrationsPath,
                routeKey: 'integrations',
            },
            {
                type: 'link',
                content: 'Exception Configuration',
                path: exceptionConfigurationPath,
                routeKey: 'exception-configuration',
            },
            {
                type: 'link',
                content: 'Access Control',
                path: accessControlBasePath,
                routeKey: 'access-control',
            },
            {
                type: 'link',
                content: 'System Configuration',
                path: systemConfigPath,
                routeKey: 'systemconfig',
            },
            {
                type: 'link',
                content: 'Administration Events',
                path: administrationEventsBasePath,
                routeKey: 'administration-events',
            },
            {
                type: 'link',
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

    function isChildLinkEnabled(childDescription: ChildDescription) {
        return childDescription.type === 'link'
            ? isRouteEnabled(routePredicates, childDescription.routeKey)
            : true;
    }

    function isChildSeparatorRelevant(
        childDescription: ChildDescription,
        index: number,
        array: ChildDescription[]
    ) {
        // A separator is relevant if it is preceded and followed by a link whose route is enabled.
        return childDescription.type === 'separator'
            ? index !== 0 && index !== array.length - 1 && array[index + 1].type === 'link'
            : true;
    }

    const navDescriptionsFiltered = navDescriptions
        .map((navDescription) => {
            switch (navDescription.type) {
                case 'parent': {
                    // Filter second-level children.
                    return {
                        ...navDescription,
                        children: navDescription.children
                            .filter(isChildLinkEnabled)
                            .filter(isChildSeparatorRelevant),
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
                            const hasChildMatchPath = children.some(
                                (childDescription) =>
                                    childDescription.type === 'link' &&
                                    Boolean(matchPath(pathname, childDescription.path))
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
                                        if (childDescription.type === 'link') {
                                            const { content, path } = childDescription;
                                            return (
                                                <NavigationItem
                                                    key={path}
                                                    isActive={isActiveLink(
                                                        pathname,
                                                        childDescription
                                                    )}
                                                    path={path}
                                                    content={
                                                        typeof content === 'function'
                                                            ? content(navDescriptionsFiltered)
                                                            : content
                                                    }
                                                />
                                            );
                                        }
                                        return <NavItemSeparator key={childDescription.key} />;
                                    })}
                                </NavExpandable>
                            );
                        }
                        default: {
                            const { content, path } = navDescription;
                            return (
                                <NavigationItem
                                    key={path}
                                    isActive={isActiveLink(pathname, navDescription)}
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
