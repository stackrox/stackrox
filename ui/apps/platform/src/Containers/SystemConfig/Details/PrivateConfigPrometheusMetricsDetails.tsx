import React, { ReactElement } from 'react';

import { PrivateConfig } from 'types/config.proto';
import { PrometheusMetricsCard } from './components/PrometheusMetricsCard';

export type PrivateConfigPrometheusMetricsDetailsProps = {
    privateConfig: PrivateConfig;
};

const PrivateConfigPrometheusMetricsDetails = ({
    privateConfig,
}: PrivateConfigPrometheusMetricsDetailsProps): ReactElement[] => {
    const imageVulnerabilitiesCfg = privateConfig?.metrics?.imageVulnerabilities;

    return [
        <PrometheusMetricsCard
            category="imageVulnerabilities"
            period={imageVulnerabilitiesCfg?.gatheringPeriodMinutes || 0}
            descriptors={imageVulnerabilitiesCfg?.descriptors}
            title="Image vulnerabilities"
        />,
    ];
};

export default PrivateConfigPrometheusMetricsDetails;
