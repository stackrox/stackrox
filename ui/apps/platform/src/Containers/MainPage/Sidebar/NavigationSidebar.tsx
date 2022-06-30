import React, { ReactElement } from 'react';
import { useLocation } from 'react-router-dom';
import { Nav, NavExpandable, NavItem, NavList, PageSidebar } from '@patternfly/react-core';

import LinkShim from 'Components/PatternFly/LinkShim';

import {
    IsRoutePathRendered,
    RoutePath,
    basePathToLabelMap,
    dashboardPath,
    networkBasePath,
    violationsBasePath,
    complianceBasePath,
    vulnManagementPath,
    vulnManagementReportsPath,
    vulnManagementRiskAcceptancePath,
    configManagementPath,
    riskBasePath,
    clustersBasePath,
    policiesBasePath,
    integrationsPath,
    accessControlBasePathV2,
    systemConfigPath,
    systemHealthPath,
} from 'routePaths';

/*
 * Nav item path like clustersBasePath = '/main/clusters' is key of routeDescriptorMap and basePathToLabelMap in routePaths.ts
 * not including parameters in path prop of some Route elements like path="/main/clusters/:clusterId?"
 */

const vulnerabilityManagementPaths: RoutePath[] = [
    vulnManagementPath,
    vulnManagementRiskAcceptancePath,
    vulnManagementReportsPath,
];

const platformConfigurationPaths: RoutePath[] = [
    clustersBasePath,
    policiesBasePath,
    integrationsPath,
    accessControlBasePathV2,
    systemConfigPath,
    systemHealthPath,
];

type NavigationSidebarProps = {
    isRoutePathRendered: IsRoutePathRendered;
};

function NavigationSidebar({ isRoutePathRendered }: NavigationSidebarProps): ReactElement {
    const { pathname } = useLocation();

    function isActiveFilter(routePath: RoutePath): boolean {
        return routePath === vulnManagementPath
            ? pathname === routePath
            : (pathname as string).startsWith(routePath);
    }

    const vulnerabilityManagementFilteredPaths =
        vulnerabilityManagementPaths.filter(isRoutePathRendered);
    const platformConfigurationFilteredPaths =
        platformConfigurationPaths.filter(isRoutePathRendered);

    // Special case for Vulnerability Management because nested nav items match only a subset of sub-routes.
    const isVulnerabilityManagementPathActive = pathname.startsWith(vulnManagementPath);
    const isPlatformConfigurationPathActive =
        platformConfigurationFilteredPaths.some(isActiveFilter);

    function navItemMapper(routePath: RoutePath): ReactElement {
        const isActive = isActiveFilter(routePath);
        return (
            <NavItem key={routePath} to={routePath} isActive={isActive} component={LinkShim}>
                {basePathToLabelMap[routePath]}
            </NavItem>
        );
    }

    const Navigation = (
        <Nav id="nav-primary-simple">
            <NavList id="nav-list-simple">
                {
                    // prettier-ignore
                    [
                        dashboardPath,
                        networkBasePath,
                        violationsBasePath,
                        complianceBasePath
                    ]
                        .filter(isRoutePathRendered)
                        .map(navItemMapper)
                }
                {vulnerabilityManagementFilteredPaths.length !== 0 && (
                    <NavExpandable
                        title="Vulnerability Management"
                        isActive={isVulnerabilityManagementPathActive}
                        isExpanded={isVulnerabilityManagementPathActive}
                    >
                        {vulnerabilityManagementFilteredPaths.map(navItemMapper)}
                    </NavExpandable>
                )}
                {
                    // prettier-ignore
                    [
                        configManagementPath,
                        riskBasePath
                    ]
                        .filter(isRoutePathRendered)
                        .map(navItemMapper)
                }
                {platformConfigurationFilteredPaths.length !== 0 && (
                    <NavExpandable
                        title="Platform Configuration"
                        isActive={isPlatformConfigurationPathActive}
                        isExpanded={isPlatformConfigurationPathActive}
                    >
                        {platformConfigurationFilteredPaths.map(navItemMapper)}
                    </NavExpandable>
                )}
            </NavList>
        </Nav>
    );

    return <PageSidebar nav={Navigation} />;
}

export default NavigationSidebar;
