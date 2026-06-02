import { Link } from 'react-router-dom-v5-compat';
import { useQuery } from '@tanstack/react-query';
import pluralize from 'pluralize';

import { gqlClient } from 'init/graphqlClient';
import { summaryKeys, alertKeys, imageKeys, deploymentKeys } from 'hooks/query/keys';
import { useServiceQuery } from 'hooks/query/useServiceQuery';
import usePermissions from 'hooks/usePermissions';
import useURLSearch from 'hooks/useURLSearch';

import { fetchDeploymentsWithProcessInfo } from 'services/DeploymentsService';

import {
    clustersBasePath,
    configManagementPath,
    urlEntityListTypes,
    violationsBasePath,
    violationsFullViewPath,
    vulnerabilitiesAllImagesPath,
    riskBasePath,
} from 'routePaths';
import { resourceTypes } from 'constants/entityTypes';
import { severities } from 'constants/severities';
import {
    getRequestQueryStringForSearchFilter,
    getUrlQueryStringForSearchFilter,
    generatePathWithQuery,
} from 'utils/searchUtils';
import type { SearchFilter } from 'types/search';

import { CommandCenterLayout } from 'design-system/layout/command-center-layout';
import { Card, CardHeader, CardTitle, CardContent } from 'design-system/ui/card';
import { Badge } from 'design-system/ui/badge';
import { Skeleton } from 'design-system/ui/skeleton';
import { Separator } from 'design-system/ui/separator';

// GraphQL queries (extracted from original widget files)

const summaryCountsQuery = `
    query summary_counts {
        clusterCount
        nodeCount
        violationCount
        deploymentCount
        imageCount
        secretCount
    }
`;

const alertsBySeverityQuery = `
    query alertCountsBySeverity($lowQuery: String, $medQuery: String, $highQuery: String, $critQuery: String) {
        LOW_SEVERITY: violationCount(query: $lowQuery)
        MEDIUM_SEVERITY: violationCount(query: $medQuery)
        HIGH_SEVERITY: violationCount(query: $highQuery)
        CRITICAL_SEVERITY: violationCount(query: $critQuery)
    }
`;

const mostRecentAlertsQuery = `
    query mostRecentAlerts($query: String) {
        alerts: violations(
            query: $query
            pagination: { limit: 3, sortOption: { field: "Violation Time", reversed: true } }
        ) {
            id
            time
            deployment { name }
            resource { resourceType, name }
            policy { name, severity }
        }
    }
`;

const imagesAtMostRiskQuery = `
    query getImagesAtMostRisk($query: String) {
        images(
            query: $query
            pagination: { limit: 6, sortOption: { field: "Image Risk Priority", reversed: false } }
        ) {
            id
            name { remote, fullName }
            priority
            imageVulnerabilityCounter {
                important { total, fixable }
                critical { total, fixable }
            }
        }
    }
`;

type SummaryData = {
    clusterCount: number;
    nodeCount: number;
    violationCount: number;
    deploymentCount: number;
    imageCount: number;
    secretCount: number;
};

type AlertCounts = Record<string, number>;

type RecentAlert = {
    id: string;
    time: string;
    deployment: { name: string } | null;
    resource: { resourceType: string; name: string } | null;
    policy: { name: string; severity: string };
};

type ImageAtRisk = {
    id: string;
    name: { remote: string; fullName: string };
    priority: number;
    imageVulnerabilityCounter: {
        important: { total: number; fixable: number };
        critical: { total: number; fixable: number };
    };
};

// Helper to build severity query string
function searchQueryBySeverity(severity: string, searchFilter: SearchFilter) {
    return getRequestQueryStringForSearchFilter({ ...searchFilter, Severity: severity });
}

