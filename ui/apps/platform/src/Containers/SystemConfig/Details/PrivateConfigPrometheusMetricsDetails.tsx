import type { ReactElement } from 'react';

import type { PrivateConfig } from 'types/config.proto';
import { PrometheusMetricsTabbedCard } from './components/PrometheusMetricsCard';

export type PrivateConfigPrometheusMetricsDetailsProps = {
    privateConfig: PrivateConfig;
};

const PrivateConfigPrometheusMetricsDetails = ({
    privateConfig,
}: PrivateConfigPrometheusMetricsDetailsProps): ReactElement => {
    return <PrometheusMetricsTabbedCard privateConfig={privateConfig} />;
};

export default PrivateConfigPrometheusMetricsDetails;
