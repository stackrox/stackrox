import { ReactElement } from 'react';

import { PrivateConfig } from 'types/config.proto';
import { PrometheusMetricsCard } from './components/PrometheusMetricsCard';

export type PrivateConfigPrometheusMetricsDetailsProps = {
    privateConfig: PrivateConfig;
};

const PrivateConfigPrometheusMetricsDetails = ({
    privateConfig,
}: PrivateConfigPrometheusMetricsDetailsProps): ReactElement[] => {
    const imageVulnerabilitiesCfg = privateConfig?.prometheusMetricsConfig?.imageVulnerabilities;

    return [
        PrometheusMetricsCard(
            'imageVulnerabilities',
            imageVulnerabilitiesCfg?.gatheringPeriodMinutes || 0,
            imageVulnerabilitiesCfg?.metrics,
            imageVulnerabilitiesCfg?.filter,
            'Image vulnerabilities'
        ),
    ];
};

export default PrivateConfigPrometheusMetricsDetails;