// === Summary Strip ===
function SummaryStrip() {
    const { data, isLoading } = useQuery({
        queryKey: summaryKeys.counts(),
        queryFn: () => gqlClient.request<SummaryData>(summaryCountsQuery),
    });

    const items = [
        {
            label: 'Violations',
            value: data?.violationCount,
            href: violationsFullViewPath,
            isCritical: true,
        },
        { label: 'Clusters', value: data?.clusterCount, href: clustersBasePath },
        {
            label: 'Deployments',
            value: data?.deploymentCount,
            href: `${configManagementPath}/${urlEntityListTypes[resourceTypes.DEPLOYMENT]}`,
        },
        {
            label: 'Images',
            value: data?.imageCount,
            href: generatePathWithQuery(
                vulnerabilitiesAllImagesPath,
                {},
                { customParams: { entityTab: 'Image' } }
            ),
        },
        {
            label: 'Nodes',
            value: data?.nodeCount,
            href: `${configManagementPath}/${urlEntityListTypes[resourceTypes.NODE]}`,
        },
        {
            label: 'Secrets',
            value: data?.secretCount,
            href: `${configManagementPath}/${urlEntityListTypes[resourceTypes.SECRET]}`,
        },
    ];

    return (
        <div className="flex gap-px overflow-hidden rounded-xl bg-border-subtle">
            {items.map((item) => (
                <Link
                    key={item.label}
                    to={item.href}
                    className="flex-1 bg-bg-secondary px-4 py-3.5 transition-colors hover:bg-bg-tertiary first:rounded-l-xl last:rounded-r-xl"
                >
                    <div className="text-2xs uppercase tracking-wide text-text-muted">
                        {item.label}
                    </div>
                    {isLoading ? (
                        <Skeleton className="mt-1 h-7 w-12" />
                    ) : (
                        <div
                            className={`mt-0.5 font-mono text-xl font-600 ${item.isCritical ? 'text-severity-critical' : 'text-text-primary'}`}
                        >
                            {item.value?.toLocaleString() ?? '—'}
                        </div>
                    )}
                </Link>
            ))}
        </div>
    );
}

