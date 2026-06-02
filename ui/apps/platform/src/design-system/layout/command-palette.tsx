import { useEffect, useState, useCallback } from 'react';
import { useNavigate } from 'react-router-dom-v5-compat';
import { Command } from 'cmdk';
import {
    LayoutDashboard,
    AlertTriangle,
    Shield,
    CheckCircle,
    Network,
    Server,
    Settings,
    Key,
    Activity,
    FileText,
} from 'lucide-react';

import {
    dashboardPath,
    violationsBasePath,
    vulnerabilitiesUserWorkloadsPath,
    complianceEnhancedCoveragePath,
    networkBasePath,
    clustersBasePath,
    integrationsPath,
    accessControlBasePath,
    systemHealthPath,
    policyManagementBasePath,
} from 'routePaths';

import { Dialog, DialogContent } from 'design-system/ui/dialog';

const pages = [
    { icon: LayoutDashboard, label: 'Dashboard', path: dashboardPath },
    { icon: AlertTriangle, label: 'Violations', path: violationsBasePath },
    { icon: Shield, label: 'Workload CVEs', path: vulnerabilitiesUserWorkloadsPath },
    { icon: CheckCircle, label: 'Compliance', path: complianceEnhancedCoveragePath },
    { icon: Network, label: 'Network Graph', path: networkBasePath },
    { icon: Server, label: 'Clusters', path: clustersBasePath },
    { icon: Settings, label: 'Integrations', path: integrationsPath },
    { icon: Key, label: 'Access Control', path: accessControlBasePath },
    { icon: Activity, label: 'System Health', path: systemHealthPath },
    { icon: FileText, label: 'Policy Management', path: policyManagementBasePath },
];

interface CommandPaletteProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
}

export function CommandPalette({ open, onOpenChange }: CommandPaletteProps) {
    const navigate = useNavigate();
    const [search, setSearch] = useState('');

    useEffect(() => {
        if (!open) {
            setSearch('');
        }
    }, [open]);

    const handleSelect = useCallback(
        (path: string) => {
            onOpenChange(false);
            navigate(path);
        },
        [navigate, onOpenChange]
    );

    return (
        <Dialog open={open} onOpenChange={onOpenChange}>
            <DialogContent className="overflow-hidden p-0 max-w-lg">
                <Command className="[&_[cmdk-group-heading]]:px-2 [&_[cmdk-group-heading]]:font-500 [&_[cmdk-group-heading]]:text-text-muted [&_[cmdk-group]]:px-2 [&_[cmdk-input-wrapper]_svg]:h-4 [&_[cmdk-input-wrapper]_svg]:w-4 [&_[cmdk-input]]:h-11 [&_[cmdk-item]]:px-2 [&_[cmdk-item]]:py-2.5">
                    <Command.Input
                        value={search}
                        onValueChange={setSearch}
                        placeholder="Search pages, violations, CVEs..."
                        className="flex h-11 w-full rounded-md bg-transparent py-3 px-4 text-sm text-text-primary outline-none placeholder:text-text-muted border-b border-border-subtle"
                    />
                    <Command.List className="max-h-80 overflow-y-auto p-2">
                        <Command.Empty className="py-6 text-center text-sm text-text-muted">
                            No results found.
                        </Command.Empty>

                        <Command.Group heading="Pages">
                            {pages.map((page) => (
                                <Command.Item
                                    key={page.path}
                                    value={page.label}
                                    onSelect={() => handleSelect(page.path)}
                                    className="flex cursor-pointer items-center gap-2 rounded-md px-2 py-2 text-sm text-text-secondary hover:bg-bg-hover data-[selected=true]:bg-bg-hover data-[selected=true]:text-text-primary"
                                >
                                    <page.icon className="h-4 w-4 text-text-muted" />
                                    {page.label}
                                </Command.Item>
                            ))}
                        </Command.Group>
                    </Command.List>
                </Command>
            </DialogContent>
        </Dialog>
    );
}

export function useCommandPalette() {
    const [open, setOpen] = useState(false);

    useEffect(() => {
        function handleKeyDown(e: KeyboardEvent) {
            if (e.key === 'k' && (e.metaKey || e.ctrlKey)) {
                e.preventDefault();
                setOpen((prev) => !prev);
            }
        }

        document.addEventListener('keydown', handleKeyDown);
        return () => document.removeEventListener('keydown', handleKeyDown);
    }, []);

    return { open, setOpen };
}
