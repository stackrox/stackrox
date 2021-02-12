import * as Icon from 'react-feather';

import {
    dashboardPath,
    networkBasePath,
    violationsBasePath,
    configManagementPath,
    complianceBasePath,
    vulnManagementPath,
    riskBasePath,
    apidocsPath,
    productDocsPath,
    basePathToLabelMap,
} from 'routePaths';
import { configureLinks } from './NavigationPanel';

export const navLinks = [
    {
        text: basePathToLabelMap[dashboardPath],
        to: dashboardPath,
        Icon: Icon.BarChart2,
    },
    {
        text: basePathToLabelMap[networkBasePath],
        to: networkBasePath,
        Icon: Icon.Share2,
    },
    {
        text: basePathToLabelMap[violationsBasePath],
        to: violationsBasePath,
        Icon: Icon.AlertTriangle,
    },
    {
        text: basePathToLabelMap[complianceBasePath],
        to: complianceBasePath,
        Icon: Icon.CheckSquare,
    },
    {
        text: basePathToLabelMap[vulnManagementPath],
        to: vulnManagementPath,
        Icon: Icon.Layers,
    },
    {
        text: basePathToLabelMap[configManagementPath],
        to: configManagementPath,
        Icon: Icon.UserCheck,
    },
    {
        text: basePathToLabelMap[riskBasePath],
        to: riskBasePath,
        Icon: Icon.ShieldOff,
    },
    {
        text: 'Platform Configuration',
        to: '',
        Icon: Icon.Settings,
        panelType: 'configure',
        data: 'configure',
        paths: configureLinks.map(({ to }) => to),
    },
];

export const apidocsLink = {
    text: basePathToLabelMap[apidocsPath],
    to: apidocsPath,
    Icon: Icon.Server,
};

export const productdocsLink = {
    text: basePathToLabelMap[productDocsPath],
    to: productDocsPath,
    Icon: Icon.HelpCircle,
};