// === Violations by Severity ===
function ViolationsBySeverity() {
    const { searchFilter } = useURLSearch();

    const { data: counts, isLoading: countsLoading } = useQuery({
        queryKey: alertKeys.summaryCounts(searchFilter),
        queryFn: () =>
            gqlClient.request<AlertCounts>(alertsBySeverityQuery, {
                lowQuery: searchQueryBySeverity(severities.LOW_SEVERITY, searchFilter),
                medQuery: searchQueryBySeverity(severities.MEDIUM_SEVERITY, searchFilter),
                highQuery: searchQueryBySeverity(severities.HIGH_SEVERITY, searchFilter),
                critQuery: searchQueryBySeverity(severities.CRITICAL_SEVERITY, searchFilter),
            }),
    });

    const { data: recentData, isLoading: recentLoading } = useQuery({
        queryKey: [...alertKeys.all, 'recent-critical', searchFilter],
        queryFn: () =>
            gqlClient.request<{ alerts: RecentAlert[] }>(mostRecentAlertsQuery, {
                query: searchQueryBySeverity(severities.CRITICAL_SEVERITY, searchFilter),
            }),
    });

    const isLoading = countsLoading || recentLoading;
    const totalCount = counts
        ? Object.values(counts).reduce((sum, c) => sum + (typeof c === 'number' ? c : 0), 0)
        : 0;

    const severityTiles = [
        {
            key: 'CRITICAL_SEVERITY',
            label: 'Critical',
            variant: 'critical' as const,
            borderClass: 'border-l-severity-critical',
        },
        {
            key: 'HIGH_SEVERITY',
            label: 'High',
            variant: 'high' as const,
            borderClass: 'border-l-severity-high',
        },
        {
            key: 'MEDIUM_SEVERITY',
            label: 'Medium',
            variant: 'medium' as const,
            borderClass: 'border-l-severity-medium',
        },
        {
            key: 'LOW_SEVERITY',
            label: 'Low',
            variant: 'low' as const,
            borderClass: 'border-l-severity-low',
        },
    ];

    return (
        <Card>
            <CardHeader>
                <CardTitle>
                    {isLoading
                        ? 'Policy violations by severity'
                        : `${totalCount} policy ${pluralize('violation', totalCount)} by severity`}
                </CardTitle>
                <Link to={violationsBasePath} className="text-2xs text-accent-blue hover:underline">
                    View all &rarr;
                </Link>
            </CardHeader>
            <CardContent>
                {isLoading ? (
                    <Skeleton className="h-20 w-full" />
                ) : (
                    <>
                        <div className="flex gap-2 mb-3">
                            {severityTiles.map((tile) => {
                                const count = counts?.[tile.key] ?? 0;
                                const qs = getUrlQueryStringForSearchFilter({
                                    ...searchFilter,
                                    Severity: tile.key,
                                });
                                return (
                                    <Link
                                        key={tile.key}
                                        to={`${violationsBasePath}&${qs}`}
                                        className={`flex-1 rounded-lg border border-border-subtle ${tile.borderClass} border-l-[3px] bg-bg-primary p-3 text-center transition-colors hover:bg-bg-tertiary`}
                                    >
                                        <div
                                            className={`font-mono text-2xl font-700 text-${tile.variant === 'critical' ? 'severity-critical' : tile.variant === 'high' ? 'severity-high' : tile.variant === 'medium' ? 'severity-medium' : 'severity-low'}`}
                                        >
                                            {count}
                                        </div>
                                        <div
                                            className={`mt-1 text-2xs uppercase tracking-wide text-${tile.variant === 'critical' ? 'severity-critical' : tile.variant === 'high' ? 'severity-high' : tile.variant === 'medium' ? 'severity-medium' : 'severity-low'}`}
                                        >
                                            {tile.label}
                                        </div>
                                    </Link>
                                );
                            })}
                        </div>
                        <div className="text-2xs text-text-muted mb-1.5">
                            Most recent critical violations
                        </div>
                        {recentData?.alerts?.map((alert) => (
                            <div
                                key={alert.id}
                                className="flex items-center gap-2.5 border-b border-border-subtle py-2 last:border-b-0"
                            >
                                <div className="h-2 w-2 shrink-0 rounded-full bg-severity-critical" />
                                <div className="min-w-0 flex-1">
                                    <div className="truncate text-xs text-text-primary">
                                        {alert.policy.name}
                                    </div>
                                    <div className="truncate font-mono text-2xs text-text-muted">
                                        {alert.deployment?.name ??
                                            alert.resource?.name ??
                                            'Unknown'}
                                    </div>
                                </div>
                                <div className="shrink-0 font-mono text-2xs text-text-muted">
                                    {getTimeAgo(alert.time)}
                                </div>
                            </div>
                        ))}
                    </>
                )}
            </CardContent>
        </Card>
    );
}

// === Images at Most Risk ===
function ImagesAtMostRisk() {
    const { searchFilter } = useURLSearch();

    const { data, isLoading } = useQuery({
        queryKey: imageKeys.atMostRisk(searchFilter),
        queryFn: () =>
            gqlClient.request<{ images: ImageAtRisk[] }>(imagesAtMostRiskQuery, {
                query: getRequestQueryStringForSearchFilter(searchFilter),
            }),
    });

    return (
        <Card>
            <CardHeader>
                <CardTitle>Images at Most Risk</CardTitle>
                <Link
                    to={vulnerabilitiesAllImagesPath}
                    className="text-2xs text-accent-blue hover:underline"
                >
                    View all &rarr;
                </Link>
            </CardHeader>
            <CardContent>
                {isLoading ? (
                    <Skeleton className="h-40 w-full" />
                ) : (
                    <table className="w-full">
                        <thead>
                            <tr className="border-b border-border-subtle">
                                <th className="pb-1.5 text-left text-2xs font-500 uppercase tracking-wide text-text-muted">
                                    Image
                                </th>
                                <th className="pb-1.5 text-center text-2xs font-500 uppercase tracking-wide text-text-muted">
                                    Critical
                                </th>
                                <th className="pb-1.5 text-center text-2xs font-500 uppercase tracking-wide text-text-muted">
                                    Important
                                </th>
                            </tr>
                        </thead>
                        <tbody>
                            {data?.images?.map((image) => (
                                <tr
                                    key={image.id}
                                    className="border-b border-border-subtle last:border-b-0 hover:bg-bg-hover"
                                >
                                    <td className="py-2 font-mono text-2xs text-text-primary">
                                        {image.name.fullName}
                                    </td>
                                    <td className="py-2 text-center">
                                        <Badge variant="critical">
                                            {image.imageVulnerabilityCounter.critical.total}
                                        </Badge>
                                    </td>
                                    <td className="py-2 text-center">
                                        <Badge variant="high">
                                            {image.imageVulnerabilityCounter.important.total}
                                        </Badge>
                                    </td>
                                </tr>
                            ))}
                        </tbody>
                    </table>
                )}
            </CardContent>
        </Card>
    );
}

