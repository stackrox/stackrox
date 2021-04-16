import {
    dashboardPath,
    networkBasePath,
    violationsBasePath,
    configManagementPath,
    complianceBasePath,
    vulnManagementPath,
    riskBasePath,
    clustersBasePath,
    policiesBasePath,
    integrationsPath,
    accessControlPath,
    systemConfigPath,
    systemHealthPath,
    basePathToLabelMap,
} from 'routePaths';

export type NavItemData = {
    isGrouped?: boolean;
    label: string;
    to?: string;
    children?: NavItemData[];
};

export const navItems: NavItemData[] = [
    {
        label: basePathToLabelMap[dashboardPath],
        to: dashboardPath,
    },
    {
        label: basePathToLabelMap[networkBasePath],
        to: networkBasePath,
    },
    {
        label: basePathToLabelMap[violationsBasePath],
        to: violationsBasePath,
    },
    {
        label: basePathToLabelMap[complianceBasePath],
        to: complianceBasePath,
    },
    {
        label: basePathToLabelMap[vulnManagementPath],
        to: vulnManagementPath,
    },
    {
        label: basePathToLabelMap[configManagementPath],
        to: configManagementPath,
    },
    {
        label: basePathToLabelMap[riskBasePath],
        to: riskBasePath,
    },
    {
        isGrouped: true,
        label: 'Platform Configuration',
        children: [
            {
                label: basePathToLabelMap[clustersBasePath],
                to: clustersBasePath,
            },
            {
                label: basePathToLabelMap[policiesBasePath],
                to: policiesBasePath,
            },
            {
                label: basePathToLabelMap[integrationsPath],
                to: integrationsPath,
            },
            {
                label: basePathToLabelMap[accessControlPath],
                to: accessControlPath,
            },
            {
                label: basePathToLabelMap[systemConfigPath],
                to: systemConfigPath,
            },
            {
                label: basePathToLabelMap[systemHealthPath],
                to: systemHealthPath,
            },
        ],
    },
];

export default navItems;
