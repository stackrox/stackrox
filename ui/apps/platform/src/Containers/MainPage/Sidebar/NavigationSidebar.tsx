import React, { ReactElement } from 'react';
import { useLocation, Location } from 'react-router-dom';
import { Nav, NavList, NavExpandable, PageSidebar } from '@patternfly/react-core';

import { IsFeatureFlagEnabled } from 'hooks/useFeatureFlags';
import { HasReadAccess } from 'hooks/usePermissions';

import {
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

import LeftNavItem from './LeftNavItem';

const platformConfigurationPaths = [
    clustersBasePath,
    policiesBasePath,
    integrationsPath,
    accessControlBasePathV2,
    systemConfigPath,
    systemHealthPath,
];

type NavigationSidebarProps = {
    hasReadAccess: HasReadAccess;
    // eslint-disable-next-line react/no-unused-prop-types
    isFeatureFlagEnabled: IsFeatureFlagEnabled;
};

function NavigationSidebar({ hasReadAccess }: NavigationSidebarProps): ReactElement {
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

    const Navigation = (
        <Nav id="nav-primary-simple">
            <NavList id="nav-list-simple">
                <LeftNavItem
                    isActive={location.pathname.includes(dashboardPath)}
                    path={dashboardPath}
                    title={basePathToLabelMap[dashboardPath]}
                />
                <LeftNavItem
                    isActive={location.pathname.includes(networkBasePath)}
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