// === Deployments at Most Risk ===
function DeploymentsAtMostRiskCC() {
    const { searchFilter } = useURLSearch();

    const { data: deployments, isLoading } = useServiceQuery(
        deploymentKeys.atMostRisk(searchFilter),
        () => {
            const { request, cancel } = fetchDeploymentsWithProcessInfo(
                searchFilter,
                { field: 'Deployment Risk Priority', reversed: false },
                0,
                5
            );
            return {
                request: request.then((results) => results.map(({ deployment }) => deployment)),
                cancel,
            };
        }
    );

    return (
        <Card>
            <CardHeader>
                <CardTitle>Deployments at Most Risk</CardTitle>
                <Link to={riskBasePath} className="text-2xs text-accent-blue hover:underline">
                    View all &rarr;
                </Link>
            </CardHeader>
            <CardContent>
                {isLoading ? (
                    <Skeleton className="h-32 w-full" />
                ) : (
                    <table className="w-full">
                        <thead>
                            <tr className="border-b border-border-subtle">
                                <th className="pb-1.5 text-left text-2xs font-500 uppercase tracking-wide text-text-muted">
                                    Deployment
                                </th>
                                <th className="pb-1.5 text-left text-2xs font-500 uppercase tracking-wide text-text-muted">
                                    Cluster
                                </th>
                                <th className="pb-1.5 text-right text-2xs font-500 uppercase tracking-wide text-text-muted">
                                    Risk
                                </th>
                            </tr>
                        </thead>
                        <tbody>
                            {deployments?.map((d) => {
                                const p = Number(d.priority);
                                return (
                                    <tr
                                        key={d.id}
                                        className="border-b border-border-subtle last:border-b-0 hover:bg-bg-hover"
                                    >
                                        <td className="py-2 text-xs text-text-primary">{d.name}</td>
                                        <td className="py-2 font-mono text-2xs text-text-muted">
                                            {d.cluster}
                                        </td>
                                        <td className="py-2 text-right">
                                            <Badge
                                                variant={
                                                    p <= 2 ? 'critical' : p <= 4 ? 'high' : 'medium'
                                                }
                                            >
                                                {p <= 2 ? 'Critical' : p <= 4 ? 'High' : 'Medium'}
                                            </Badge>
                                        </td>
                                    </tr>
                                );
                            })}
                        </tbody>
                    </table>
                )}
            </CardContent>
        </Card>
    );
}

// === Time helper ===
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

// === Main Dashboard Page ===
export default function DashboardPageCC() {
    return (
        <CommandCenterLayout title="Dashboard">
            <div className="flex flex-col gap-4 p-5">
                <SummaryStrip />

                <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
                    <ViolationsBySeverity />
                    <ImagesAtMostRisk />
                </div>

                <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
                    <DeploymentsAtMostRiskCC />
                </div>
            </div>
        </CommandCenterLayout>
    );
}
