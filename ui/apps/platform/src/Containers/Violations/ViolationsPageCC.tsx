import { useState, useMemo, useCallback } from 'react';
import { Link } from 'react-router-dom-v5-compat';
import { useQuery } from '@tanstack/react-query';

import { fetchAlerts, fetchAlertCount } from 'services/AlertsService';
import { alertKeys } from 'hooks/query/keys';
import useURLSearch from 'hooks/useURLSearch';
import useURLSort from 'hooks/useURLSort';
import useURLPagination from 'hooks/useURLPagination';
import useURLStringUnion from 'hooks/useURLStringUnion';

import { violationsBasePath } from 'routePaths';
import type { SearchFilter } from 'types/search';
import type { ListAlert, DeploymentListAlert, ResourceListAlert } from 'types/alert.proto';
import type { SortOption } from 'types/table';
import { severityLabels } from 'messages/common';
import LIFECYCLE_STAGES from 'constants/lifecycleStages';

import { CommandCenterLayout } from 'design-system/layout/command-center-layout';
import { Badge } from 'design-system/ui/badge';
import { Skeleton } from 'design-system/ui/skeleton';
import { cn } from 'design-system/lib/utils';

import { violationStateTabs } from './types';
import type { ViolationStateTab } from './types';

function getEntityName(alert: ListAlert): string {
    if ('deployment' in alert) {
        return (alert as DeploymentListAlert).deployment.name;
    }
    if ('resource' in alert) {
        return (alert as ResourceListAlert).resource.name;
    }
    if ('node' in alert) {
        return alert.node.name;
    }
    return 'Unknown';
}

function getClusterNamespace(alert: ListAlert): string {
    const { clusterName, namespace } = alert.commonEntityInfo;
    return namespace ? `${clusterName} / ${namespace}` : clusterName;
}

function getTimeAgo(isoTime: string): string {
    const diff = Date.now() - new Date(isoTime).getTime();
    const minutes = Math.floor(diff / 60000);
    if (minutes < 1) {
        return 'just now';
    }
    if (minutes < 60) {
        return `${minutes}m ago`;
    }
    const hours = Math.floor(minutes / 60);
    if (hours < 24) {
        return `${hours}h ago`;
    }
    const days = Math.floor(hours / 24);
    return `${days}d ago`;
}

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

function getSeverityDotClass(severity: string): string {
    switch (severity) {
        case 'CRITICAL_SEVERITY':
            return 'bg-severity-critical';
        case 'HIGH_SEVERITY':
            return 'bg-severity-high';
        case 'MEDIUM_SEVERITY':
            return 'bg-severity-medium';
        default:
            return 'bg-severity-low';
    }
}

function getLifecycleLabel(stage: string): string {
    switch (stage) {
        case LIFECYCLE_STAGES.DEPLOY:
            return 'Deploy';
        case LIFECYCLE_STAGES.RUNTIME:
            return 'Runtime';
        case LIFECYCLE_STAGES.BUILD:
            return 'Build';
        default:
            return stage;
    }
}

const defaultSortOption: SortOption = { field: 'Violation Time', direction: 'desc' };
const sortFields = ['Policy', 'Severity', 'Violation Time'];

