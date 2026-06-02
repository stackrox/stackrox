import { useQuery } from '@tanstack/react-query';

import { getPolicies } from 'services/PoliciesService';
import type { ListPolicy } from 'types/policy.proto';

import { CommandCenterLayout } from 'design-system/layout/command-center-layout';
import { Badge } from 'design-system/ui/badge';
import { Skeleton } from 'design-system/ui/skeleton';
import { cn } from 'design-system/lib/utils';

type SeverityVariant = 'critical' | 'high' | 'medium' | 'low';

function getSeverityVariant(severity: string): SeverityVariant {
    switch (severity) {
        case 'CRITICAL_SEVERITY':
            return 'critical';
        case 'HIGH_SEVERITY':
            return 'high';
        case 'MEDIUM_SEVERITY':
            return 'medium';
        default:
            return 'low';
    }
}

function getSeverityLabel(severity: string): string {
    switch (severity) {
        case 'CRITICAL_SEVERITY':
            return 'Critical';
        case 'HIGH_SEVERITY':
            return 'High';
        case 'MEDIUM_SEVERITY':
            return 'Medium';
        case 'LOW_SEVERITY':
            return 'Low';
        default:
            return severity;
    }
}

function formatLifecycle(stages: string[]): string {
    return stages.map((s) => s.charAt(0) + s.slice(1).toLowerCase()).join(', ');
}

export default function PoliciesPageCC() {
    const { data: policies, isLoading } = useQuery({
        queryKey: ['policies', 'list'],
        queryFn: () => getPolicies(),
    });

    const enabledCount = policies?.filter((p) => !p.disabled).length ?? 0;
    const disabledCount = policies?.filter((p) => p.disabled).length ?? 0;

    return (
        <CommandCenterLayout title="Policies">
            <div className="flex h-full flex-col">
                {/* Header */}
                <div className="flex shrink-0 items-center gap-3 border-b border-border-subtle bg-bg-secondary px-5 py-3">
                    <h2 className="text-sm font-600 text-text-primary">Policy Management</h2>
                    {policies && (
                        <>
                            <Badge variant="success">{enabledCount} enabled</Badge>
                            <Badge variant="low">{disabledCount} disabled</Badge>
                        </>
                    )}
                </div>

                {/* Table */}
                <div className="flex-1 overflow-y-auto">
                    {isLoading ? (
                        <div className="space-y-2 p-5">
                            {Array.from({ length: 10 }).map((_, i) => (
                                <Skeleton key={`skeleton-${i}`} className="h-10 w-full" />
                            ))}
                        </div>
                    ) : (
                        <table className="w-full border-collapse">
                            <thead className="sticky top-0 z-10 bg-bg-secondary">
                                <tr>
                                    <th className="px-4 py-2 text-left text-2xs font-500 uppercase tracking-wide text-text-muted">
                                        Policy
                                    </th>
                                    <th className="px-4 py-2 text-left text-2xs font-500 uppercase tracking-wide text-text-muted">
                                        Severity
                                    </th>
                                    <th className="px-4 py-2 text-left text-2xs font-500 uppercase tracking-wide text-text-muted">
                                        Lifecycle
                                    </th>
                                    <th className="px-4 py-2 text-left text-2xs font-500 uppercase tracking-wide text-text-muted">
                                        Source
                                    </th>
                                    <th className="px-4 py-2 text-left text-2xs font-500 uppercase tracking-wide text-text-muted">
                                        Status
                                    </th>
                                </tr>
                            </thead>
                            <tbody>
                                {policies?.map((policy) => (
                                    <tr
                                        key={policy.id}
                                        className={cn(
                                            'border-b border-border-subtle transition-colors hover:bg-bg-hover',
                                            policy.disabled && 'opacity-50'
                                        )}
                                    >
                                        <td className="px-4 py-2.5">
                                            <div className="text-xs text-text-primary">
                                                {policy.name}
                                            </div>
                                            {policy.description && (
                                                <div className="mt-0.5 max-w-lg truncate text-2xs text-text-muted">
                                                    {policy.description}
                                                </div>
                                            )}
                                        </td>
                                        <td className="px-4 py-2.5">
                                            <Badge variant={getSeverityVariant(policy.severity)}>
                                                {getSeverityLabel(policy.severity)}
                                            </Badge>
                                        </td>
                                        <td className="px-4 py-2.5 text-2xs text-text-secondary">
                                            {formatLifecycle(policy.lifecycleStages)}
                                        </td>
                                        <td className="px-4 py-2.5">
                                            <span
                                                className={cn(
                                                    'text-2xs',
                                                    policy.isDefault
                                                        ? 'text-text-muted'
                                                        : 'text-accent-blue'
                                                )}
                                            >
                                                {policy.isDefault ? 'System' : 'Custom'}
                                            </span>
                                        </td>
                                        <td className="px-4 py-2.5">
                                            <span
                                                className={cn(
                                                    'inline-flex items-center gap-1.5 text-2xs',
                                                    policy.disabled
                                                        ? 'text-text-muted'
                                                        : 'text-success'
                                                )}
                                            >
                                                <span
                                                    className={cn(
                                                        'inline-block h-1.5 w-1.5 rounded-full',
                                                        policy.disabled
                                                            ? 'bg-severity-low'
                                                            : 'bg-success'
                                                    )}
                                                />
                                                {policy.disabled ? 'Disabled' : 'Enabled'}
                                            </span>
                                        </td>
                                    </tr>
                                ))}
                            </tbody>
                        </table>
                    )}
                </div>

                {/* Bottom bar */}
                <div className="flex h-8 shrink-0 items-center border-t border-border-subtle bg-bg-secondary px-5 text-2xs text-text-muted">
                    {policies && `${policies.length} policies`}
                </div>
            </div>
        </CommandCenterLayout>
    );
}
