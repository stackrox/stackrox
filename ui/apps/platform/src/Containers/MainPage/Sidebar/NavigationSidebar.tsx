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
    isRenderedRoutePath: IsRenderedRoutePath;
};

function NavigationSidebar({ isRenderedRoutePath }: NavigationSidebarProps): ReactElement {
    const { pathname } = useLocation();

    function isActiveFilter(routePath: RoutePath): boolean {
        return routePath === vulnManagementPath
            ? pathname === routePath
            : (pathname as string).startsWith(routePath);
    }

    const vulnerabilityManagementFilteredPaths =
        vulnerabilityManagementPaths.filter(isRenderedRoutePath);
    const platformConfigurationFilteredPaths =
        platformConfigurationPaths.filter(isRenderedRoutePath);

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
                {[dashboardPath, networkBasePath, violationsBasePath, complianceBasePath]
                    .filter(isRenderedRoutePath)
                    .map(navItemMapper)}
                {vulnerabilityManagementFilteredPaths.length !== 0 && (
                    <NavExpandable
                        title="Vulnerability Management"
                        isActive={isVulnerabilityManagementPathActive}
                        isExpanded={isVulnerabilityManagementPathActive}
                    >
                        {vulnerabilityManagementFilteredPaths.map(navItemMapper)}
                    </NavExpandable>
                )}
                {[configManagementPath, riskBasePath]
                    .filter(isRenderedRoutePath)
                    .map(navItemMapper)}
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
