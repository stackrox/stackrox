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
    const nodeVulnerabilitiesCfg = privateConfig?.metrics?.nodeVulnerabilities;
    const policyViolationsCfg = privateConfig?.metrics?.policyViolations;

    return [
        <PrometheusMetricsCard
            category="imageVulnerabilities"
            key="imageVulnerabilities"
            period={imageVulnerabilitiesCfg?.gatheringPeriodMinutes || 0}
            descriptors={imageVulnerabilitiesCfg?.descriptors}
            title="Image vulnerabilities"
        />,
        <PrometheusMetricsCard
            category="nodeVulnerabilities"
            key="nodeVulnerabilities"
            period={nodeVulnerabilitiesCfg?.gatheringPeriodMinutes || 0}
            descriptors={nodeVulnerabilitiesCfg?.descriptors}
            title="Node vulnerabilities"
        />,
        <PrometheusMetricsCard
            category="policyViolations"
            key="policyViolations"
            period={policyViolationsCfg?.gatheringPeriodMinutes || 0}
            descriptors={policyViolationsCfg?.descriptors}
            title="Policy violations"
        />,
    ];
};

export default PrivateConfigPrometheusMetricsDetails;
