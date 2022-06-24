import React, { ReactElement } from 'react';
import { useLocation } from 'react-router-dom';
import { Nav, NavExpandable, NavItem, NavList, PageSidebar } from '@patternfly/react-core';

import LinkShim from 'Components/PatternFly/LinkShim';

import {
    IsRenderedRoutePath,
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

const unfilteredPathsVulnerabilityManagement: RoutePath[] = [
    vulnManagementPath,
    vulnManagementRiskAcceptancePath,
    vulnManagementReportsPath,
];

const unfilteredPathsPlatformConfiguration: RoutePath[] = [
    clustersBasePath,
    policiesBasePath,
    integrationsPath,
    accessControlBasePathV2,
    systemConfigPath,
    systemHealthPath,
];

type NavigationSidebarProps = {
    isRenderedRoutePath: IsRenderedRoutePath;
};

function NavigationSidebar({ isRenderedRoutePath }: NavigationSidebarProps): ReactElement {
    const { pathname } = useLocation();

    function isActiveFilter(routePath: RoutePath): boolean {
        return routePath === vulnManagementPath
            ? pathname === routePath
            : (pathname as string).startsWith(routePath);
    }

    const filteredPathsVulnerabilityManagement =
        unfilteredPathsVulnerabilityManagement.filter(isRenderedRoutePath);
    const filteredPathsPlatformConfiguration =
        unfilteredPathsPlatformConfiguration.filter(isRenderedRoutePath);

    // Special case because nested nav items match only a subset of sub-routes.
    const isActiveVulnerabilityManagement = pathname.startsWith(vulnManagementPath);
    const isActivePlatformConfiguration = filteredPathsPlatformConfiguration.some(isActiveFilter);

    function navItemMapper(routePath: RoutePath): ReactElement {
        const isActive = isActiveFilter(routePath);
        // Delete nullish coalescing and empty string if we replace string with RoutePath string union type.
        return (
            <NavItem key={routePath} to={routePath} isActive={isActive} component={LinkShim}>
                {basePathToLabelMap[routePath] ?? ''}
            </NavItem>
        );
    }

    const Navigation = (
        <Nav id="nav-primary-simple">
            <NavList id="nav-list-simple">
                {[dashboardPath, networkBasePath, violationsBasePath, complianceBasePath]
                    .filter(isRenderedRoutePath)
                    .map(navItemMapper)}
                {filteredPathsVulnerabilityManagement.length !== 0 && (
                    <NavExpandable
                        title="Vulnerability Management"
                        isActive={isActiveVulnerabilityManagement}
                        isExpanded={isActiveVulnerabilityManagement}
                    >
                        {filteredPathsVulnerabilityManagement.map(navItemMapper)}
                    </NavExpandable>
                )}
                {[configManagementPath, riskBasePath]
                    .filter(isRenderedRoutePath)
                    .map(navItemMapper)}
                {filteredPathsPlatformConfiguration.length !== 0 && (
                    <NavExpandable
                        title="Platform Configuration"
                        isActive={isActivePlatformConfiguration}
                        isExpanded={isActivePlatformConfiguration}
                    >
                        {filteredPathsPlatformConfiguration.map(navItemMapper)}
                    </NavExpandable>
                )}
            </NavList>
        </Nav>
    );

    return <PageSidebar nav={Navigation} />;
}

export default NavigationSidebar;
