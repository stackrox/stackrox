import { useQuery } from '@tanstack/react-query';
import { Link } from 'react-router-dom-v5-compat';
import { Image, Bell, Database, KeyRound, type LucideIcon } from 'lucide-react';

import { fetchIntegration } from 'services/IntegrationsService';
import axios from 'services/instance';

import { CommandCenterLayout } from 'design-system/layout/command-center-layout';
import { Card, CardHeader, CardTitle, CardContent } from 'design-system/ui/card';
import { Badge } from 'design-system/ui/badge';
import { Skeleton } from 'design-system/ui/skeleton';

type IntegrationCategory = {
    key: string;
    label: string;
    icon: LucideIcon;
    description: string;
    apiPath: string;
    countField: string;
};

const categories: IntegrationCategory[] = [
    {
        key: 'imageIntegrations',
        label: 'Image Integrations',
        icon: Image,
        description: 'Container registries, scanners, and image signature verification',
        apiPath: '/v1/imageintegrations',
        countField: 'integrations',
    },
    {
        key: 'notifiers',
        label: 'Notifier Integrations',
        icon: Bell,
        description: 'Slack, email, PagerDuty, and other notification channels',
        apiPath: '/v1/notifiers',
        countField: 'notifiers',
    },
    {
        key: 'backups',
        label: 'Backup Integrations',
        icon: Database,
        description: 'External backup destinations for database exports',
        apiPath: '/v1/externalbackups',
        countField: 'externalBackups',
    },
    {
        key: 'signatureIntegrations',
        label: 'Signature Integrations',
        icon: KeyRound,
        description: 'Cosign and other image signature verification keys',
        apiPath: '/v1/signatureintegrations',
        countField: 'integrations',
    },
];

function IntegrationCategoryCard({ category }: { category: IntegrationCategory }) {
    const { data, isLoading } = useQuery({
        queryKey: ['integrations', category.key, 'count'],
        queryFn: async () => {
            const response = await axios.get(category.apiPath);
            const items = response.data[category.countField] ?? response.data.integrations ?? [];
            return Array.isArray(items) ? items.length : 0;
        },
    });

    return (
        <Card>
            <CardHeader>
                <div className="flex items-center gap-2">
                    <category.icon className="h-4 w-4 text-text-muted" />
                    <CardTitle>{category.label}</CardTitle>
                </div>
                {!isLoading && data !== undefined && (
                    <Badge variant={data > 0 ? 'info' : 'default'}>{data} configured</Badge>
                )}
            </CardHeader>
            <CardContent>
                <p className="text-xs text-text-muted">{category.description}</p>
                {isLoading && <Skeleton className="mt-2 h-4 w-20" />}
            </CardContent>
        </Card>
    );
}

export default function IntegrationsPageCC() {
    return (
        <CommandCenterLayout title="Integrations">
            <div className="p-5">
                <div className="mb-4">
                    <h2 className="text-sm font-600 text-text-primary">Platform Integrations</h2>
                    <p className="mt-1 text-xs text-text-muted">
                        Configure connections to external services for scanning, notifications, and
                        backups.
                    </p>
                </div>

                <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
                    {categories.map((cat) => (
                        <IntegrationCategoryCard key={cat.key} category={cat} />
                    ))}
                </div>
            </div>
        </CommandCenterLayout>
    );
}