export default function ViolationsPageCC() {
    const { searchFilter, setSearchFilter } = useURLSearch();
    const { page, perPage, setPage, setPerPage } = useURLPagination(50);
    const { sortOption, getSortParams } = useURLSort({ sortFields, defaultSortOption });
    const [selectedTab, setSelectedTab] = useURLStringUnion('violationState', violationStateTabs);
    const [selectedAlertId, setSelectedAlertId] = useState<string | null>(null);

    const alertSearchFilter: SearchFilter = useMemo(
        () => ({ ...searchFilter, 'Violation State': selectedTab }),
        [searchFilter, selectedTab]
    );

    const { data: alerts, isLoading: alertsLoading } = useQuery({
        queryKey: [...alertKeys.list(alertSearchFilter), page, perPage, sortOption],
        queryFn: () => {
            const { request, cancel } = fetchAlerts({
                alertSearchFilter,
                sortOption,
                page,
                perPage,
            });
            return request;
        },
        refetchInterval: 5000,
    });

    const { data: alertCount } = useQuery({
        queryKey: alertKeys.count(alertSearchFilter),
        queryFn: () => {
            const { request } = fetchAlertCount(alertSearchFilter);
            return request;
        },
        refetchInterval: 5000,
    });

    const selectedAlert = useMemo(
        () => alerts?.find((a) => a.id === selectedAlertId) ?? null,
        [alerts, selectedAlertId]
    );

    const handleTabChange = useCallback(
        (tab: ViolationStateTab) => {
            setSelectedTab(tab);
            setSearchFilter({});
            setPage(1);
            setSelectedAlertId(null);
        },
        [setSelectedTab, setSearchFilter, setPage]
    );

    const tabs: { key: ViolationStateTab; label: string }[] = [
        { key: 'ACTIVE', label: 'Active' },
        { key: 'RESOLVED', label: 'Resolved' },
        { key: 'ATTEMPTED', label: 'Attempted' },
    ];

    return (
        <CommandCenterLayout title="Violations">
            <div className="flex h-full flex-col">
                {/* View tabs */}
                <div className="flex shrink-0 items-center gap-0 border-b border-border-subtle bg-bg-secondary px-5">
                    {tabs.map((tab) => (
                        <button
                            key={tab.key}
                            type="button"
                            onClick={() => handleTabChange(tab.key)}
                            className={cn(
                                'px-3.5 py-2 text-xs border-b-2 transition-colors',
                                selectedTab === tab.key
                                    ? 'text-accent-blue border-accent-blue'
                                    : 'text-text-muted border-transparent hover:text-text-secondary'
                            )}
                        >
                            {tab.label}
                            {alertCount !== undefined && tab.key === selectedTab && (
                                <span
                                    className={cn(
                                        'ml-1.5 rounded-full px-1.5 py-0.5 font-mono text-2xs',
                                        selectedTab === tab.key
                                            ? 'bg-accent-blue/15 text-accent-blue'
                                            : 'bg-bg-tertiary text-text-muted'
                                    )}
                                >
                                    {alertCount}
                                </span>
                            )}
                        </button>
                    ))}
                </div>

                {/* Content: table + optional detail panel */}
                <div className="flex flex-1 overflow-hidden">
                    {/* Table */}
                    <div className="flex-1 overflow-y-auto">
                        {alertsLoading && !alerts ? (
                            <div className="p-5 space-y-2">
                                {Array.from({ length: 8 }).map((_, i) => (
                                    <Skeleton key={`skeleton-${i}`} className="h-10 w-full" />
                                ))}
                            </div>
                        ) : (
                            <table className="w-full border-collapse">
                                <thead className="sticky top-0 z-10 bg-bg-secondary">
                                    <tr>
                                        <th className="w-8 px-3 py-2" />
                                        <th className="px-3 py-2 text-left text-2xs font-500 uppercase tracking-wide text-text-muted">
                                            Policy
                                        </th>
                                        <th className="px-3 py-2 text-left text-2xs font-500 uppercase tracking-wide text-text-muted">
                                            Severity
                                        </th>
                                        <th className="px-3 py-2 text-left text-2xs font-500 uppercase tracking-wide text-text-muted">
                                            Entity
                                        </th>
                                        <th className="px-3 py-2 text-left text-2xs font-500 uppercase tracking-wide text-text-muted">
                                            Cluster / Namespace
                                        </th>
                                        <th className="px-3 py-2 text-left text-2xs font-500 uppercase tracking-wide text-text-muted">
                                            Lifecycle
                                        </th>
                                        <th className="px-3 py-2 text-right text-2xs font-500 uppercase tracking-wide text-text-muted">
                                            Time
                                        </th>
                                    </tr>
                                </thead>
                                <tbody>
                                    {alerts?.map((alert) => {
                                        const severity = alert.policy.severity;
                                        const isSelected = alert.id === selectedAlertId;
                                        return (
                                            <tr
                                                key={alert.id}
                                                onClick={() =>
                                                    setSelectedAlertId(isSelected ? null : alert.id)
                                                }
                                                className={cn(
                                                    'cursor-pointer border-b border-border-subtle transition-colors',
                                                    isSelected
                                                        ? 'bg-bg-selected'
                                                        : 'hover:bg-bg-hover'
                                                )}
                                            >
                                                <td className="px-3 py-2.5">
                                                    <div
                                                        className={cn(
                                                            'h-2 w-2 rounded-full',
                                                            getSeverityDotClass(severity)
                                                        )}
                                                    />
                                                </td>
                                                <td className="px-3 py-2.5 text-xs text-text-primary">
                                                    {alert.policy.name}
                                                </td>
                                                <td className="px-3 py-2.5">
                                                    <Badge variant={getSeverityVariant(severity)}>
                                                        {severityLabels[severity] ?? severity}
                                                    </Badge>
                                                </td>
                                                <td className="px-3 py-2.5 font-mono text-2xs text-text-secondary">
                                                    {getEntityName(alert)}
                                                </td>
                                                <td className="px-3 py-2.5 font-mono text-2xs text-text-muted">
                                                    {getClusterNamespace(alert)}
                                                </td>
                                                <td className="px-3 py-2.5">
                                                    <span className="inline-flex items-center gap-1 rounded border border-border-subtle bg-bg-tertiary px-1.5 py-0.5 text-2xs text-text-muted">
                                                        {getLifecycleLabel(alert.lifecycleStage)}
                                                    </span>
                                                </td>
                                                <td className="px-3 py-2.5 text-right font-mono text-2xs text-text-muted">
                                                    {getTimeAgo(alert.time)}
                                                </td>
                                            </tr>
                                        );
                                    })}
                                    {alerts?.length === 0 && (
                                        <tr>
                                            <td
                                                colSpan={7}
                                                className="px-3 py-12 text-center text-sm text-text-muted"
                                            >
                                                No violations found
                                            </td>
                                        </tr>
                                    )}
                                </tbody>
                            </table>
                        )}

                        {/* Pagination */}
                        {alertCount !== undefined && alertCount > perPage && (
                            <div className="flex items-center justify-between border-t border-border-subtle bg-bg-secondary px-5 py-2 text-xs text-text-muted">
                                <span>
                                    Showing {(page - 1) * perPage + 1}-
                                    {Math.min(page * perPage, alertCount)} of {alertCount}
                                </span>
                                <div className="flex gap-1">
                                    <button
                                        type="button"
                                        disabled={page === 1}
                                        onClick={() => setPage(page - 1)}
                                        className="rounded border border-border bg-bg-tertiary px-2 py-1 text-text-secondary disabled:opacity-50"
                                    >
                                        Prev
                                    </button>
                                    <button
                                        type="button"
                                        disabled={page * perPage >= alertCount}
                                        onClick={() => setPage(page + 1)}
                                        className="rounded border border-border bg-bg-tertiary px-2 py-1 text-text-secondary disabled:opacity-50"
                                    >
                                        Next
                                    </button>
                                </div>
                            </div>
                        )}
                    </div>

                    {/* Detail panel */}
                    {selectedAlert && (
                        <div className="w-96 shrink-0 overflow-y-auto border-l border-border-subtle bg-bg-secondary">
                            <div className="border-b border-border-subtle p-4">
                                <div className="mb-2 flex items-center gap-2">
                                    <Badge
                                        variant={getSeverityVariant(selectedAlert.policy.severity)}
                                    >
                                        {severityLabels[selectedAlert.policy.severity]}
                                    </Badge>
                                    <span className="inline-flex items-center gap-1 rounded border border-border-subtle bg-bg-tertiary px-1.5 py-0.5 text-2xs text-text-muted">
                                        {getLifecycleLabel(selectedAlert.lifecycleStage)}
                                    </span>
                                </div>
                                <h2 className="text-base font-600 text-text-primary leading-snug">
                                    {selectedAlert.policy.name}
                                </h2>
                                <p className="mt-1 text-xs text-text-muted">
                                    Triggered {getTimeAgo(selectedAlert.time)}
                                </p>
                            </div>

                            <div className="border-b border-border-subtle p-4">
                                <div className="flex gap-2">
                                    <Link
                                        to={`${violationsBasePath}/${selectedAlert.id}`}
                                        className="flex-1 rounded-md bg-accent-blue py-1.5 text-center text-xs text-white hover:opacity-90"
                                    >
                                        View Details
                                    </Link>
                                </div>
                            </div>

                            <div className="border-b border-border-subtle p-4">
                                <h3 className="mb-2 text-2xs font-500 uppercase tracking-wide text-text-muted">
                                    Entity
                                </h3>
                                <DetailRow label="Name" value={getEntityName(selectedAlert)} mono />
                                <DetailRow
                                    label="Cluster"
                                    value={selectedAlert.commonEntityInfo.clusterName}
                                    mono
                                />
                                <DetailRow
                                    label="Namespace"
                                    value={selectedAlert.commonEntityInfo.namespace}
                                    mono
                                />
                            </div>

                            <div className="border-b border-border-subtle p-4">
                                <h3 className="mb-2 text-2xs font-500 uppercase tracking-wide text-text-muted">
                                    Policy
                                </h3>
                                <DetailRow label="Name" value={selectedAlert.policy.name} />
                                <DetailRow
                                    label="Categories"
                                    value={selectedAlert.policy.categories?.join(', ') ?? '—'}
                                />
                            </div>

                            <div className="p-4">
                                <h3 className="mb-2 text-2xs font-500 uppercase tracking-wide text-text-muted">
                                    Keyboard Shortcuts
                                </h3>
                                <div className="flex flex-wrap gap-2 text-2xs text-text-muted">
                                    <span>
                                        <kbd className="rounded border border-border bg-bg-tertiary px-1 py-0.5 font-mono text-2xs">
                                            j
                                        </kbd>{' '}
                                        /{' '}
                                        <kbd className="rounded border border-border bg-bg-tertiary px-1 py-0.5 font-mono text-2xs">
                                            k
                                        </kbd>{' '}
                                        navigate
                                    </span>
                                </div>
                            </div>
                        </div>
                    )}
                </div>

                {/* Bottom bar */}
                <div className="flex h-8 shrink-0 items-center border-t border-border-subtle bg-bg-secondary px-5 text-2xs text-text-muted">
                    <span>
                        {alertCount !== undefined
                            ? `${alertCount} ${selectedTab.toLowerCase()} ${alertCount === 1 ? 'violation' : 'violations'}`
                            : 'Loading...'}
                    </span>
                    <span className="ml-auto">
                        <kbd className="rounded border border-border bg-bg-tertiary px-1 font-mono text-2xs">
                            j
                        </kbd>
                        /
                        <kbd className="rounded border border-border bg-bg-tertiary px-1 font-mono text-2xs">
                            k
                        </kbd>{' '}
                        navigate ·{' '}
                        <kbd className="rounded border border-border bg-bg-tertiary px-1 font-mono text-2xs">
                            Enter
                        </kbd>{' '}
                        select ·{' '}
                        <kbd className="rounded border border-border bg-bg-tertiary px-1 font-mono text-2xs">
                            /
                        </kbd>{' '}
                        filter
                    </span>
                </div>
            </div>
        </CommandCenterLayout>
    );
}

function DetailRow({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
    return (
        <div className="flex items-center justify-between py-1">
            <span className="text-xs text-text-muted">{label}</span>
            <span className={cn('text-xs text-text-secondary', mono && 'font-mono text-2xs')}>
                {value || '—'}
            </span>
        </div>
    );
}
