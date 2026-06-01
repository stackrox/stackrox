import { useQuery } from '@tanstack/react-query';

import { getComplianceProfilesStats } from 'services/ComplianceResultsStatsService';
import type { ComplianceProfileScanStats } from 'services/ComplianceResultsStatsService';
import type { ComplianceCheckStatusCount } from 'services/ComplianceCommon';
import { complianceKeys } from 'hooks/query/keys';

import { CommandCenterLayout } from 'design-system/layout/command-center-layout';
import { Card, CardHeader, CardTitle, CardContent } from 'design-system/ui/card';
import { Badge } from 'design-system/ui/badge';
import { Skeleton } from 'design-system/ui/skeleton';
import { cn } from 'design-system/lib/utils';

function getPassPercentage(checkStats: ComplianceCheckStatusCount[]): number {
    const total = checkStats.reduce((sum, s) => sum + s.count, 0);
    if (total === 0) return 0;
    const pass = checkStats.find((s) => s.status === 'PASS')?.count ?? 0;
    return Math.round((pass / total) * 100);
}

function getStatusCounts(checkStats: ComplianceCheckStatusCount[]) {
    const counts: Record<string, number> = {};
    for (const s of checkStats) {
        counts[s.status] = s.count;
    }
    return counts;
}

function getGaugeColor(pct: number): string {
    if (pct >= 80) return 'bg-success';
    if (pct >= 50) return 'bg-severity-medium';
    return 'bg-severity-critical';
}

function getGaugeTextColor(pct: number): string {
    if (pct >= 80) return 'text-success';
    if (pct >= 50) return 'text-severity-medium';
    return 'text-severity-critical';
}

function ProfileCard({ profile }: { profile: ComplianceProfileScanStats }) {
    const pct = getPassPercentage(profile.checkStats);
    const counts = getStatusCounts(profile.checkStats);
    const total = profile.checkStats.reduce((sum, s) => sum + s.count, 0);

    return (
        <Card>
            <CardHeader>
                <CardTitle>{profile.title || profile.profileName}</CardTitle>
                <span className="font-mono text-2xs text-text-muted">v{profile.version}</span>
            </CardHeader>
            <CardContent>
                {/* Gauge bar */}
                <div className="mb-3 flex items-center gap-3">
                    <div className="h-2 flex-1 overflow-hidden rounded-full bg-bg-primary">
                        <div
                            className={cn('h-full rounded-full transition-all', getGaugeColor(pct))}
                            style={{ width: `${pct}%` }}
                        />
                    </div>
                    <span className={cn('font-mono text-sm font-600', getGaugeTextColor(pct))}>
                        {pct}%
                    </span>
                </div>

                {/* Status breakdown */}
                <div className="grid grid-cols-4 gap-2">
                    <div className="rounded-lg bg-bg-primary p-2.5 text-center">
                        <div className="font-mono text-lg font-600 text-success">{counts.PASS ?? 0}</div>
                        <div className="text-2xs text-text-muted">Pass</div>
                    </div>
                    <div className="rounded-lg bg-bg-primary p-2.5 text-center">
                        <div className="font-mono text-lg font-600 text-severity-critical">{counts.FAIL ?? 0}</div>
                        <div className="text-2xs text-text-muted">Fail</div>
                    </div>
                    <div className="rounded-lg bg-bg-primary p-2.5 text-center">
                        <div className="font-mono text-lg font-600 text-severity-medium">{counts.MANUAL ?? 0}</div>
                        <div className="text-2xs text-text-muted">Manual</div>
                    </div>
                    <div className="rounded-lg bg-bg-primary p-2.5 text-center">
                        <div className="font-mono text-lg font-600 text-text-muted">
                            {(counts.ERROR ?? 0) + (counts.INFO ?? 0) + (counts.NOT_APPLICABLE ?? 0)}
                        </div>
                        <div className="text-2xs text-text-muted">Other</div>
                    </div>
                </div>

                <div className="mt-2 text-right text-2xs text-text-muted">{total} total checks</div>
            </CardContent>
        </Card>
    );
}

export default function CompliancePageCC() {
    const { data, isLoading, error } = useQuery({
        queryKey: complianceKeys.standards(),
        queryFn: () => getComplianceProfilesStats({}),
    });

    const profiles = data?.scanStats ?? [];

    return (
        <CommandCenterLayout title="Compliance">
            <div className="p-5">
                <div className="mb-4 flex items-center gap-3">
                    <h2 className="text-sm font-600 text-text-primary">Compliance Profiles</h2>
                    {profiles.length > 0 && (
                        <span className="rounded-full bg-bg-tertiary px-2 py-0.5 font-mono text-2xs text-text-muted">
                            {profiles.length}
                        </span>
                    )}
                </div>

                {isLoading ? (
                    <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
                        {Array.from({ length: 3 }).map((_, i) => (
                            <Skeleton key={i} className="h-48 w-full rounded-lg" />
                        ))}
                    </div>
                ) : error ? (
                    <div className="rounded-lg border border-severity-critical/30 bg-severity-critical/10 p-4 text-sm text-severity-critical">
                        Failed to load compliance data
                    </div>
                ) : profiles.length === 0 ? (
                    <div className="rounded-lg border border-border-subtle bg-bg-secondary p-8 text-center text-sm text-text-muted">
                        No compliance scan configurations found. Create a scan configuration to start monitoring compliance.
                    </div>
                ) : (
                    <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
                        {profiles.map((profile) => (
                            <ProfileCard key={profile.profileName} profile={profile} />
                        ))}
                    </div>
                )}
            </div>
        </CommandCenterLayout>
    );
}
