import { Link } from 'react-router-dom-v5-compat';
import { useQuery } from '@tanstack/react-query';

import { gqlClient } from 'init/graphqlClient';
import { cveKeys } from 'hooks/query/keys';
import useURLSearch from 'hooks/useURLSearch';
import useURLPagination from 'hooks/useURLPagination';
import useURLSort from 'hooks/useURLSort';
import { getRequestQueryStringForSearchFilter, getPaginationParams } from 'utils/searchUtils';
import type { SearchFilter } from 'types/search';

import { CommandCenterLayout } from 'design-system/layout/command-center-layout';
import { Badge } from 'design-system/ui/badge';
import { Skeleton } from 'design-system/ui/skeleton';
import { cn } from 'design-system/lib/utils';

const cveListQueryStr = `
    query getImageCVEList($query: String, $pagination: Pagination) {
        imageCVEs(query: $query, pagination: $pagination) {
            cve
            affectedImageCountBySeverity {
                critical { total }
                important { total }
                moderate { total }
                low { total }
            }
            topCVSS
            affectedImageCount
            firstDiscoveredInSystem
            publishedOn
            distroTuples {
                summary
                operatingSystem
                cvss
            }
        }
    }
`;

const cveCountQueryStr = `
    query getImageCVECount($query: String) {
        imageCVECount(query: $query)
    }
`;

type ImageCVE = {
    cve: string;
    affectedImageCountBySeverity: {
        critical: { total: number };
        important: { total: number };
        moderate: { total: number };
        low: { total: number };
    };
    topCVSS: number;
    affectedImageCount: number;
    firstDiscoveredInSystem: string | null;
    publishedOn: string | null;
    distroTuples: Array<{
        summary: string;
        operatingSystem: string;
        cvss: number;
    }>;
};

function getCvssVariant(cvss: number): 'critical' | 'high' | 'medium' | 'low' {
    if (cvss >= 9.0) return 'critical';
    if (cvss >= 7.0) return 'high';
    if (cvss >= 4.0) return 'medium';
    return 'low';
}

function formatDate(isoDate: string | null): string {
    if (!isoDate) return '—';
    return new Date(isoDate).toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' });
}

function getDaysAgo(isoDate: string | null): string {
    if (!isoDate) return '';
    const days = Math.floor((Date.now() - new Date(isoDate).getTime()) / 86400000);
    if (days === 0) return 'today';
    if (days === 1) return '1 day ago';
    return `${days} days ago`;
}

const sortFields = ['CVE', 'CVSS', 'Image Sha', 'CVE Created Time'];

