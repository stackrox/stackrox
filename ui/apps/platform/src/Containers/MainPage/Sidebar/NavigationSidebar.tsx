import React, { ReactElement } from 'react';
import { useLocation, Location } from 'react-router-dom';
import { Nav, NavList, NavExpandable, PageSidebar } from '@patternfly/react-core';

import { IsFeatureFlagEnabled } from 'hooks/useFeatureFlags';
import { HasReadAccess } from 'hooks/usePermissions';

import {
    basePathToLabelMap,
    dashboardPath,
    networkBasePath,
    networkPathPF,
    violationsBasePath,
    complianceBasePath,
    vulnManagementPath,
    vulnManagementReportsPath,
    vulnManagementRiskAcceptancePath,
    configManagementPath,
    riskBasePath,
    clustersBasePath,
    policyManagementBasePath,
    integrationsPath,
    accessControlBasePathV2,
    systemConfigPath,
    systemHealthPath,
    collectionsPath,
} from 'routePaths';

import LeftNavItem from './LeftNavItem';

type NavigationSidebarProps = {
    hasReadAccess: HasReadAccess;
    isFeatureFlagEnabled: IsFeatureFlagEnabled;
};

function NavigationSidebar({
    hasReadAccess,
    isFeatureFlagEnabled,
}: NavigationSidebarProps): ReactElement {
    const location: Location = useLocation();

    const vulnerabilityManagementPaths = [vulnManagementPath];
    if (
        hasReadAccess('VulnerabilityManagementRequests') ||
        hasReadAccess('VulnerabilityManagementApprovals')
    ) {
        vulnerabilityManagementPaths.push(vulnManagementRiskAcceptancePath);
    }
    if (hasReadAccess('VulnerabilityReports')) {
        vulnerabilityManagementPaths.push(vulnManagementReportsPath);
    }

    const platformConfigurationPaths = [
        clustersBasePath,
        policyManagementBasePath,
        integrationsPath,
        accessControlBasePathV2,
        systemConfigPath,
        systemHealthPath,
    ];

    // TODO
    // - This must be restricted based on permissions once the BE is in place https://issues.redhat.com/browse/ROX-12695
    // - See also https://issues.redhat.com/browse/ROX-12619
    if (isFeatureFlagEnabled('ROX_OBJECT_COLLECTIONS')) {
        // Insert 'Collections' after 'Policy Management'
        platformConfigurationPaths.splice(
            platformConfigurationPaths.indexOf(policyManagementBasePath) + 1,
            0,
            collectionsPath
        );
    }

    // TODO remove this temporary extra config menu item when the PF network graph goes live in the main menu
    if (isFeatureFlagEnabled('ROX_NETWORK_GRAPH_PATTERNFLY')) {
        platformConfigurationPaths.push(networkPathPF);
    }

    const Navigation = (
        <Nav id="nav-primary-simple">
            <NavList id="nav-list-simple">
                <LeftNavItem
                    isActive={location.pathname.includes(dashboardPath)}
                    path={dashboardPath}
                    title={basePathToLabelMap[dashboardPath]}
                />
                <LeftNavItem
                    isActive={
                        location.pathname.includes(networkBasePath) &&
                        !location.pathname.includes(networkPathPF)
                    }
                    path={networkBasePath}
                    title={basePathToLabelMap[networkBasePath]}
                />
                <LeftNavItem
                    isActive={location.pathname.includes(violationsBasePath)}
                    path={violationsBasePath}
                    title={basePathToLabelMap[violationsBasePath]}
                />
                <LeftNavItem
                    isActive={location.pathname.includes(complianceBasePath)}
                    path={complianceBasePath}
                    title={basePathToLabelMap[complianceBasePath]}
                />
                <NavExpandable
                    id="VulnerabilityManagement"
                    title="Vulnerability Management"
                    isActive={vulnerabilityManagementPaths.some((path) =>
                        location.pathname.includes(path)
                    )}
                    isExpanded={vulnerabilityManagementPaths.some((path) =>
                        location.pathname.includes(path)
                    )}
                >
                    {vulnerabilityManagementPaths.map((path) => {
                        const isActive =
                            path === vulnManagementPath ? false : location.pathname.includes(path);
                        return (
                            <LeftNavItem
                                key={path}
                                isActive={isActive}
                                path={path}
                                title={basePathToLabelMap[path]}
                            />
                        );
                    })}
                </NavExpandable>
                <LeftNavItem
                    isActive={location.pathname.includes(configManagementPath)}
                    path={configManagementPath}
                    title={basePathToLabelMap[configManagementPath]}
                />
                <LeftNavItem
                    isActive={location.pathname.includes(riskBasePath)}
                    path={riskBasePath}
                    title={basePathToLabelMap[riskBasePath]}
                />
                <NavExpandable
                    id="PlatformConfiguration"
                    title="Platform Configuration"
                    isActive={platformConfigurationPaths.some((path) =>
                        location.pathname.includes(path)
                    )}
                    isExpanded={platformConfigurationPaths.some((path) =>
                        location.pathname.includes(path)
                    )}
                >
                    {platformConfigurationPaths.map((path) => {
                        const isActive = location.pathname.includes(path);
                        return (
                            <LeftNavItem
                                key={path}
                                isActive={isActive}
                                path={path}
                                title={basePathToLabelMap[path]}
                            />
                        );
                    })}
                </NavExpandable>
            </NavList>
        </Nav>
    );

    return <PageSidebar nav={Navigation} />;
}

export default NavigationSidebar;
