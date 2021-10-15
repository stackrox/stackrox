import React, { ReactElement } from 'react';
import { useLocation, Location } from 'react-router-dom';
import { Nav, NavList, NavExpandable, PageSidebar } from '@patternfly/react-core';

import {
    dashboardPath,
    networkBasePath,
    violationsBasePath,
    complianceBasePath,
    vulnManagementPath,
    vulnManagementPoliciesPath,
    vulnManagementCVEsPath,
    vulnManagementClustersPath,
    vulnManagementNamespacesPath,
    vulnManagementDeploymentsPath,
    vulnManagementImagesPath,
    vulnManagementComponentsPath,
    vulnManagementNodesPath,
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

const vulnerabilityManagementPaths = [
    vulnManagementPath,
    vulnManagementPoliciesPath,
    vulnManagementCVEsPath,
    vulnManagementClustersPath,
    vulnManagementNamespacesPath,
    vulnManagementDeploymentsPath,
    vulnManagementImagesPath,
    vulnManagementComponentsPath,
    vulnManagementNodesPath,
];
const platformConfigurationPaths = [
    clustersBasePath,
    policiesBasePath,
    integrationsPath,
    accessControlBasePathV2,
    systemConfigPath,
    systemHealthPath,
];

function NavigationSideBar(): ReactElement {
    const location: Location = useLocation();

    const Navigation = (
        <Nav id="nav-primary-simple">
            <NavList id="nav-list-simple">
                <LeftNavItem
                    isActive={location.pathname.includes(dashboardPath)}
                    path={dashboardPath}
                />
                <LeftNavItem
                    isActive={location.pathname.includes(networkBasePath)}
                    path={networkBasePath}
                />
                <LeftNavItem
                    isActive={location.pathname.includes(violationsBasePath)}
                    path={violationsBasePath}
                />
                <LeftNavItem
                    isActive={location.pathname.includes(complianceBasePath)}
                    path={complianceBasePath}
                />
                <NavExpandable
                    id="Vulnerability Management"
                    title="Vulnerability Management"
                    isActive={vulnerabilityManagementPaths.some((paths) =>
                        location.pathname.includes(paths)
                    )}
                >
                    {vulnerabilityManagementPaths.map((path) => {
                        const isActive =
                            path === vulnManagementPath ? false : location.pathname.includes(path);
                        return <LeftNavItem key={path} isActive={isActive} path={path} />;
                    })}
                </NavExpandable>
                <LeftNavItem
                    isActive={location.pathname.includes(configManagementPath)}
                    path={configManagementPath}
                />
                <LeftNavItem
                    isActive={location.pathname.includes(riskBasePath)}
                    path={riskBasePath}
                />
                <NavExpandable
                    id="Platform Configuration"
                    title="Platform Configuration"
                    isActive={platformConfigurationPaths.some((path) =>
                        location.pathname.includes(path)
                    )}
                >
                    {platformConfigurationPaths.map((path) => {
                        const isActive = location.pathname.includes(path);
                        return <LeftNavItem key={path} isActive={isActive} path={path} />;
                    })}
                </NavExpandable>
            </NavList>
        </Nav>
    );

    return <PageSidebar nav={Navigation} />;
}

export default NavigationSideBar;