export default function WorkloadCvesPageCC() {
    const { searchFilter } = useURLSearch();
    const { page, perPage, setPage } = useURLPagination(20);
    const { sortOption } = useURLSort({
        sortFields,
        defaultSortOption: { field: 'CVSS', direction: 'desc' },
    });

    const query = getRequestQueryStringForSearchFilter(searchFilter);

    const { data, isLoading } = useQuery({
        queryKey: [...cveKeys.list(searchFilter), page, perPage, sortOption],
        queryFn: () =>
            gqlClient.request<{ imageCVEs: ImageCVE[] }>(cveListQueryStr, {
                query,
                pagination: getPaginationParams({ page, perPage, sortOption }),
            }),
    });

    const { data: countData } = useQuery({
        queryKey: [...cveKeys.all, 'count', searchFilter],
        queryFn: () => gqlClient.request<{ imageCVECount: number }>(cveCountQueryStr, { query }),
    });

    const cves = data?.imageCVEs ?? [];
    const totalCount = countData?.imageCVECount ?? 0;

    return (
        <CommandCenterLayout title="Vulnerabilities">
            <div className="flex h-full flex-col">
                {/* Header */}
                <div className="flex shrink-0 items-center gap-3 border-b border-border-subtle bg-bg-secondary px-5 py-3">
                    <h2 className="text-sm font-600 text-text-primary">Workload CVEs</h2>
                    {totalCount > 0 && (
                        <span className="rounded-full bg-severity-critical/15 px-2 py-0.5 font-mono text-2xs text-severity-critical">
                            {totalCount.toLocaleString()}
                        </span>
                    )}
                </div>

                {/* Table */}
                <div className="flex-1 overflow-y-auto">
                    {isLoading ? (
                        <div className="space-y-2 p-5">
                            {Array.from({ length: 10 }).map((_, i) => (
                                <Skeleton key={i} className="h-10 w-full" />
                            ))}
                        </div>
                    ) : (
                        <table className="w-full border-collapse">
                            <thead className="sticky top-0 z-10 bg-bg-secondary">
                                <tr>
                                    <th className="px-4 py-2 text-left text-2xs font-500 uppercase tracking-wide text-text-muted">CVE</th>
                                    <th className="px-4 py-2 text-left text-2xs font-500 uppercase tracking-wide text-text-muted">Top CVSS</th>
                                    <th className="px-4 py-2 text-center text-2xs font-500 uppercase tracking-wide text-text-muted">Critical</th>
                                    <th className="px-4 py-2 text-center text-2xs font-500 uppercase tracking-wide text-text-muted">Important</th>
                                    <th className="px-4 py-2 text-center text-2xs font-500 uppercase tracking-wide text-text-muted">Moderate</th>
                                    <th className="px-4 py-2 text-center text-2xs font-500 uppercase tracking-wide text-text-muted">Low</th>
                                    <th className="px-4 py-2 text-right text-2xs font-500 uppercase tracking-wide text-text-muted">Images</th>
                                    <th className="px-4 py-2 text-right text-2xs font-500 uppercase tracking-wide text-text-muted">First Seen</th>
                                    <th className="px-4 py-2 text-right text-2xs font-500 uppercase tracking-wide text-text-muted">Published</th>
                                </tr>
                            </thead>
                            <tbody>
                                {cves.map((cve) => (
                                    <tr
                                        key={cve.cve}
                                        className="border-b border-border-subtle transition-colors hover:bg-bg-hover"
                                    >
                                        <td className="px-4 py-2.5">
                                            <span className="font-mono text-xs font-500 text-accent-blue">{cve.cve}</span>
                                        </td>
                                        <td className="px-4 py-2.5">
                                            <Badge variant={getCvssVariant(cve.topCVSS)}>
                                                {cve.topCVSS.toFixed(1)}
                                            </Badge>
                                        </td>
                                        <td className="px-4 py-2.5 text-center">
                                            {cve.affectedImageCountBySeverity.critical.total > 0 && (
                                                <Badge variant="critical">{cve.affectedImageCountBySeverity.critical.total}</Badge>
                                            )}
                                        </td>
                                        <td className="px-4 py-2.5 text-center">
                                            {cve.affectedImageCountBySeverity.important.total > 0 && (
                                                <Badge variant="high">{cve.affectedImageCountBySeverity.important.total}</Badge>
                                            )}
                                        </td>
                                        <td className="px-4 py-2.5 text-center">
                                            {cve.affectedImageCountBySeverity.moderate.total > 0 && (
                                                <Badge variant="medium">{cve.affectedImageCountBySeverity.moderate.total}</Badge>
                                            )}
                                        </td>
                                        <td className="px-4 py-2.5 text-center">
                                            {cve.affectedImageCountBySeverity.low.total > 0 && (
                                                <Badge variant="low">{cve.affectedImageCountBySeverity.low.total}</Badge>
                                            )}
                                        </td>
                                        <td className="px-4 py-2.5 text-right font-mono text-xs text-text-secondary">
                                            {cve.affectedImageCount}
                                        </td>
                                        <td className="px-4 py-2.5 text-right">
                                            <span className="font-mono text-2xs text-text-muted" title={formatDate(cve.firstDiscoveredInSystem)}>
                                                {getDaysAgo(cve.firstDiscoveredInSystem)}
                                            </span>
                                        </td>
                                        <td className="px-4 py-2.5 text-right font-mono text-2xs text-text-muted">
                                            {formatDate(cve.publishedOn)}
                                        </td>
                                    </tr>
                                ))}
                                {cves.length === 0 && (
                                    <tr>
                                        <td colSpan={9} className="px-4 py-12 text-center text-sm text-text-muted">
                                            No CVEs found
                                        </td>
                                    </tr>
                                )}
                            </tbody>
                        </table>
                    )}
                </div>

                {/* Bottom bar */}
                <div className="flex h-8 shrink-0 items-center justify-between border-t border-border-subtle bg-bg-secondary px-5 text-2xs text-text-muted">
                    <span>
                        {totalCount > 0
                            ? `Showing ${(page - 1) * perPage + 1}-${Math.min(page * perPage, totalCount)} of ${totalCount.toLocaleString()} CVEs`
                            : 'No CVEs'}
                    </span>
                    {totalCount > perPage && (
                        <div className="flex gap-1">
                            <button
                                type="button"
                                disabled={page === 1}
                                onClick={() => setPage(page - 1)}
                                className="rounded border border-border bg-bg-tertiary px-2 py-0.5 text-text-secondary disabled:opacity-50"
                            >
                                Prev
                            </button>
                            <button
                                type="button"
                                disabled={page * perPage >= totalCount}
                                onClick={() => setPage(page + 1)}
                                className="rounded border border-border bg-bg-tertiary px-2 py-0.5 text-text-secondary disabled:opacity-50"
                            >
                                Next
                            </button>
                        </div>
                    )}
                </div>
            </div>
        </CommandCenterLayout>
    );
}
