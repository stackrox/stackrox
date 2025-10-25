import type { ReactElement } from 'react';

import type { PrivateConfig, PrometheusMetricsCategory } from 'types/config.proto';
import { PrometheusMetricsCard } from './components/PrometheusMetricsCard';

export type PrivateConfigPrometheusMetricsDetailsProps = {
    privateConfig: PrivateConfig;
};

const PrivateConfigPrometheusMetricsDetails = ({
    privateConfig,
}: PrivateConfigPrometheusMetricsDetailsProps): ReactElement[] => {
    const categoryTitles: Record<PrometheusMetricsCategory, string> = {
        imageVulnerabilities: 'Image vulnerabilities',
        nodeVulnerabilities: 'Node vulnerabilities',
        policyViolations: 'Policy violations',
    };

    return Object.entries(categoryTitles).map(([category, title]) => {
        const config = privateConfig?.metrics?.[category];
        return (
            <PrometheusMetricsCard
                category={category as PrometheusMetricsCategory}
                key={category}
                period={config?.gatheringPeriodMinutes || 0}
                descriptors={config?.descriptors}
                title={title}
            />
        );
    });
};

export default PrivateConfigPrometheusMetricsDetails;
