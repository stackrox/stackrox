import { useQuery } from '@tanstack/react-query';

import { fetchClustersWithRetentionInfo } from 'services/ClustersService';
import { clusterKeys } from 'hooks/query/keys';
import type { Cluster } from 'types/cluster.proto';

import { CommandCenterLayout } from 'design-system/layout/command-center-layout';
import { Skeleton } from 'design-system/ui/skeleton';
import { cn } from 'design-system/lib/utils';

function getHealthLabel(status?: string): string {
    switch (status) {
        case 'HEALTHY':
            return 'Healthy';
        case 'DEGRADED':
            return 'Degraded';
        case 'UNHEALTHY':
            return 'Unhealthy';
        case 'UNINITIALIZED':
            return 'Uninitialized';
        default:
            return 'Unknown';
    }
}

function getStatusDotClass(status?: string): string {
    switch (status) {
        case 'HEALTHY':
            return 'bg-success';
        case 'DEGRADED':
            return 'bg-severity-medium';
        case 'UNHEALTHY':
            return 'bg-severity-critical';
        default:
            return 'bg-severity-low';
    }
}

function getTimeAgo(isoTime?: string): string {
    if (!isoTime) {
        return '—';
    }
    const diff = Date.now() - new Date(isoTime).getTime();
    const seconds = Math.floor(diff / 1000);
    if (seconds < 60) {
        return `${seconds}s ago`;
    }
    const minutes = Math.floor(seconds / 60);
    if (minutes < 60) {
        return `${minutes}m ago`;
    }
    const hours = Math.floor(minutes / 60);
    return `${hours}h ago`;
}

export default function ClustersPageCC() {
    const { data, isLoading } = useQuery({
        queryKey: clusterKeys.lists(),
        queryFn: () => fetchClustersWithRetentionInfo(),
        refetchInterval: 30000,
    });
    const clusters: Cluster[] = data?.clusters ?? [];

    return (
        <CommandCenterLayout title="Clusters">
            <div className="flex h-full flex-col">
                {/* Header */}
                <div className="flex shrink-0 items-center gap-3 border-b border-border-subtle bg-bg-secondary px-5 py-3">
                    <h2 className="text-sm font-600 text-text-primary">Secured Clusters</h2>
                    {clusters && (
                        <span className="rounded-full bg-bg-tertiary px-2 py-0.5 font-mono text-2xs text-text-muted">
                            {clusters.length}
                        </span>
                    )}
                </div>

                {/* Table */}
                <div className="flex-1 overflow-y-auto">
                    {isLoading ? (
                        <div className="space-y-2 p-5">
                            {Array.from({ length: 5 }).map((_, i) => (
                                <Skeleton key={`skeleton-${i}`} className="h-12 w-full" />
                            ))}
                        </div>
                    ) : (
                        <table className="w-full border-collapse">
                            <thead className="sticky top-0 z-10 bg-bg-secondary">
                                <tr>
                                    <th className="px-4 py-2 text-left text-2xs font-500 uppercase tracking-wide text-text-muted">
                                        Cluster
                                    </th>
                                    <th className="px-4 py-2 text-left text-2xs font-500 uppercase tracking-wide text-text-muted">
                                        Status
                                    </th>
                                    <th className="px-4 py-2 text-left text-2xs font-500 uppercase tracking-wide text-text-muted">
                                        Type
                                    </th>
                                    <th className="px-4 py-2 text-left text-2xs font-500 uppercase tracking-wide text-text-muted">
                                        Sensor Version
                                    </th>
                                    <th className="px-4 py-2 text-right text-2xs font-500 uppercase tracking-wide text-text-muted">
                                        Last Contact
                                    </th>
                                </tr>
                            </thead>
                            <tbody>
                                {clusters?.map((cluster) => {
                                    const overallStatus = cluster.healthStatus?.overallHealthStatus;
                                    return (
                                        <tr
                                            key={cluster.id}
                                            className="border-b border-border-subtle transition-colors hover:bg-bg-hover"
                                        >
                                            <td className="px-4 py-2.5 text-xs font-500 text-text-primary">
                                                {cluster.name}
                                            </td>
                                            <td className="px-4 py-2.5">
                                                <span className="inline-flex items-center gap-1.5 text-xs text-text-secondary">
                                                    <span
                                                        className={cn(
                                                            'inline-block h-1.5 w-1.5 rounded-full',
                                                            getStatusDotClass(overallStatus)
                                                        )}
                                                    />
                                                    {getHealthLabel(overallStatus)}
                                                </span>
                                            </td>
                                            <td className="px-4 py-2.5 font-mono text-2xs text-text-muted">
                                                {cluster.type}
                                            </td>
                                            <td className="px-4 py-2.5">
                                                <span
                                                    className={cn(
                                                        'font-mono text-2xs',
                                                        cluster.status?.sensorVersion
                                                            ? 'text-success'
                                                            : 'text-text-muted'
                                                    )}
                                                >
                                                    {cluster.status?.sensorVersion || '—'}
                                                </span>
                                            </td>
                                            <td className="px-4 py-2.5 text-right font-mono text-2xs text-text-muted">
                                                {getTimeAgo(cluster.healthStatus?.lastContact)}
                                            </td>
                                        </tr>
                                    );
                                })}
                                {clusters?.length === 0 && (
                                    <tr>
                                        <td
                                            colSpan={5}
                                            className="px-4 py-12 text-center text-sm text-text-muted"
                                        >
                                            No clusters found
                                        </td>
                                    </tr>
                                )}
                            </tbody>
                        </table>
                    )}
                </div>

                {/* Bottom bar */}
                <div className="flex h-8 shrink-0 items-center border-t border-border-subtle bg-bg-secondary px-5 text-2xs text-text-muted">
                    {clusters &&
                        `${clusters.length} ${clusters.length === 1 ? 'cluster' : 'clusters'}`}
                </div>
            </div>
        </CommandCenterLayout>
    );
}
