import { useQuery } from '@tanstack/react-query';

import axios from 'services/instance';
import { fetchClusters } from 'services/ClustersService';

import { CommandCenterLayout } from 'design-system/layout/command-center-layout';
import { Card, CardHeader, CardTitle, CardContent } from 'design-system/ui/card';
import { Badge } from 'design-system/ui/badge';
import { Skeleton } from 'design-system/ui/skeleton';
import { cn } from 'design-system/lib/utils';

type IntegrationHealth = {
    id: string;
    name: string;
    type: string;
    status: string;
    errorMessage: string;
    lastTimestamp: string;
};

function HealthStatusDot({ status }: { status: string }) {
    return (
        <span className={cn(
            'inline-block h-2 w-2 rounded-full',
            status === 'HEALTHY' ? 'bg-success' :
            status === 'UNHEALTHY' ? 'bg-severity-critical' :
            status === 'UNINITIALIZED' ? 'bg-severity-low' :
            'bg-severity-medium'
        )} />
    );
}

function IntegrationHealthCard({ title, endpoint }: { title: string; endpoint: string }) {
    const { data, isLoading } = useQuery({
        queryKey: ['system-health', endpoint],
        queryFn: async () => {
            const response = await axios.get<{ integrationHealth: IntegrationHealth[] }>(endpoint);
            return response.data.integrationHealth ?? [];
        },
        refetchInterval: 30000,
    });

    const healthy = data?.filter((i) => i.status === 'HEALTHY').length ?? 0;
    const unhealthy = data?.filter((i) => i.status !== 'HEALTHY').length ?? 0;

    return (
        <Card>
            <CardHeader>
                <CardTitle>{title}</CardTitle>
                {!isLoading && data && (
                    <div className="flex gap-1.5">
                        {healthy > 0 && <Badge variant="success">{healthy} healthy</Badge>}
                        {unhealthy > 0 && <Badge variant="critical">{unhealthy} unhealthy</Badge>}
                    </div>
                )}
            </CardHeader>
            <CardContent>
                {isLoading ? (
                    <Skeleton className="h-16 w-full" />
                ) : data && data.length > 0 ? (
                    <div className="space-y-1.5">
                        {data.map((item) => (
                            <div key={item.id} className="flex items-center gap-2 rounded bg-bg-primary px-2.5 py-1.5">
                                <HealthStatusDot status={item.status} />
                                <span className="flex-1 text-xs text-text-primary">{item.name}</span>
                                <span className="text-2xs text-text-muted">{item.type}</span>
                            </div>
                        ))}
                    </div>
                ) : (
                    <p className="text-xs text-text-muted">No integrations configured</p>
                )}
            </CardContent>
        </Card>
    );
}

export default function SystemHealthPageCC() {
    const { data: clusters, isLoading: clustersLoading } = useQuery({
        queryKey: ['system-health', 'clusters'],
        queryFn: () => fetchClusters(),
        refetchInterval: 30000,
    });

    return (
        <CommandCenterLayout title="System Health">
            <div className="p-5">
                <div className="mb-4">
                    <h2 className="text-sm font-600 text-text-primary">System Health</h2>
                    <p className="mt-1 text-xs text-text-muted">
                        Monitor the health of clusters, integrations, and system components.
                    </p>
                </div>

                {/* Quick stats */}
                <div className="mb-4 grid grid-cols-2 gap-4 md:grid-cols-4">
                    <Card>
                        <CardContent className="pt-4">
                            {clustersLoading ? (
                                <Skeleton className="h-10 w-full" />
                            ) : (
                                <div className="text-center">
                                    <div className="font-mono text-3xl font-700 text-text-primary">{clusters?.length ?? 0}</div>
                                    <div className="mt-1 text-xs text-text-muted">Clusters</div>
                                </div>
                            )}
                        </CardContent>
                    </Card>
                </div>

                {/* Integration health cards */}
                <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
                    <IntegrationHealthCard
                        title="Image Integrations"
                        endpoint="/v1/integrationhealth/imageintegrations"
                    />
                    <IntegrationHealthCard
                        title="Notifier Integrations"
                        endpoint="/v1/integrationhealth/notifiers"
                    />
                    <IntegrationHealthCard
                        title="Backup Integrations"
                        endpoint="/v1/integrationhealth/externalbackups"
                    />
                </div>
            </div>
        </CommandCenterLayout>
    );
}
