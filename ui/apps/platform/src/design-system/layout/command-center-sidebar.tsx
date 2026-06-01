import { useLocation, Link } from 'react-router-dom-v5-compat';
import {
    LayoutDashboard,
    AlertTriangle,
    Shield,
    CheckCircle,
    Network,
    Server,
    Settings,
    type LucideIcon,
} from 'lucide-react';

import {
    dashboardPath,
    violationsBasePath,
    vulnerabilitiesUserWorkloadsPath,
    complianceEnhancedCoveragePath,
    networkBasePath,
    clustersBasePath,
    integrationsPath,
} from 'routePaths';

import { cn } from 'design-system/lib/utils';
import {
    Tooltip,
    TooltipContent,
    TooltipProvider,
    TooltipTrigger,
} from 'design-system/ui/tooltip';

interface NavItem {
    icon: LucideIcon;
    label: string;
    path: string;
    matchPaths: string[];
}

const navItems: NavItem[] = [
    {
        icon: LayoutDashboard,
        label: 'Dashboard',
        path: dashboardPath,
        matchPaths: [dashboardPath],
    },
    {
        icon: AlertTriangle,
        label: 'Violations',
        path: violationsBasePath,
        matchPaths: [violationsBasePath],
    },
    {
        icon: Shield,
        label: 'Vulnerabilities',
        path: vulnerabilitiesUserWorkloadsPath,
        matchPaths: ['/main/vulnerabilities'],
    },
    {
        icon: CheckCircle,
        label: 'Compliance',
        path: complianceEnhancedCoveragePath,
        matchPaths: ['/main/compliance'],
    },
    {
        icon: Network,
        label: 'Network',
        path: networkBasePath,
        matchPaths: [networkBasePath, '/main/listening-endpoints'],
    },
    {
        icon: Server,
        label: 'Clusters',
        path: clustersBasePath,
        matchPaths: [clustersBasePath],
    },
];

const bottomNavItems: NavItem[] = [
    {
        icon: Settings,
        label: 'Platform Configuration',
        path: integrationsPath,
        matchPaths: [
            integrationsPath,
            '/main/access-control',
            '/main/system-config',
            '/main/system-health',
            '/main/policy-management',
        ],
    },
];

function isActive(pathname: string, matchPaths: string[]): boolean {
    return matchPaths.some((p) => pathname.startsWith(p));
}

function SidebarNavItem({ item }: { item: NavItem }) {
    const { pathname } = useLocation();
    const active = isActive(pathname, item.matchPaths);

    return (
        <Tooltip>
            <TooltipTrigger asChild>
                <Link
                    to={item.path}
                    className={cn(
                        'flex h-10 w-10 items-center justify-center rounded-lg transition-colors',
                        active
                            ? 'bg-bg-tertiary text-accent-blue'
                            : 'text-text-muted hover:bg-bg-hover hover:text-text-secondary'
                    )}
                >
                    <item.icon className="h-[18px] w-[18px]" />
                </Link>
            </TooltipTrigger>
            <TooltipContent side="right" sideOffset={8}>
                {item.label}
            </TooltipContent>
        </Tooltip>
    );
}

export function CommandCenterSidebar() {
    return (
        <TooltipProvider delayDuration={0}>
            <nav className="flex h-full w-14 flex-col items-center border-r border-border-subtle bg-bg-secondary py-3 gap-1">
                <Link
                    to={dashboardPath}
                    className="mb-4 flex h-8 w-8 items-center justify-center rounded-lg bg-gradient-to-br from-accent-blue to-accent-purple text-sm font-700 text-white"
                >
                    R
                </Link>

                {navItems.map((item) => (
                    <SidebarNavItem key={item.path} item={item} />
                ))}

                <div className="flex-1" />

                {bottomNavItems.map((item) => (
                    <SidebarNavItem key={item.path} item={item} />
                ))}
            </nav>
        </TooltipProvider>
    );
}
