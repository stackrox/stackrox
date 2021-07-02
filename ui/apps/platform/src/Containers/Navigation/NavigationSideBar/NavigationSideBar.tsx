import React, { ReactElement } from 'react';
import { useLocation, Location } from 'react-router-dom';
import { Nav, NavList, NavExpandable, PageSidebar } from '@patternfly/react-core';

import {
    dashboardPath,
    networkBasePath,
    violationsBasePath,
    complianceBasePath,
    vulnManagementPath,
    configManagementPath,
    riskBasePath,
    clustersBasePath,
    policiesBasePath,
    integrationsPath,
    accessControlPath,
    accessControlBasePathV2,
    systemConfigPath,
    systemHealthPath,
} from 'routePaths';
import useFeatureFlagEnabled from 'hooks/useFeatureFlagEnabled';
import { knownBackendFlags } from 'utils/featureFlags';

import LeftNavItem from './LeftNavItem';

function NavigationSideBar(): ReactElement {
    const location: Location = useLocation();
    const isAccessControlV2 = useFeatureFlagEnabled(knownBackendFlags.ROX_SCOPED_ACCESS_CONTROL);

    const pathsExpandable = [
        clustersBasePath,
        policiesBasePath,
        integrationsPath,
        isAccessControlV2 ? accessControlBasePathV2 : accessControlPath,
        systemConfigPath,
        systemHealthPath,
    ];

    const Navigation = (
        <Nav id="nav-primary-simple">
            <NavList id="nav-list-simple">
                <LeftNavItem location={location} path={dashboardPath} />
                <LeftNavItem location={location} path={networkBasePath} />
                <LeftNavItem location={location} path={violationsBasePath} />
                <LeftNavItem location={location} path={complianceBasePath} />
                <LeftNavItem location={location} path={vulnManagementPath} />
                <LeftNavItem location={location} path={configManagementPath} />
                <LeftNavItem location={location} path={riskBasePath} />
                <NavExpandable
                    id="Platform Configuration"
                    title="Platform Configuration"
                    isActive={pathsExpandable.some((pathExpandable) =>
                        location.pathname.includes(pathExpandable)
                    )}
                >
                    {pathsExpandable.map((path) => (
                        <LeftNavItem key={path} location={location} path={path} />
                    ))}
                </NavExpandable>
            </NavList>
        </Nav>
    );

    return <PageSidebar nav={Navigation} />;
}

export default NavigationSideBar;
