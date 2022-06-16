import React, { ReactElement } from 'react';
import { matchPath, useLocation } from 'react-router-dom';
import { Nav, NavExpandable, NavItem, NavList, PageSidebar } from '@patternfly/react-core';

import LinkShim from 'Components/PatternFly/LinkShim';

import {
    IsRenderedRoutePath,
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
 * Nav item path like violationsBasePath = '/main/violations' is key of routeDescriptorMap and basePathToLabelMap in routePaths.ts
 * not including parameters in path prop of some Route elements like path="/main/violations/:alertId?"
 */

const unfilteredPathsUnexpandable1: string[] = [
    dashboardPath,
    networkBasePath,
    violationsBasePath,
    complianceBasePath,
];

const unfilteredPathsVulnerabilityManagement: string[] = [
    vulnManagementPath,
    vulnManagementRiskAcceptancePath,
    vulnManagementReportsPath,
];

const unfilteredPathsUnexpandable2: string[] = [configManagementPath, riskBasePath];

const unfilteredPathsPlatformConfiguration: string[] = [
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

    function isActiveFilter(routePath: string): boolean {
        // React Router 5
        return matchPath(pathname, {
            path: routePath,
            strict: true,
            exact: routePath === vulnManagementPath,
        }) as boolean;
        /*
        // React Router 6
        return matchPath({
            path: routePath,
            caseSensitive: true,
            end: routePath === vulnManagementPath
        }, pathname);
        */
    }

    const filteredPathsUnexpandable1 = unfilteredPathsUnexpandable1.filter(isRenderedRoutePath);
    const filteredPathsVulnerabilityManagement =
        unfilteredPathsVulnerabilityManagement.filter(isRenderedRoutePath);
    const filteredPathsUnexpandable2 = unfilteredPathsUnexpandable2.filter(isRenderedRoutePath);
    const filteredPathsPlatformConfiguration =
        unfilteredPathsPlatformConfiguration.filter(isRenderedRoutePath);

    // Special case because nested nav items match only a subset of sub-routes.
    // React Router 5 see isActiveFilter above for args in React Router 6
    const isActiveVulnerabilityManagement = matchPath(pathname, {
        path: vulnManagementPath,
        strict: true,
        exact: false,
    }) as boolean;
    const isActivePlatformConfiguration = filteredPathsPlatformConfiguration.some(isActiveFilter);

    function navItemMapper(routePath: string): ReactElement {
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
                {filteredPathsUnexpandable1.map(navItemMapper)}
                {filteredPathsVulnerabilityManagement.length !== 0 && (
                    <NavExpandable
                        title="Vulnerability Management"
                        isActive={isActiveVulnerabilityManagement}
                        isExpanded={isActiveVulnerabilityManagement}
                    >
                        {filteredPathsVulnerabilityManagement.map(navItemMapper)}
                    </NavExpandable>
                )}
                {filteredPathsUnexpandable2.map(navItemMapper)}
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
