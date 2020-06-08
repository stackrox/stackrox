import * as Icon from 'react-feather';

import { knownBackendFlags } from 'utils/featureFlags';
import { configureLinks } from './NavigationPanel';

export const navLinks = [
    {
        text: 'Dashboard',
        to: '/main/dashboard',
        Icon: Icon.BarChart2,
    },
    {
        text: 'Network Graph',
        to: '/main/network',
        Icon: Icon.Share2,
    },
    {
        text: 'Violations',
        to: '/main/violations',
        Icon: Icon.AlertTriangle,
    },
    {
        text: 'Compliance',
        to: '/main/compliance',
        Icon: Icon.CheckSquare,
    },
    {
        text: 'Vulnerability Management',
        to: '/main/vulnerability-management',
        Icon: Icon.Layers,
        featureFlag: knownBackendFlags.ROX_VULN_MGMT_UI,
    },
    {
        text: 'Configuration Management',
        to: '/main/configmanagement',
        Icon: Icon.UserCheck,
    },
    {
        text: 'Risk',
        to: '/main/risk',
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
    text: 'API Reference',
    to: '/main/apidocs', // overrides dark mode: see .redoc-wrap rule in app.css
    Icon: Icon.Server,
};

export const productdocsLink = {
    text: 'Help Center',
    to: '/docs/product',
    Icon: Icon.HelpCircle,
};
