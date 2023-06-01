import React, { ReactElement } from 'react';
import { useLocation, Location } from 'react-router-dom';
import { Nav, NavList, NavExpandable, PageSidebar } from '@patternfly/react-core';

import { IsFeatureFlagEnabled } from 'hooks/useFeatureFlags';
import { HasReadAccess } from 'hooks/usePermissions';

import {
    basePathToLabelMap,
    dashboardPath,
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
    accessControlBasePath,
    systemConfigPath,
    systemHealthPath,
    collectionsBasePath,
    vulnerabilitiesWorkloadCvesPath,
} from 'routePaths';

import LeftNavItem from './LeftNavItem';
import NetworkGraphNavItems from './NetworkGraphNavItems';

import './NavigationSidebar.css';
import BadgedNavItem from './BadgedNavItem';

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
        accessControlBasePath,
        systemConfigPath,
        systemHealthPath,
    ];

    if (isFeatureFlagEnabled('ROX_POSTGRES_DATASTORE') && hasReadAccess('WorkflowAdministration')) {
        // Insert 'Collections' after 'Policy Management'
        platformConfigurationPaths.splice(
            platformConfigurationPaths.indexOf(policyManagementBasePath) + 1,
            0,
            collectionsBasePath
        );
    }

    const vulnerabilitiesPaths = [vulnerabilitiesWorkloadCvesPath];
    const isWorkloadCvesEnabled =
        isFeatureFlagEnabled('ROX_VULN_MGMT_WORKLOAD_CVES') &&
        isFeatureFlagEnabled('ROX_POSTGRES_DATASTORE');

    const Navigation = (
        <Nav id="nav-primary-simple">
            <NavList id="nav-list-simple">
                <LeftNavItem
                    isActive={location.pathname.includes(dashboardPath)}
                    path={dashboardPath}
                    title={basePathToLabelMap[dashboardPath]}
                />
                <NetworkGraphNavItems isFeatureFlagEnabled={isFeatureFlagEnabled} />
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
                {isWorkloadCvesEnabled && (
                    <NavExpandable
                        id="Vulnerabilities"
                        title="Vulnerability Management (2.0)"
                        isActive={vulnerabilitiesPaths.some((path) =>
                            location.pathname.includes(path)
                        )}
                        isExpanded={vulnerabilitiesPaths.some((path) =>
                            location.pathname.includes(path)
                        )}
                    >
                        <BadgedNavItem
                            variant="TechPreview"
                            key={vulnerabilitiesWorkloadCvesPath}
                            isActive={location.pathname.includes(vulnerabilitiesWorkloadCvesPath)}
                            path={vulnerabilitiesWorkloadCvesPath}
                            title={basePathToLabelMap[vulnerabilitiesWorkloadCvesPath]}
                        />
                    </NavExpandable>
                )}
                <NavExpandable
                    id="VulnerabilityManagement"
                    title={
                        isWorkloadCvesEnabled
                            ? 'Vulnerability Management (1.0)'
                            : 'Vulnerability Management'
                    }
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
