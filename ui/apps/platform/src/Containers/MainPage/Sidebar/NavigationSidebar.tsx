import React, { ReactElement } from 'react';
import { useLocation, Location } from 'react-router-dom';
import {
    Nav,
    NavList,
    NavExpandable,
    PageSidebar,
    Flex,
    FlexItem,
    Badge,
} from '@patternfly/react-core';

import { IsFeatureFlagEnabled } from 'hooks/useFeatureFlags';
import { HasReadAccess } from 'hooks/usePermissions';

import {
    basePathToLabelMap,
    dashboardPath,
    networkBasePath,
    networkBasePathPF,
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

    const Navigation = (
        <Nav id="nav-primary-simple">
            <NavList id="nav-list-simple">
                <LeftNavItem
                    isActive={location.pathname.includes(dashboardPath)}
                    path={dashboardPath}
                    title={basePathToLabelMap[dashboardPath]}
                />
                {isFeatureFlagEnabled('ROX_NETWORK_GRAPH_PATTERNFLY') && (
                    <LeftNavItem
                        isActive={location.pathname.includes(networkBasePathPF)}
                        path={networkBasePathPF}
                        title={
                            <Flex>
                                <FlexItem>Network Graph</FlexItem>
                                <FlexItem>
                                    <Badge
                                        style={{
                                            backgroundColor: 'var(--pf-global--palette--cyan-400)',
                                        }}
                                    >
                                        2.0 preview
                                    </Badge>
                                </FlexItem>
                            </Flex>
                        }
                    />
                )}
                <LeftNavItem
                    isActive={
                        location.pathname.includes(networkBasePath) &&
                        !location.pathname.includes(networkBasePathPF)
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

                {isFeatureFlagEnabled('ROX_VULN_MGMT_WORKLOAD_CVES') &&
                    isFeatureFlagEnabled('ROX_POSTGRES_DATASTORE') && (
                        // TODO We need to designate this as Tech Preview in a more standard way, based on UX guidance
                        <NavExpandable
                            id="Vulnerabilities"
                            title="Vulnerabilities (preview)"
                            isActive={vulnerabilitiesPaths.some((path) =>
                                location.pathname.includes(path)
                            )}
                            isExpanded={vulnerabilitiesPaths.some((path) =>
                                location.pathname.includes(path)
                            )}
                        >
                            {vulnerabilitiesPaths.map((path) => {
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
                    )}
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
